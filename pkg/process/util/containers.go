// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package util

import (
	"sync"
	"time"

	model "github.com/DataDog/agent-payload/v5/process"

	"github.com/DataDog/datadog-agent/pkg/tagger"
	"github.com/DataDog/datadog-agent/pkg/tagger/collectors"
	"github.com/DataDog/datadog-agent/pkg/util/containers"
	"github.com/DataDog/datadog-agent/pkg/util/containers/v2/metrics"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/DataDog/datadog-agent/pkg/util/system"
	"github.com/DataDog/datadog-agent/pkg/workloadmeta"
)

const (
	floatNanoseconds float64 = float64(time.Second)
)

// ContainerRateMetrics holds previous values for a container,
// in order to compute rates
type ContainerRateMetrics struct {
	UserCPU            float64
	SystemCPU          float64
	TotalCPU           float64
	IOReadBytes        float64
	IOWriteBytes       float64
	NetworkRcvdBytes   float64
	NetworkSentBytes   float64
	NetworkRcvdPackets float64
	NetworkSentPackets float64
}

// NullContainerRates can be safely used for containers that have no
// previous rate values stored (new containers)
var NullContainerRates = ContainerRateMetrics{
	UserCPU:   -1,
	SystemCPU: -1,
	TotalCPU:  -1,
}

var (
	initContainerProvider   sync.Once
	sharedContainerProvider ContainerProvider
)

// ContainerProvider defines the interface for a container metrics provider
type ContainerProvider interface {
	GetContainers(cacheValidity time.Duration, previousContainers map[string]*ContainerRateMetrics, previousTime, currentTime time.Time) ([]*model.Container, map[string]*ContainerRateMetrics, map[int]string, error)
}

// GetSharedContainerProvider returns a shared ContainerProvider
func GetSharedContainerProvider() ContainerProvider {
	initContainerProvider.Do(func() {
		sharedContainerProvider = NewDefaultContainerProvider()
	})
	return sharedContainerProvider
}

// containerProvider provides data about containers usable by process-agent
type containerProvider struct {
	metricsProvider metrics.Provider
	metadataStore   workloadmeta.Store
	filter          *containers.Filter
}

// NewContainerProvider returns a ContainerProvider instance
func NewContainerProvider(provider metrics.Provider, metadataStore workloadmeta.Store, filter *containers.Filter) ContainerProvider {
	return &containerProvider{
		metricsProvider: provider,
		metadataStore:   metadataStore,
		filter:          filter,
	}
}

// NewDefaultContainerProvider returns a ContainerProvider built with default metrics provider and metadata provider
func NewDefaultContainerProvider() ContainerProvider {
	containerFilter, err := containers.GetSharedMetricFilter()
	if err != nil {
		log.Warnf("Can't get container include/exclude filter, no filtering will be applied: %w", err)
	}

	return NewContainerProvider(metrics.GetProvider(), workloadmeta.GetGlobalStore(), containerFilter)
}

// GetContainers returns containers found on the machine
func (p *containerProvider) GetContainers(cacheValidity time.Duration, previousContainers map[string]*ContainerRateMetrics, previousTime, currentTime time.Time) ([]*model.Container, map[string]*ContainerRateMetrics, map[int]string, error) {
	containersMetadata, err := p.metadataStore.ListContainers()
	if err != nil {
		return nil, nil, nil, err
	}

	hostCPUCount := float64(system.HostCPUCount())
	processContainers := make([]*model.Container, 0)
	rateStats := make(map[string]*ContainerRateMetrics)
	pidToCid := make(map[int]string)
	for _, container := range containersMetadata {
		if !container.State.Running {
			continue
		}

		if p.filter != nil && p.filter.IsExcluded(container.Name, container.Image.Name, container.Labels["io.kubernetes.pod.namespace"]) {
			continue
		}

		if container.Runtime == workloadmeta.ContainerRuntimeGarden && len(container.CollectorTags) == 0 {
			log.Debugf("No tags found for garden container: %s, skipping", container.ID)
			continue
		}

		entityID := containers.BuildTaggerEntityName(container.ID)
		tags, err := tagger.Tag(entityID, collectors.HighCardinality)
		if err != nil {
			log.Debugf("Could not collect tags for container %q, err: %w", container.ID[:12], err)
		}
		tags = append(tags, container.CollectorTags...)

		outPreviousStats := NullContainerRates
		// Name and Image fields exist but are never filled
		processContainer := &model.Container{
			Type:      string(container.Runtime),
			Id:        container.ID,
			Started:   container.State.StartedAt.Unix(),
			Created:   container.State.CreatedAt.Unix(),
			Tags:      tags,
			State:     convertContainerStatus(container.State.Status),
			Health:    convertHealthStatus(container.State.Health),
			Addresses: computeContainerAddrs(container),
		}
		// Always adding container if we have metadata as we do want to report containers without stats
		processContainers = append(processContainers, processContainer)

		// Gathering container & network statistics
		previousContainerRates := previousContainers[container.ID]
		if previousContainerRates == nil {
			previousContainerRates = &NullContainerRates
		}

		collector := p.metricsProvider.GetCollector(string(container.Runtime))
		if collector == nil {
			log.Infof("No metrics collector available for runtime: %s, skipping container: %s", container.Runtime, container.ID)
			continue
		}

		containerStats, err := collector.GetContainerStats(container.ID, cacheValidity)
		if err != nil || containerStats == nil {
			log.Debugf("Container stats for: %v not available through collector %q, err: %w", container, collector.ID(), err)
			// If main container stats are missing, we skip the container
			continue
		}
		computeContainerStats(hostCPUCount, currentTime, previousTime, containerStats, previousContainerRates, &outPreviousStats, processContainer)

		// Building PID to CID mapping for NPM
		if containerStats.PID != nil {
			for _, pid := range containerStats.PID.PIDs {
				pidToCid[pid] = container.ID
			}
		}

		containerNetworkStats, err := collector.GetContainerNetworkStats(container.ID, cacheValidity)
		if err != nil {
			log.Debugf("Container network stats for: %v not available through collector %q, err: %w", container, collector.ID(), err)
		}
		computeContainerNetworkStats(currentTime, previousTime, containerNetworkStats, previousContainerRates, &outPreviousStats, processContainer)

		// Storing previous stats
		rateStats[processContainer.Id] = &outPreviousStats
	}

	return processContainers, rateStats, pidToCid, nil
}

func computeContainerStats(hostCPUCount float64, currentTime, previousTime time.Time, inStats *metrics.ContainerStats, previousStats, outPreviousStats *ContainerRateMetrics, outStats *model.Container) {
	if inStats == nil {
		return
	}

	if inStats.CPU != nil {
		outPreviousStats.TotalCPU = statValue(inStats.CPU.Total, -1)
		outPreviousStats.UserCPU = statValue(inStats.CPU.User, -1)
		outPreviousStats.SystemCPU = statValue(inStats.CPU.System, -1)

		outStats.CpuLimit = float32(statValue(inStats.CPU.Limit, 0))
		outStats.TotalPct = float32(cpuRateValue(outPreviousStats.TotalCPU, previousStats.TotalCPU, hostCPUCount, currentTime, previousTime))
		outStats.UserPct = float32(cpuRateValue(outPreviousStats.UserCPU, previousStats.UserCPU, hostCPUCount, currentTime, previousTime))
		outStats.SystemPct = float32(cpuRateValue(outPreviousStats.SystemCPU, previousStats.SystemCPU, hostCPUCount, currentTime, previousTime))
	}

	if inStats.Memory != nil {
		outStats.MemoryLimit = uint64(statValue(inStats.Memory.Limit, 0))
		outStats.MemCache = uint64(statValue(inStats.Memory.Cache, 0))
		outStats.MemRss = uint64(statValue(inStats.Memory.UsageTotal, 0))
	}

	if inStats.PID != nil {
		outStats.ThreadCount = uint64(statValue(inStats.PID.ThreadCount, 0))
		outStats.ThreadLimit = uint64(statValue(inStats.PID.ThreadLimit, 0))
	}

	if inStats.IO != nil {
		outPreviousStats.IOReadBytes = statValue(inStats.IO.ReadBytes, 0)
		outPreviousStats.IOWriteBytes = statValue(inStats.IO.WriteBytes, 0)

		outStats.Rbps = float32(rateValue(outPreviousStats.IOReadBytes, previousStats.IOReadBytes, currentTime, previousTime))
		outStats.Wbps = float32(rateValue(outPreviousStats.IOWriteBytes, previousStats.IOWriteBytes, currentTime, previousTime))
	}
}

func computeContainerNetworkStats(currentTime, previousTime time.Time, inStats *metrics.ContainerNetworkStats, previousStats, outPreviousStats *ContainerRateMetrics, outStats *model.Container) {
	if inStats == nil {
		return
	}

	outPreviousStats.NetworkRcvdBytes = statValue(inStats.BytesRcvd, 0)
	outPreviousStats.NetworkSentBytes = statValue(inStats.BytesSent, 0)
	outPreviousStats.NetworkRcvdPackets = statValue(inStats.PacketsRcvd, 0)
	outPreviousStats.NetworkSentPackets = statValue(inStats.PacketsSent, 0)

	outStats.NetRcvdBps = float32(rateValue(outPreviousStats.NetworkRcvdBytes, previousStats.NetworkRcvdBytes, currentTime, previousTime))
	outStats.NetSentBps = float32(rateValue(outPreviousStats.NetworkSentBytes, previousStats.NetworkSentBytes, currentTime, previousTime))
	outStats.NetRcvdPs = float32(rateValue(outPreviousStats.NetworkRcvdPackets, previousStats.NetworkRcvdPackets, currentTime, previousTime))
	outStats.NetSentPs = float32(rateValue(outPreviousStats.NetworkSentPackets, previousStats.NetworkSentPackets, currentTime, previousTime))
}

func computeContainerAddrs(container *workloadmeta.Container) []*model.ContainerAddr {
	if len(container.NetworkIPs) == 0 || len(container.Ports) == 0 {
		return nil
	}

	addrs := make([]*model.ContainerAddr, 0, len(container.NetworkIPs)*len(container.Ports))
	for _, containerIP := range container.NetworkIPs {
		for _, port := range container.Ports {
			addrs = append(addrs, &model.ContainerAddr{
				Ip:       containerIP,
				Port:     int32(port.Port),
				Protocol: model.ConnectionType(model.ConnectionType_value[port.Protocol]),
			})
		}
	}
	return addrs
}

func convertHealthStatus(health workloadmeta.ContainerHealth) model.ContainerHealth {
	// This works because unknown keys will return 0 (which is unknown)
	return model.ContainerHealth(model.ContainerHealth_value[string(health)])
}

func convertContainerStatus(status workloadmeta.ContainerStatus) model.ContainerState {
	if status == workloadmeta.ContainerStatusStopped {
		return model.ContainerState_exited
	}

	return model.ContainerState(model.ContainerState_value[string(status)])
}

func statValue(val *float64, def float64) float64 {
	if val != nil {
		return *val
	}

	return def
}

func cpuRateValue(current, previous, hostCPUCount float64, currentTs, previousTs time.Time) float64 {
	if current == -1 || previous == -1 {
		return -1
	}

	return 100 * rateValue(current, previous, currentTs, previousTs) / floatNanoseconds
}

func rateValue(current, previous float64, currentTs, previousTs time.Time) float64 {
	if previousTs.IsZero() {
		return 0
	}

	valueDiff := current - previous
	if valueDiff < 0 {
		valueDiff = 0
	}

	timeDiff := currentTs.Sub(previousTs).Seconds()
	return valueDiff / timeDiff
}

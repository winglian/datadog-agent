package report

import (
	json "encoding/json"
	"sort"
	"strconv"
	"time"

	"github.com/DataDog/datadog-agent/pkg/epforwarder"
	"github.com/DataDog/datadog-agent/pkg/util"
	"github.com/DataDog/datadog-agent/pkg/util/log"

	"github.com/DataDog/datadog-agent/pkg/collector/corechecks/snmp/checkconfig"
	"github.com/DataDog/datadog-agent/pkg/collector/corechecks/snmp/common"
	"github.com/DataDog/datadog-agent/pkg/collector/corechecks/snmp/metadata"
	"github.com/DataDog/datadog-agent/pkg/collector/corechecks/snmp/valuestore"
)

// interfaceNameTagKey matches the `interface` tag used in `_generic-if.yaml` for ifName
var interfaceNameTagKey = "interface"

// ReportNetworkDeviceMetadata reports device metadata
func (ms *MetricSender) ReportNetworkDeviceMetadata(config *checkconfig.CheckConfig, store *valuestore.ResultValueStore, origTags []string, collectTime time.Time, deviceStatus metadata.DeviceStatus) {
	tags := common.CopyStrings(origTags)
	tags = util.SortUniqInPlace(tags)

	metadataStore := buildMetadata(config.Metadata, store)

	device := buildNetworkDeviceMetadata(config.DeviceID, config.DeviceIDTags, config, metadataStore, tags, deviceStatus)

	interfaces := buildNetworkInterfacesMetadata(config.DeviceID, metadataStore)

	metadataPayloads := batchPayloads(config.Namespace, config.ResolvedSubnetName, collectTime, metadata.PayloadMetadataBatchSize, device, interfaces)

	for _, payload := range metadataPayloads {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			log.Errorf("Error marshalling device metadata: %s", err)
			return
		}
		ms.sender.EventPlatformEvent(string(payloadBytes), epforwarder.EventTypeNetworkDevicesMetadata)
	}
}

func buildMetadata(metadataConfigs checkconfig.MetadataConfig, values *valuestore.ResultValueStore) *metadata.Store {
	metadataStore := metadata.NewMetadataStore()

	for resourceName, metadataConfig := range metadataConfigs {
		for fieldName, symbol := range metadataConfig.Fields {
			if resourceName == "device" {
				value, err := values.GetScalarValue(symbol.OID)
				if err != nil {
					log.Debugf("report scalar: error getting scalar value: %v", err)
					continue
				}
				metadataStore.AddScalarValue(resourceName+"."+fieldName, value)
			} else {
				metricValues, err := values.GetColumnValues(symbol.OID)
				if err != nil {
					continue
				}
				for fullIndex, value := range metricValues {
					metadataStore.AddColumnValue(resourceName+"."+fieldName, fullIndex, value)
				}
			}
		}
		if indexOid, ok := metadata.ResourceIndex[resourceName]; ok {
			metricValues, err := values.GetColumnValues(indexOid)
			if err != nil {
				continue
			}
			for fullIndex := range metricValues {
				// TODO: TEST ME
				tags := metadataConfig.Tags.GetTags(fullIndex, values)
				metadataStore.AddTags(resourceName, fullIndex, tags)
				idTags := metadataConfig.IdTags.GetTags(fullIndex, values)
				metadataStore.AddIdTags(resourceName, fullIndex, idTags)
			}
		}
	}
	return metadataStore
}

func buildNetworkDeviceMetadata(deviceID string, idTags []string, config *checkconfig.CheckConfig, store *metadata.Store, tags []string, deviceStatus metadata.DeviceStatus) metadata.DeviceMetadata {
	var vendor, sysName, sysDescr, sysObjectID, serialNumber string
	if store != nil {
		sysName = store.GetScalarAsString("device.name")
		sysDescr = store.GetScalarAsString("device.description")
		sysObjectID = store.GetScalarAsString("device.sys_object_id")
		serialNumber = store.GetScalarAsString("device.serial_number")
	}

	if config.ProfileDef != nil {
		vendor = config.ProfileDef.Device.Vendor
	}

	return metadata.DeviceMetadata{
		ID:           deviceID,
		IDTags:       idTags,
		Name:         sysName,
		Description:  sysDescr,
		IPAddress:    config.IPAddress,
		SysObjectID:  sysObjectID,
		Profile:      config.Profile,
		Vendor:       vendor,
		Tags:         tags,
		Subnet:       config.ResolvedSubnetName,
		Status:       deviceStatus,
		SerialNumber: serialNumber,
	}
}

func buildNetworkInterfacesMetadata(deviceID string, store *metadata.Store) []metadata.InterfaceMetadata {
	if store == nil {
		// it's expected that the value store is nil if we can't reach the device
		// in that case, we just return a nil slice.
		return nil
	}
	indexes := store.GetColumnIndexes("interface.name")
	sort.Strings(indexes)
	var interfaces []metadata.InterfaceMetadata
	for _, strIndex := range indexes {
		index, err := strconv.Atoi(strIndex)
		if err != nil {
			log.Warnf("interface metadata: invalid index: %s", index)
			continue
		}

		ifIdTags := store.GetIdTags("interface", strIndex)
		ifTags := store.GetTags("interface", strIndex)

		name := store.GetColumnAsString("interface.name", strIndex)
		networkInterface := metadata.InterfaceMetadata{
			DeviceID:    deviceID,
			Index:       int32(index),
			Name:        name,
			Alias:       store.GetColumnAsString("interface.alias", strIndex),
			Description: store.GetColumnAsString("interface.description", strIndex),
			MacAddress:  store.GetColumnAsString("interface.mac_address", strIndex),
			AdminStatus: int32(store.GetColumnAsFloat("interface.admin_status", strIndex)),
			OperStatus:  int32(store.GetColumnAsFloat("interface.oper_status", strIndex)),
			Tags:        ifTags,
			IDTags:      ifIdTags,
		}
		interfaces = append(interfaces, networkInterface)
	}
	return interfaces
}

func batchPayloads(namespace string, subnet string, collectTime time.Time, batchSize int, device metadata.DeviceMetadata, interfaces []metadata.InterfaceMetadata) []metadata.NetworkDevicesMetadata {
	var payloads []metadata.NetworkDevicesMetadata
	var resourceCount int
	payload := metadata.NetworkDevicesMetadata{
		Devices: []metadata.DeviceMetadata{
			device,
		},
		Subnet:           subnet,
		Namespace:        namespace,
		CollectTimestamp: collectTime.Unix(),
	}
	resourceCount++

	for _, interfaceMetadata := range interfaces {
		if resourceCount == batchSize {
			payloads = append(payloads, payload)
			payload = metadata.NetworkDevicesMetadata{
				Subnet:           subnet,
				Namespace:        namespace,
				CollectTimestamp: collectTime.Unix(),
			}
			resourceCount = 0
		}
		resourceCount++
		payload.Interfaces = append(payload.Interfaces, interfaceMetadata)
	}

	payloads = append(payloads, payload)
	return payloads
}

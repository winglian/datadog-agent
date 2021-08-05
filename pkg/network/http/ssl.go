// +build linux_bpf

package http

import (
	"fmt"
	"math"

	"github.com/DataDog/datadog-agent/pkg/network/ebpf/probes"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/DataDog/ebpf"
	"github.com/DataDog/ebpf/manager"
	"golang.org/x/sys/unix"
)

var libSSLProbes = []string{
	"uprobe/SSL_set_bio",
	"uprobe/SSL_set_fd",
	"uprobe/SSL_read",
	"uretprobe/SSL_read",
	"uprobe/SSL_write",
	"uprobe/SSL_shutdown",
}

var libCryptoProbes = []string{
	"uprobe/BIO_new_socket",
	"uretprobe/BIO_new_socket",
}

// these maps are shared among the "main" HTTP (socket-filter) program and
// the (uprobe-based) OpenSSL programs
var sharedHTTPMaps = []probes.BPFMapName{
	probes.HttpInFlightMap,
	probes.HttpBatchesMap,
	probes.HttpBatchStateMap,
}

// sslProgram encapsulates the uprobe management for one specific OpenSSL library "instance"
// TODO: replace `Manager` by something lighter so we can avoid things such as parsing
// the eBPF ELF repeatedly
type sslProgram struct {
	mainProgram *ebpfProgram
	mgr         *manager.Manager
	offsets     []manager.ConstantEditor
	sockFDMap   *ebpf.Map
	sslPath     string
	cryptoPath  string
}

var _ subprogram = &sslProgram{}

func createSSLPrograms(mainProgram *ebpfProgram, offsets []manager.ConstantEditor, sockFD *ebpf.Map) []subprogram {
	var subprograms []subprogram
	for _, sslCryptoPair := range findOpenSSLLibraries(mainProgram.cfg.ProcRoot) {
		sslProg, err := newSSLProgram(mainProgram, offsets, sockFD, sslCryptoPair[0], sslCryptoPair[1])
		if err != nil {
			log.Errorf("error creating SSL program: %s", err)
			continue
		}

		subprograms = append(subprograms, sslProg)
	}

	return subprograms
}

func newSSLProgram(
	mainProgram *ebpfProgram,
	offsets []manager.ConstantEditor,
	sockFD *ebpf.Map,
	sslPath, cryptoPath string,
) (*sslProgram, error) {
	if sockFD == nil {
		return nil, fmt.Errorf("sockFD map not provided")
	}
	if sslPath == "" {
		return nil, fmt.Errorf("path to libssl not provided")
	}

	var probes []*manager.Probe
	for _, sec := range libSSLProbes {
		probes = append(probes, &manager.Probe{
			Section:    sec,
			BinaryPath: sslPath,
		})
	}

	// libcrypto probes are optional
	if cryptoPath != "" {
		for _, sec := range libCryptoProbes {
			probes = append(probes, &manager.Probe{
				Section:    sec,
				BinaryPath: cryptoPath,
			})
		}
	}

	return &sslProgram{
		mainProgram: mainProgram,
		mgr:         &manager.Manager{Probes: probes},
		offsets:     offsets,
		sockFDMap:   sockFD,
		sslPath:     sslPath,
		cryptoPath:  cryptoPath,
	}, nil
}

func (p *sslProgram) Init() error {
	var selectors []manager.ProbesSelector

	// Determine which probes we want to activate
	toActivate := append([]string(nil), libSSLProbes...)
	if p.cryptoPath != "" {
		toActivate = append(toActivate, libCryptoProbes...)
	}

	for _, sec := range toActivate {
		selectors = append(selectors, &manager.ProbeSelector{
			ProbeIdentificationPair: manager.ProbeIdentificationPair{
				Section: sec,
			},
		})
	}

	// set up shared maps
	// * the sockFD is shared with the network-tracer program;
	// * all other HTTP-specific maps are shared with the "core" HTTP program;
	editors := map[string]*ebpf.Map{
		"sock_by_pid_fd": p.sockFDMap,
	}

	for _, mapName := range sharedHTTPMaps {
		name := string(mapName)
		m, _, _ := p.mainProgram.GetMap(name)
		if m == nil {
			return fmt.Errorf("couldn't retrieve map: %s", m)
		}
		editors[name] = m
	}

	log.Debugf("https tracing enabled. ssl=%s crypto=%s", p.sslPath, p.cryptoPath)
	return p.mgr.InitWithOptions(
		p.mainProgram.bytecode,
		manager.Options{
			RLimit: &unix.Rlimit{
				Cur: math.MaxUint64,
				Max: math.MaxUint64,
			},
			ActivatedProbes: selectors,
			ConstantEditors: p.offsets,
			MapEditors:      editors,
		},
	)
}

func (p *sslProgram) Start() error {
	return p.mgr.Start()
}

func (p *sslProgram) Close() error {
	return p.mgr.Stop(manager.CleanAll)
}

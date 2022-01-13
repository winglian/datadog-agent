/*
 * // Unless explicitly stated otherwise all files in this repository are licensed
 * // under the Apache License Version 2.0.
 * // This product includes software developed at Datadog (https://www.datadoghq.com/).
 * // Copyright 2016-present Datadog, Inc.
 */

package netlink

import (
	"context"

	"golang.org/x/sys/unix"
)

type DebugConntrackEntry struct {
	Proto  string
	Family string
	Origin DebugConntrackTuple
	Reply  DebugConntrackTuple
}

type DebugConntrackTuple struct {
	Src DebugConntrackAddress
	Dst DebugConntrackAddress
}

type DebugConntrackAddress struct {
	IP   string
	Port uint16
}

func (ctr *realConntracker) DumpTable(ctx context.Context) (map[uint32][]DebugConntrackEntry, error) {
	table := make(map[uint32][]DebugConntrackEntry)
	keys := ctr.cache.cache.Keys()
	if len(keys) == 0 {
		return table, nil
	}

	// netlink conntracker does not store netns values
	ns := uint32(0)
	table[ns] = []DebugConntrackEntry{}

	for _, k := range keys {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		ck, ok := k.(connKey)
		if !ok {
			continue
		}
		v, ok := ctr.cache.cache.Peek(ck)
		if !ok {
			continue
		}
		te, ok := v.(*translationEntry)
		if !ok {
			continue
		}

		table[ns] = append(table[ns], DebugConntrackEntry{
			Family: ck.transport.String(),
			Origin: DebugConntrackTuple{
				Src: DebugConntrackAddress{
					IP:   ck.srcIP.String(),
					Port: ck.srcPort,
				},
				Dst: DebugConntrackAddress{
					IP:   ck.dstIP.String(),
					Port: ck.dstPort,
				},
			},
			Reply: DebugConntrackTuple{
				Src: DebugConntrackAddress{
					IP:   te.ReplSrcIP.String(),
					Port: te.ReplSrcPort,
				},
				Dst: DebugConntrackAddress{
					IP:   te.ReplDstIP.String(),
					Port: te.ReplDstPort,
				},
			},
		})
	}
	return table, nil
}

func DumpHostTable(ctx context.Context, procRoot string) (map[uint32][]DebugConntrackEntry, error) {
	consumer := NewConsumer(procRoot, -1, true)
	decoder := NewDecoder()
	defer consumer.Stop()

	table := make(map[uint32][]DebugConntrackEntry)

	for _, family := range []uint8{unix.AF_INET, unix.AF_INET6} {
		events, err := consumer.DumpTable(family)
		if err != nil {
			return nil, err
		}

		fstr := "v4"
		if family == unix.AF_INET6 {
			fstr = "v6"
		}

	dumploop:
		for {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case ev, ok := <-events:
				if !ok {
					break dumploop
				}
				conns := decoder.DecodeAndReleaseEvent(ev)
				for _, c := range conns {
					if !IsNAT(c) {
						continue
					}

					ns := uint32(c.NetNS)
					_, ok := table[ns]
					if !ok {
						table[ns] = []DebugConntrackEntry{}
					}

					src, sok := formatKey(c.Origin)
					dst, dok := formatKey(c.Reply)
					if !sok || !dok {
						continue
					}

					table[ns] = append(table[ns], DebugConntrackEntry{
						Family: fstr,
						Proto:  src.transport.String(),
						Origin: DebugConntrackTuple{
							Src: DebugConntrackAddress{
								IP:   src.srcIP.String(),
								Port: src.srcPort,
							},
							Dst: DebugConntrackAddress{
								IP:   src.dstIP.String(),
								Port: src.dstPort,
							},
						},
						Reply: DebugConntrackTuple{
							Src: DebugConntrackAddress{
								IP:   dst.srcIP.String(),
								Port: dst.srcPort,
							},
							Dst: DebugConntrackAddress{
								IP:   dst.dstIP.String(),
								Port: dst.dstPort,
							},
						},
					})
				}
			}
		}
	}
	return table, nil
}

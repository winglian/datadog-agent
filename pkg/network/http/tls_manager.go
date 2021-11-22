// +build linux_bpf

package http

import (
	"unsafe"

	"fmt"

	"github.com/DataDog/ebpf"
	"github.com/google/gopacket/layers"
)

/*
#include "../ebpf/c/tls-types.h"
*/
import "C"

func (r tlsBufferRing) Size() int {
	if r.full != 0 {
		return int(C.TLS_BUFFER_RING_SIZE)
	}
	s := int(r.h) - int(r.t)
	if s < 0 {
		return int(C.TLS_BUFFER_RING_SIZE) + s
	}
	return s
}

func (r tlsBufferRing) Get(i int) int {
	return (int(r.t) + i) & int(C.TLS_BUFFER_RING_SIZE_MASK)
}

type tlsManager struct {
	cnx           map[C.conn_tuple_t][]byte
	bufferMap     *ebpf.Map
	bufferRingMap *ebpf.Map
}

func newTLSManager(bufferMap, bufferRingMap *ebpf.Map) *tlsManager {
	for key := uint32(0); key < bufferMap.ABI().MaxEntries; key++ {
		buf := new(tlsBuffer)
		bufferMap.Put(unsafe.Pointer(&key), unsafe.Pointer(buf))
	}

	return &tlsManager{
		cnx:           make(map[C.conn_tuple_t][]byte),
		bufferMap:     bufferMap,
		bufferRingMap: bufferRingMap,
	}
}

func (t *tlsManager) flushRing(r tlsBufferRing) error {
	r.t = r.h
	r.full = 0
	k := uint32(0)
	if err := t.bufferRingMap.Put(unsafe.Pointer(&k), unsafe.Pointer(&r)); err != nil {
		return fmt.Errorf("ring put error %w", err)
	}
	return nil
}

func (t *tlsManager) readPackets(r tlsBufferRing) (buffers []tlsBuffer, err error) {
	defer t.flushRing(r)

	for i := 0; i < r.Size(); i++ {
		var tlsBuf tlsBuffer
		k := uint32(r.Get(i))
		if err := t.bufferMap.Lookup(unsafe.Pointer(&k), unsafe.Pointer(&tlsBuf)); err != nil {
			return buffers, fmt.Errorf("buffer lookup error %w", err)
		}
		buffers = append(buffers, tlsBuf)
		fmt.Printf("tup %+v %d len\n", tlsBuf.tup, tlsBuf.len)
	}
	return buffers, nil
}

type TLSDecodeFeedback struct {
	t     *tlsManager
	tuple *C.conn_tuple_t
}

func (cdf TLSDecodeFeedback) SetTruncated() {
	fmt.Printf("SetTruncated() %+v\n", cdf.tuple)
}

func (t *tlsManager) Poll() error {
	var r tlsBufferRing
	k := uint32(0)
	if err := t.bufferRingMap.Lookup(unsafe.Pointer(&k), unsafe.Pointer(&r)); err != nil {
		return fmt.Errorf("ring lookup error %w", err)
	}
	fmt.Printf("TLS Poll() ring %+v\n", r)
	if r.Size() == 0 {
		return nil
	}

	buffers, err := t.readPackets(r)
	if err != nil {
		return fmt.Errorf("read packets error %w", err)
	}

	for _, b := range buffers {
		d := C.GoBytes(unsafe.Pointer(&b.buffer[0]), C.int(b.len))
		t.cnx[b.tup] = append(t.cnx[b.tup], d...)
	}

	for tuple, data := range t.cnx {
		//		fmt.Printf("tup %+v\n", c)
		protoTLS := &layers.TLS{}
		err := protoTLS.DecodeFromBytes(data, TLSDecodeFeedback{t, &tuple})
		if err != nil {
			fmt.Println("err", err)
		}
		fmt.Printf("TLS %+v\n", protoTLS.Handshake)
	}
	t.cnx = make(map[C.conn_tuple_t][]byte)

	return nil
}

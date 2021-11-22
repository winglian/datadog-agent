#ifndef __TLS_H
#define __TLS_H

#include "tracer.h"
#include "tls-types.h"
#include "tls-maps.h"
#include "tls-ring.h"

#include <uapi/linux/ptrace.h>

static __always_inline int isTLS(tls_header_t *hdr, struct __sk_buff* skb, u32 offset) {
    if (skb->len - offset < TLS_HEADER_SIZE) {
        return 0;
    }
    __u8 app = load_byte(skb, offset);
    if ((app != TLS_HANDSHAKE) &&
        (app != TLS_APPLICATION_DATA)) {
            return 0;
    }
    hdr->app = app;
    
    __u16 version = load_half(skb, offset + 1);
    if ((version != TLS_VERSION10) &&
        (version != TLS_VERSION11) &&
        (version != TLS_VERSION12)) {
            return 0;
    }
    hdr->version = version;
    
    __u16 length = load_half(skb, offset + 3);
    hdr->length = length;
    __u16 skblen = skb->len - offset - TLS_HEADER_SIZE;
    if (skblen < length) {
        return 0;
    }
    if (skblen == length) {
        return 1;
    }
//    return skblen - length;

    /*
    log_debug("app layer   %x\n", app);
    log_debug("app version %x\n", version);
    log_debug("app length  %d\n", length);
    log_debug("app skblen  %d\n", skb->len - offset - TLS_HEADER_SIZE);
    */
    return 1;
}

static __always_inline int pkt_copy(char * buffer, struct __sk_buff* skb, u32 offset) {
    int i;
#pragma unroll
    for (i = 0; i < (skb->len - offset) && i < TLS_BUFFER_SIZE; i++) {
        buffer[i] = load_byte(skb, offset + i);
    }
    return i;
}

static __always_inline int stream_pkt(struct __sk_buff* skb, skb_info_t *skb_info) {
    __u32 k = 0;
    tls_buffer_ring_t new_entry = { 0 };
    bpf_map_update_elem(&tls_buffer_ring, &k, &new_entry, BPF_NOEXIST);
    tls_buffer_ring_t *r = bpf_map_lookup_elem(&tls_buffer_ring, &k);
    if (r == NULL || r->full) { // add telemetry on full here
        return 0;
    }
    k = ring_add(r);
    tls_buffer_t *buffer = bpf_map_lookup_elem(&tls_buffer, &k);
    if (buffer == NULL) {
        return 0;
    }
    __builtin_memcpy(&buffer->tup, &skb_info->tup, sizeof(conn_tuple_t));
    buffer->len = pkt_copy(&buffer->buffer[0], skb, skb_info->data_off);
    log_debug("tls stream pkt len %d ====", skb->len - skb_info->data_off);
    return 1;
}

static __always_inline int tls_process(struct __sk_buff* skb, skb_info_t *skb_info) {
    if (skb_info->tcp_flags & TCPHDR_FIN) {
        bpf_map_delete_elem(&tls_in_flight, &skb_info->tup);
        return 0;
    }
    if (skb->len - skb_info->data_off == 0) {
        return 0;
    }

    tls_transaction_t *tls = NULL;
    tls_transaction_t new_entry = { 0 };
    bpf_map_update_elem(&tls_in_flight, &skb_info->tup, &new_entry, BPF_NOEXIST);
    tls = bpf_map_lookup_elem(&tls_in_flight, &skb_info->tup);
    if (tls == NULL) {
        return 0;
    }
    /* cnx classified */
    if ((tls->isTLS == 1 && tls->handshake_done == 1)
        || tls->packets > TLS_MAX_PACKET_CLASSIFIER) {
        log_debug("tls classifieddd12");
        return 0;
    }
    tls->packets++;

    tls_header_t tls_hdr;
    if (!isTLS(&tls_hdr, skb, skb_info->data_off)) {
        if (tls->isTLS == 1 && tls->handshake_done == 0) {
            if(!stream_pkt(skb, skb_info)) {
                return 0;
            }
        }
        return 0;
    }
    if (tls_hdr.app == TLS_APPLICATION_DATA) {
        tls->handshake_done = 1;
    }
    if (tls->handshake_done == 0) {
        if(!stream_pkt(skb, skb_info)) {
            return 0;
        }
    }

    __builtin_memcpy(&tls->tup, &skb_info->tup, sizeof(conn_tuple_t));
    tls->isTLS = 1;

    return 0;
}

#endif

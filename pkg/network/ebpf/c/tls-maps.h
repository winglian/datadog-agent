#ifndef __TLS_MAPS_H
#define __TLS_MAPS_H

#include "tracer.h"
#include "bpf_helpers.h"
#include "tls-types.h"

/* This map is used to keep track of in-flight TLS transactions for each TCP connection */
struct bpf_map_def SEC("maps/tls_in_flight") tls_in_flight = {
    .type = BPF_MAP_TYPE_HASH,
    .key_size = sizeof(conn_tuple_t),
    .value_size = sizeof(tls_transaction_t),
    .max_entries = 1, // This will get overridden at runtime using max_tracked_connections
    .pinning = 0,
    .namespace = "",
};

/* This map is used to keep TLS handshake buffer */
struct bpf_map_def SEC("maps/tls_buffer") tls_buffer = {
    .type = BPF_MAP_TYPE_HASH,
    .key_size = sizeof(__u32),
    .value_size = sizeof(tls_buffer_t),
    .max_entries = TLS_BUFFER_RING_SIZE,
    .pinning = 0,
    .namespace = "",
};

/* This map is used to keep TLS handshake buffer */
struct bpf_map_def SEC("maps/tls_buffer_ring") tls_buffer_ring = {
    .type = BPF_MAP_TYPE_HASH,
    .key_size = sizeof(__u32),
    .value_size = sizeof(tls_buffer_ring_t),
    .max_entries = 1,
    .pinning = 0,
    .namespace = "",
};

/* This map used for send TLS hello handsharke to the userspace  */
struct bpf_map_def SEC("maps/tls_handshake") tls_handshake = {
    .type = BPF_MAP_TYPE_PERF_EVENT_ARRAY,
    .key_size = sizeof(__u32),
    .value_size = sizeof(__u32),
    .max_entries = 0, // This will get overridden at runtime
    .pinning = 0,
    .namespace = "",
};

#endif

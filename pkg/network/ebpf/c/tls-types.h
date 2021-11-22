#ifndef __TLS_TYPES_H
#define __TLS_TYPES_H

#include "tracer.h"

typedef struct {
    __u8 app;
    __u16 version;
    __u16 length;
} tls_header_t;
    
#define TLS_HEADER_SIZE 5

#define SSL_VERSION20 0x0200
#define SSL_VERSION30 0x0300
#define TLS_VERSION10 0x0301
#define TLS_VERSION11 0x0302
#define TLS_VERSION12 0x0303
#define TLS_VERSION13 0x0304

#define TLS_ALERT 0x15
#define TLS_HANDSHAKE 0x16
#define TLS_APPLICATION_DATA 0x17

#define TLS_MAX_PACKET_CLASSIFIER 10

/* packets here is used as guard for miss classification */
typedef struct {
    conn_tuple_t tup;
    __u8 isTLS;
    __u8 packets;
    __u8 handshake_done;
    __u8 __padding;
} tls_transaction_t;

#define TLS_BUFFER_SIZE 3000
typedef struct {
    conn_tuple_t tup;
    __u16 len;
    char buffer[TLS_BUFFER_SIZE];
} tls_buffer_t;

#define TLS_BUFFER_RING_SIZE 128
#define TLS_BUFFER_RING_SIZE_MASK (TLS_BUFFER_RING_SIZE-1)
typedef struct {
    __u32 h;
    __u32 t;
    __u32 full;
} tls_buffer_ring_t;

#endif

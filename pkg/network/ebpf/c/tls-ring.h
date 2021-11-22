#ifndef __TLS_RING_H
#define __TLS_RING_H

#include "tls-types.h"
#include "tracer.h"

/* r->full need to be check before calling */
static __always_inline int ring_add(tls_buffer_ring_t *r) {
    int cur = r->h;
    r->h = (r->h + 1) & TLS_BUFFER_RING_SIZE_MASK;
    r->full = (r->h == r->t);
    return cur;
}

#endif

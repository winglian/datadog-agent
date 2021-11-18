#ifndef _NETWORK_PARSER_H_
#define _NETWORK_PARSER_H_

#define DNS_PORT 53
#define DNS_MAX_LENGTH 256
#define DNS_A_RECORD 1
#define DNS_COMPRESSION_FLAG 3

struct cursor {
	void *pos;
	void *end;
};

__attribute__((always_inline)) void xdp_cursor_init(struct cursor *c, struct xdp_md *ctx) {
	c->end = (void *)(long)ctx->data_end;
	c->pos = (void *)(long)ctx->data;
}

__attribute__((always_inline)) void tc_cursor_init(struct cursor *c, struct __sk_buff *skb) {
	c->end = (void *)(long)skb->data_end;
	c->pos = (void *)(long)skb->data;
}

#define PARSE_FUNC(STRUCT)			                                                 \
__attribute__((always_inline)) struct STRUCT *parse_ ## STRUCT (struct cursor *c) {	 \
	struct STRUCT *ret = c->pos;			                                         \
	if (c->pos + sizeof(struct STRUCT) > c->end)	                                 \
		return 0;				                                                     \
	c->pos += sizeof(struct STRUCT);		                                         \
	return ret;					                                                     \
}

PARSE_FUNC(ethhdr)
PARSE_FUNC(iphdr)
PARSE_FUNC(udphdr)
PARSE_FUNC(tcphdr)

struct pkt_ctx_t {
    struct cursor *c;
    struct ethhdr *eth;
    struct iphdr *ipv4;
    struct tcphdr *tcp;
    struct udphdr *udp;
};

#endif

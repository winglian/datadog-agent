#ifndef _DNS_H_
#define _DNS_H_

struct dnshdr {
    uint16_t id;
    union {
        struct {
            uint8_t  rd     : 1;
            uint8_t  tc     : 1;
            uint8_t  aa     : 1;
            uint8_t  opcode : 4;
            uint8_t  qr     : 1;

            uint8_t  rcode  : 4;
            uint8_t  cd     : 1;
            uint8_t  ad     : 1;
            uint8_t  z      : 1;
            uint8_t  ra     : 1;
        }        as_bits_and_pieces;
        uint16_t as_value;
    } flags;
    uint16_t qdcount;
    uint16_t ancount;
    uint16_t nscount;
    uint16_t arcount;
};

PARSE_FUNC(dnshdr)

struct dns_name_t {
    char name[DNS_MAX_LENGTH];
};

struct bpf_map_def SEC("maps/dns_name_gen") dns_name_gen = {
    .type = BPF_MAP_TYPE_PERCPU_ARRAY,
    .key_size = sizeof(u32),
    .value_size = sizeof(struct dns_name_t),
    .max_entries = 1,
    .pinning = 0,
    .namespace = "",
};

struct bpf_map_def SEC("maps/dns_table") dns_table = {
    .type = BPF_MAP_TYPE_LRU_HASH,
    .key_size = sizeof(struct dns_name_t),
    .value_size = sizeof(u32),
    .max_entries = 512,
    .pinning = 0,
    .namespace = "",
};

struct dns_request_cache_key_t {
    u32 saddr;
    u32 daddr;
    u16 source_port;
    u16 dest_port;
    u16 request_id;
    u16 padding;
};

struct dns_request_cache_t {
    u32 name_length;
    u32 ip;
};

struct bpf_map_def SEC("maps/dns_request_cache") dns_request_cache = {
    .type = BPF_MAP_TYPE_LRU_HASH,
    .key_size = sizeof(struct dns_request_cache_key_t),
    .value_size = sizeof(struct dns_request_cache_t),
    .max_entries = 1024,
    .pinning = 0,
    .namespace = "",
};

__attribute__((always_inline)) int handle_dns_req(struct __sk_buff *skb, struct cursor *c, struct pkt_ctx_t *pkt) {
    struct dnshdr header = {};
    u32 offset = ((u32)(long)c->pos - skb->data);

    if (bpf_skb_load_bytes(skb, offset, &header, sizeof(header)) < 0) {
        return TC_ACT_OK;
    }
    offset += sizeof(header);

    u32 qname_length = 0;
    u8 end_of_name = 0;
    u32 key_gen = 0;
    struct dns_name_t *name = bpf_map_lookup_elem(&dns_name_gen, &key_gen);
    if (name == NULL)
        return TC_ACT_OK;

    #pragma unroll
    for (int i = 0; i < DNS_MAX_LENGTH; i++) {
        if (end_of_name) {
            name->name[i] = 0;
            continue;
        }

        if (bpf_skb_load_bytes(skb, offset, &name->name[i], sizeof(u8)) < 0) {
            return TC_ACT_OK;
        }

        qname_length += 1;
        offset += 1;

        if (name->name[i] == 0) {
            end_of_name = 1;
        }
    }

    // Handle qtype
    u16 qtype = 0;
    if (bpf_skb_load_bytes(skb, offset, &qtype, sizeof(u16)) < 0) {
        return TC_ACT_OK;
    }
    qtype = htons(qtype);
    offset += sizeof(u16);

    // Handle qclass
    u16 qclass = 0;
    if (bpf_skb_load_bytes(skb, offset, &qclass, sizeof(u16)) < 0) {
        return TC_ACT_OK;
    }
    qclass = htons(qclass);
    offset += sizeof(u16);

    // Lookup DNS name and cache DNS request id <-> IP
    u32 *ip = bpf_map_lookup_elem(&dns_table, name->name);
    if (ip == NULL)
        return TC_ACT_OK;

    struct dns_request_cache_key_t key = {
        .saddr = pkt->ipv4->saddr,
        .daddr = pkt->ipv4->daddr,
        .source_port = pkt->udp->source,
        .dest_port = pkt->udp->dest,
        .request_id = header.id,
    };
    struct dns_request_cache_t entry = {
        .name_length = qname_length,
        .ip = *ip,
    };
    bpf_map_update_elem(&dns_request_cache, &key, &entry, BPF_ANY);

    return TC_ACT_OK;
}

#endif

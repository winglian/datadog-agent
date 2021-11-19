#ifndef _DNS_H_
#define _DNS_H_

struct bpf_map_def SEC("maps/pid_dns_eval_prog_ids") pid_dns_eval_prog_ids = {
    .type = BPF_MAP_TYPE_HASH,
    .key_size = sizeof(u32),
    .value_size = sizeof(u32),
    .max_entries = 1024,
};

struct bpf_map_def SEC("maps/dns_eval_progs") dns_eval_progs = {
    .type = BPF_MAP_TYPE_PROG_ARRAY,
    .key_size = sizeof(u32),
    .value_size = sizeof(u32),
    .max_entries = 100,
};

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

struct dns_request_flow_t {
    u32 saddr;
    u32 daddr;
    u16 source_port;
    u16 dest_port;
};

struct bpf_map_def SEC("maps/dns_request_cache") dns_request_cache = {
    .type = BPF_MAP_TYPE_LRU_HASH,
    .key_size = sizeof(struct dns_request_flow_t),
    .value_size = sizeof(struct dns_name_t),
    .max_entries = 1024,
    .pinning = 0,
    .namespace = "",
};

__attribute__((always_inline)) void fill_dns_request_flow(struct pkt_ctx_t *pkt, struct dns_request_flow_t *key) {
    key->saddr = pkt->ipv4->saddr;
    key->daddr = pkt->ipv4->daddr;
    key->source_port = pkt->udp->source;
    key->dest_port = pkt->udp->dest;
    return;
}

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

    // Resolve pid
    struct flow_pid_key_t flow_key = {};
    flow_key.addr[0] = pkt->ipv4->saddr;
    flow_key.port = pkt->udp->source;

    struct flow_pid_value_t *pid = bpf_map_lookup_elem(&flow_pid, &flow_key);
    if (!pid) {
        // Try with IP set to 0.0.0.0
        flow_key.addr[0] = 0;
        flow_key.addr[1] = 0;
        pid = bpf_map_lookup_elem(&flow_pid, &flow_key);
        if (!pid) {
            // this process isn't tracked, ignore packet
            return TC_ACT_OK;
        }
    }

    // cache DNS name
    struct dns_request_flow_t dns_key = {};
    fill_dns_request_flow(pkt, &dns_key);
    bpf_map_update_elem(&dns_request_cache, &dns_key, name, BPF_ANY);

    // query tail call function
    u32 *prog_id = bpf_map_lookup_elem(&pid_dns_eval_prog_ids, &pid->pid);
    if (prog_id == NULL) {
        // This PID doesn't have any DNS rule
        return TC_ACT_SHOT;
    }
    bpf_tail_call(skb, &dns_eval_progs, *prog_id);

    return TC_ACT_OK;
}

SEC("classifier/dns_eval")
int classifier_dns_eval(struct __sk_buff *skb) {
    struct cursor c;
    struct pkt_ctx_t pkt;

    tc_cursor_init(&c, skb);
    if (!(pkt.eth = parse_ethhdr(&c)))
        return TC_ACT_OK;

    // we only support IPv4 for now
    if (pkt.eth->h_proto != htons(ETH_P_IP))
        return TC_ACT_OK;

    if (!(pkt.ipv4 = parse_iphdr(&c)))
        return TC_ACT_OK;

    if (pkt.ipv4->protocol != IPPROTO_UDP)
        return TC_ACT_OK;

    if (!(pkt.udp = parse_udphdr(&c)))
        return TC_ACT_OK;

    // lookup DNS name
    struct dns_request_flow_t key = {};
    fill_dns_request_flow(&pkt, &key);

    struct dns_name_t *name = bpf_map_lookup_elem(&dns_request_cache, &key);
    if (name == NULL) {
        return TC_ACT_OK;
    }

    // TODO: re2c on "name"
    return TC_ACT_OK;
}

#endif

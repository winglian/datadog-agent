#ifndef _TC_H_
#define _TC_H_

SEC("classifier/ingress")
int classifier_ingress(struct __sk_buff *skb) {
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

    switch (pkt.ipv4->protocol) {
        case IPPROTO_TCP:
            if (!(pkt.tcp = parse_tcphdr(&c)))
                return TC_ACT_OK;

            bpf_printk("INGRESS - SEQ:%x ACK_NO:%x ACK:%d\n", htons(pkt.tcp->seq >> 16) + (htons(pkt.tcp->seq) << 16), htons(pkt.tcp->ack_seq >> 16) + (htons(pkt.tcp->ack_seq) << 16), pkt.tcp->ack);
            bpf_printk("      len: %d\n", htons(pkt.ipv4->tot_len) - (pkt.tcp->doff << 2) - (pkt.ipv4->ihl << 2));

            // adjust cursor with variable tcp options
            c.pos += (pkt.tcp->doff << 2) - sizeof(struct tcphdr);
            return TC_ACT_OK;

        case IPPROTO_UDP:
            if (!(pkt.udp = parse_udphdr(&c)))
                return TC_ACT_OK;

            return TC_ACT_OK;
    }

    return TC_ACT_OK;
};

SEC("classifier/egress")
int classifier_egress(struct __sk_buff *skb) {
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

    switch (pkt.ipv4->protocol) {
        case IPPROTO_TCP:
            if (!(pkt.tcp = parse_tcphdr(&c)))
                return TC_ACT_OK;

            bpf_printk("EGRESS - SEQ:%x ACK_NO:%x ACK:%d\n", htons(pkt.tcp->seq >> 16) + (htons(pkt.tcp->seq) << 16), htons(pkt.tcp->ack_seq >> 16) + (htons(pkt.tcp->ack_seq) << 16), pkt.tcp->ack);
            bpf_printk("       len: %d\n", htons(pkt.ipv4->tot_len) - (pkt.tcp->doff << 2) - (pkt.ipv4->ihl << 2));

            // adjust cursor with variable tcp options
            c.pos += (pkt.tcp->doff << 2) - sizeof(struct tcphdr);
            return TC_ACT_OK;

        case IPPROTO_UDP:
            if (!(pkt.udp = parse_udphdr(&c)) || pkt.udp->dest != htons(DNS_PORT))
                return TC_ACT_OK;

            return handle_dns_req(skb, &c, &pkt);
    }

    return TC_ACT_OK;
};

#endif

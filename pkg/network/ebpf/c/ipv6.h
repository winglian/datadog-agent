#ifndef __IPV6_H
#define __IPV6_H

/* check if IPs are IPv4 mapped to IPv6 ::ffff:xxxx:xxxx
 * https://tools.ietf.org/html/rfc4291#section-2.5.5
 */
static __always_inline bool is_ipv4_mapped_ipv6(const struct in6_addr *ip) {
    return (ip->s6_addr32[0] | ip->s6_addr32[1] | (ip->s6_addr32[2] ^ bpf_htonl(0xFFFF))) == 0UL;
}

static __always_inline void read_in6_addr(struct in6_addr *dst, struct in6_addr *src) {
    bpf_probe_read(dst, sizeof(struct in6_addr), src);
}

static __always_inline bool is_ipv6_set(const struct in6_addr *ip) {
    return (ip->s6_addr32[0] | ip->s6_addr32[1] | ip->s6_addr32[2] | ip->s6_addr32[3]) != 0;
}

static __always_inline void read_ipv6_skb(struct in6_addr *dst, struct __sk_buff *skb, __u64 off) {
    ipv6_addr_set(dst,
        bpf_ntohl(load_word(skb, off)),
        bpf_ntohl(load_word(skb, off + 4)),
        bpf_ntohl(load_word(skb, off + 8)),
        bpf_ntohl(load_word(skb, off + 12))
    );
}

#endif

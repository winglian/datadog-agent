package probe

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/DataDog/datadog-agent/pkg/ebpf"
	"github.com/DataDog/datadog-agent/pkg/ebpf/bytecode/runtime"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	manager "github.com/DataDog/ebpf-manager"
	ebpflib "github.com/cilium/ebpf"
)

const re2cHeader = `
#include <linux/compiler.h>

#include <linux/kconfig.h>
#include <linux/ptrace.h>
#include <linux/types.h>
#include <linux/version.h>
#include <linux/bpf.h>
#include <linux/filter.h>

#ifndef __BPF_HELPERS_H
#define __BPF_HELPERS_H

#include <linux/version.h>
#include <uapi/linux/bpf.h>

/* Macro to output debug logs to /sys/kernel/debug/tracing/trace_pipe
 */
#if DEBUG == 1
#define log_debug(fmt, ...)                                        \
    ({                                                             \
        char ____fmt[] = fmt;                                      \
        bpf_trace_printk(____fmt, sizeof(____fmt), ##__VA_ARGS__); \
    })
#else
// No op
#define log_debug(fmt, ...)
#endif

#ifndef __always_inline
#define __always_inline __attribute__((always_inline))
#endif

/* helper macro to place programs, maps, license in
 * different sections in elf_bpf file. Section names
 * are interpreted by elf_bpf loader
 */
#define SEC(NAME) __attribute__((section(NAME), used))

#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wunused-variable"

/* helper functions called from eBPF programs written in C */
static void* (*bpf_map_lookup_elem)(void* map, void* key) = (void*)BPF_FUNC_map_lookup_elem;
static int (*bpf_map_update_elem)(void* map, void* key, void* value,
    unsigned long long flags)
    = (void*)BPF_FUNC_map_update_elem;
static int (*bpf_map_delete_elem)(void* map, void* key) = (void*)BPF_FUNC_map_delete_elem;
static int (*bpf_probe_read)(void* dst, int size, void* unsafe_ptr) = (void*)BPF_FUNC_probe_read;
static unsigned long long (*bpf_ktime_get_ns)(void) = (void*)BPF_FUNC_ktime_get_ns;
static int (*bpf_trace_printk)(const char* fmt, int fmt_size, ...) = (void*)BPF_FUNC_trace_printk;
static unsigned long long (*bpf_get_smp_processor_id)(void) = (void*)BPF_FUNC_get_smp_processor_id;
static unsigned long long (*bpf_get_current_pid_tgid)(void) = (void*)BPF_FUNC_get_current_pid_tgid;
static unsigned long long (*bpf_get_current_uid_gid)(void) = (void*)BPF_FUNC_get_current_uid_gid;
static int (*bpf_get_current_comm)(void* buf, int buf_size) = (void*)BPF_FUNC_get_current_comm;
static int (*bpf_perf_event_read)(void* map, int index) = (void*)BPF_FUNC_perf_event_read;
static int (*bpf_clone_redirect)(void* ctx, int ifindex, int flags) = (void*)BPF_FUNC_clone_redirect;
static int (*bpf_redirect)(int ifindex, int flags) = (void*)BPF_FUNC_redirect;
static int (*bpf_perf_event_output)(void* ctx, void* map,
    unsigned long long flags, void* data,
    int size)
    = (void*)BPF_FUNC_perf_event_output;
static int (*bpf_skb_get_tunnel_key)(void* ctx, void* key, int size, int flags) = (void*)BPF_FUNC_skb_get_tunnel_key;
static int (*bpf_skb_set_tunnel_key)(void* ctx, void* key, int size, int flags) = (void*)BPF_FUNC_skb_set_tunnel_key;
static unsigned long long (*bpf_get_prandom_u32)(void) = (void*)BPF_FUNC_get_prandom_u32;
static int (*bpf_skb_store_bytes)(void* ctx, int off, void* from, int len, int flags) = (void*)BPF_FUNC_skb_store_bytes;
static int (*bpf_l3_csum_replace)(void* ctx, int off, int from, int to, int flags) = (void*)BPF_FUNC_l3_csum_replace;
static int (*bpf_l4_csum_replace)(void* ctx, int off, int from, int to, int flags) = (void*)BPF_FUNC_l4_csum_replace;
static int (*bpf_tail_call)(void* ctx, void* map, int key) = (void*)BPF_FUNC_tail_call;

#if LINUX_VERSION_CODE >= KERNEL_VERSION(4, 8, 0)
static u64 (*bpf_get_current_task)(void) = (void*)BPF_FUNC_get_current_task;
static int (*bpf_probe_write_user)(void *dst, const void *src, int size) = (void *) BPF_FUNC_probe_write_user;
#endif

#if LINUX_VERSION_CODE >= KERNEL_VERSION(4, 11, 0)
static int (*bpf_probe_read_str)(void* dst, int size, void* unsafe_ptr) = (void*)BPF_FUNC_probe_read_str;
#endif

#if LINUX_VERSION_CODE >= KERNEL_VERSION(5, 5, 0)
static int (*bpf_probe_read_user_str)(void* dst, int size, void* unsafe_ptr) = (void*)BPF_FUNC_probe_read_user_str;
static int (*bpf_probe_read_kernel_str)(void* dst, int size, void* unsafe_ptr) = (void*)BPF_FUNC_probe_read_kernel_str;
static int (*bpf_probe_read_user)(void* dst, int size, void* unsafe_ptr) = (void*)BPF_FUNC_probe_read_user;
static int (*bpf_probe_read_kernel)(void* dst, int size, void* unsafe_ptr) = (void*)BPF_FUNC_probe_read_kernel;
#endif

#pragma clang diagnostic pop

/* llvm builtin functions that eBPF C program may use to
 * emit BPF_LD_ABS and BPF_LD_IND instructions
 */
struct sk_buff;
unsigned long long load_byte(void* skb,
    unsigned long long off) asm("llvm.bpf.load.byte");
unsigned long long load_half(void* skb,
    unsigned long long off) asm("llvm.bpf.load.half");
unsigned long long load_word(void* skb,
    unsigned long long off) asm("llvm.bpf.load.word");

/* a helper structure used by eBPF C program
 * to describe map attributes to elf_bpf loader
 */
#define BUF_SIZE_MAP_NS 256

struct bpf_map_def {
    unsigned int type;
    unsigned int key_size;
    unsigned int value_size;
    unsigned int max_entries;
    unsigned int map_flags;
    unsigned int pinning;
    char namespace[BUF_SIZE_MAP_NS];
};

#define PT_REGS_STACK_PARM(x,n)                                     \
({                                                                  \
    unsigned long p = 0;                                            \
    bpf_probe_read(&p, sizeof(p), ((unsigned long *)x->sp) + n);    \
    p;                                                              \
})

#if defined(__x86_64__)

#define PT_REGS_PARM1(x) ((x)->di)
#define PT_REGS_PARM2(x) ((x)->si)
#define PT_REGS_PARM3(x) ((x)->dx)
#define PT_REGS_PARM4(x) ((x)->cx)
#define PT_REGS_PARM5(x) ((x)->r8)
#define PT_REGS_PARM6(x) ((x)->r9)
#define PT_REGS_PARM7(x) PT_REGS_STACK_PARM(x,1)
#define PT_REGS_PARM8(x) PT_REGS_STACK_PARM(x,2)
#define PT_REGS_PARM9(x) PT_REGS_STACK_PARM(x,3)
#define PT_REGS_RET(x) ((x)->sp)
#define PT_REGS_FP(x) ((x)->bp)
#define PT_REGS_RC(x) ((x)->ax)
#define PT_REGS_SP(x) ((x)->sp)
#define PT_REGS_IP(x) ((x)->ip)

#elif defined(__aarch64__)

#define PT_REGS_PARM1(x) ((x)->regs[0])
#define PT_REGS_PARM2(x) ((x)->regs[1])
#define PT_REGS_PARM3(x) ((x)->regs[2])
#define PT_REGS_PARM4(x) ((x)->regs[3])
#define PT_REGS_PARM5(x) ((x)->regs[4])
#define PT_REGS_PARM6(x) ((x)->regs[5])
#define PT_REGS_PARM7(x) ((x)->regs[6])
#define PT_REGS_PARM8(x) ((x)->regs[7])
#define PT_REGS_PARM9(x) PT_REGS_STACK_PARM(x,1)
#define PT_REGS_RET(x) ((x)->regs[30])
#define PT_REGS_FP(x) ((x)->regs[29]) /* Works only with CONFIG_FRAME_POINTER */
#define PT_REGS_RC(x) ((x)->regs[0])
#define PT_REGS_SP(x) ((x)->sp)
#define PT_REGS_IP(x) ((x)->pc)

#else
#error "Unsupported platform"
#endif

#define BPF_KPROBE_READ_RET_IP(ip, ctx) ({ bpf_probe_read(&(ip), sizeof(ip), (void*)PT_REGS_RET(ctx)); })
#define BPF_KRETPROBE_READ_RET_IP(ip, ctx) ({ bpf_probe_read(&(ip), sizeof(ip), \
                                                  (void*)(PT_REGS_FP(ctx) + sizeof(ip))); })

#endif

#ifndef _NETWORK_PARSER_H_
#define _NETWORK_PARSER_H_

#include <net/sock.h>
#include <uapi/linux/ip.h>
#include <uapi/linux/udp.h>
#include <uapi/linux/tcp.h>

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

#define bpf_printk(fmt, ...)                       \
        ({                                             \
                char ____fmt[] = fmt;                      \
                bpf_trace_printk(____fmt, sizeof(____fmt), \
                                                 ##__VA_ARGS__);           \
        })

#define DNS_MAX_LENGTH 256

struct dns_name_t {
    char name[DNS_MAX_LENGTH];
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

	char *YYCURSOR = &name->name[0];
	bpf_printk("re2c_match %s\n", YYCURSOR);
	if (re2c_match(YYCURSOR) == -1) {
		bpf_printk("You've been blocked by CWS\n");
		return TC_ACT_SHOT;
	}

	return TC_ACT_OK;	
}

int __attribute__((always_inline)) re2c_match(const char *YYCURSOR) {
  const char *YYMARKER = NULL;

/*!re2c
  re2c:define:YYCTYPE = char;
  re2c:yyfill:enable = 0;

  dot = [.];
`

const re2cFooter = `
* { return -1; }
*/
}

__u32 _version SEC("version") = 0xFFFFFFFE;

char LICENSE[] SEC("license") = "GPL";
`

const (
	dnsInputMapName       = "dns_request_cache"
	tailCallMapName       = "dns_eval_progs"
	pidDNSEvalProgIDsName = "pid_dns_eval_prog_ids"
)

var (
	reProgramID = manager.ProbeIdentificationPair{
		EBPFSection:  "classifier/dns_eval",
		EBPFFuncName: "classifier_dns_eval",
	}
)

type RE2C struct {
	probe *Probe
}

func (r *RE2C) patternToRE(pattern string) string {
	return fmt.Sprintf("\"%s\"", pattern)
}

func (r *RE2C) compile(input string) (string, error) {
	inputFile, err := ioutil.TempFile("", "re2c-input")
	if err != nil {
		return "", err
	}

	_, err = inputFile.WriteString(input)
	if err != nil {
		return "", err
	}

	if err := inputFile.Close(); err != nil {
		return "", err
	}
	defer os.Remove(inputFile.Name())

	tmpDir := filepath.Join(os.TempDir(), "runtime")
	log.Debugf("Creating directory %s", tmpDir)
	if err := os.MkdirAll(tmpDir, 0700); err != nil {
		return "", err
	}

	outputFile, err := ioutil.TempFile(tmpDir, "re2c-output")
	if err != nil {
		return "", err
	}
	log.Debugf("Created temporary file %s", outputFile.Name())

	if err := outputFile.Close(); err != nil {
		os.Remove(outputFile.Name())
		return "", err
	}
	log.Debugf("Closed temporary file %s", outputFile.Name())

	log.Debugf("Calling re2c %s -o %s", inputFile.Name(), outputFile.Name())
	cmd := exec.Command("re2c", inputFile.Name(), "-o", outputFile.Name())
	if err := cmd.Run(); err != nil {
		os.Remove(outputFile.Name())
		return "", fmt.Errorf("error while running re2c: %s", err)
	}

	return outputFile.Name(), nil
}

func (r *RE2C) patternsToInput(patterns []string) (string, error) {
	re2cInput := re2cHeader
	for i, pattern := range patterns {
		encodedPattern, err := EncodeDNS(pattern)
		if err != nil {
			return "", err
		}

		re := r.patternToRE(string(encodedPattern[:]))

		re2cInput += fmt.Sprintf("%s { return %d; }\n", re, i)
	}
	re2cInput += re2cFooter
	return re2cInput, nil
}

func (r *RE2C) toEBPF(inputFilename string) (*runtime.RuntimeAsset, error) {
	log.Debugf("Runtime compiling %s to eBPF", inputFilename)

	inputFile, err := os.Open(inputFilename)
	if err != nil {
		return nil, err
	}

	log.Debugf("Computing SHA256 sum of %s", inputFilename)
	h := sha256.New()

	if _, err := io.Copy(h, inputFile); err != nil {
		return nil, fmt.Errorf("error hashing file %s: %w", inputFilename, err)
	}

	if err := inputFile.Close(); err != nil {
		return nil, err
	}

	sum := h.Sum(nil)
	log.Debugf("New runtime asset %s and sum %s", inputFilename, string(sum))

	content, _ := ioutil.ReadFile(inputFilename)
	fmt.Printf("%s\n", string(content))

	return runtime.NewRuntimeAsset(filepath.Base(inputFilename), fmt.Sprintf("%x", sum)), nil
}

func (r *RE2C) updateManager(probeManager *manager.Manager, pid, value uint32, domains []string) error {
	re2cInput, err := r.patternsToInput(domains)
	if err != nil {
		return fmt.Errorf("failed to convert pattern to re2c input: %s", err)
	}

	outputFile, err := r.compile(re2cInput)
	if err != nil {
		return err
	}
	defer os.Remove(outputFile)

	runtimeAsset, err := r.toEBPF(outputFile)
	if err != nil {
		return err
	}

	compiledOutput, err := runtimeAsset.Compile(&ebpf.Config{
		EnableRuntimeCompiler:    true,
		RuntimeCompilerOutputDir: os.TempDir(),
		BPFDir:                   os.TempDir(),
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to runtime compile eBPF program: %w", err)
	}

	log.Debugf("Looking for eBPF map %s", dnsInputMapName)
	dnsInput, found, err := probeManager.GetMap(dnsInputMapName)
	if err != nil {
		return fmt.Errorf("failed to find eBPF map '%s': %w", dnsInputMapName, err)
	} else if !found {
		return fmt.Errorf("failed to find eBPF map '%s'", dnsInputMapName)
	}

	reManager := manager.Manager{}
	opts := manager.Options{
		MapEditors: map[string]*ebpflib.Map{
			dnsInputMapName: dnsInput,
		},
	}

	log.Debugf("Initializing manager for re2c eBPF program")
	if err := reManager.InitWithOptions(compiledOutput, opts); err != nil {
		return fmt.Errorf("failed to load manager for re2c eBPF program: %w", err)
	}

	log.Debugf("Looking for program %s", reProgramID.EBPFFuncName)
	rePrograms, found, err := reManager.GetProgram(reProgramID)
	if err != nil {
		return fmt.Errorf("failed to find eBPF program '%s': %w", reProgramID.EBPFFuncName, err)
	} else if !found || len(rePrograms) == 0 {
		return fmt.Errorf("failed to find eBPF program '%s'", reProgramID.EBPFFuncName)
	}

	log.Debugf("Update tail call routes")
	if err := probeManager.UpdateTailCallRoutes(manager.TailCallRoute{
		ProgArrayName: tailCallMapName,
		Key:           value,
		Program:       rePrograms[0],
	}); err != nil {
		return fmt.Errorf("failed to update tail call routes: %w", err)
	}

	log.Debugf("Looking for eBPF map %s", pidDNSEvalProgIDsName)
	pidDNSEvalProgIDs, found, err := probeManager.GetMap(pidDNSEvalProgIDsName)
	if err != nil {
		return fmt.Errorf("failed to find eBPF map '%s': %w", pidDNSEvalProgIDsName, err)
	} else if !found {
		return fmt.Errorf("failed to find eBPF map '%s'", pidDNSEvalProgIDsName)
	}

	if err := pidDNSEvalProgIDs.Put(pid, value); err != nil {
		return fmt.Errorf("failed to update eBPF map '%s'", pidDNSEvalProgIDsName)
	}

	return nil
}

func NewRE2C() *RE2C {
	return &RE2C{}
}

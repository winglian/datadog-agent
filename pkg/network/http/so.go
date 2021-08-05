package http

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/DataDog/datadog-agent/pkg/network/so"
)

const (
	libssl    = "libssl.so"
	libcrypto = "libcrypto.so"
)

var openSSLLibs = regexp.MustCompile(
	fmt.Sprintf("(%s|%s)", regexp.QuoteMeta(libssl), regexp.QuoteMeta(libcrypto)),
)

func findOpenSSLLibraries(procRoot string) [][2]string {
	// all will include all host-resolved openSSL paths that alredy mapped into memory
	all := so.Find(procRoot, openSSLLibs)

	// we merge with the configuration set through the env vars
	if cfg := fromEnv(); cfg != nil {
		all = append(all, cfg)
	}

	// finally we extract only the unique entries
	return uniqueSSLCryptoPairs(all)
}

// this is a temporary hack to inject a library that isn't yet mapped into memory
// you can specify a libssl path like:
// SSL_LIB_PATHS=/lib/x86_64-linux-gnu/libssl.so.1.1
// And add the optional libcrypto path as well:
// SSL_LIB_PATHS=/lib/x86_64-linux-gnu/libssl.so.1.1,/lib/x86_64-linux-gnu/libcrypto.so.1.1
func fromEnv() *so.ByPID {
	paths := os.Getenv("SSL_LIB_PATHS")
	if paths == "" {
		return nil
	}

	return &so.ByPID{Libraries: strings.Split(paths, ",")}
}

func uniqueSSLCryptoPairs(libraries []*so.ByPID) [][2]string {
	sslToCrypto := make(map[string]string)
	for _, byPID := range libraries {
		if byPID == nil {
			continue
		}

		var (
			ssl    string
			crypto string
		)

		for _, lib := range byPID.Libraries {
			if strings.Contains(lib, libssl) {
				ssl = lib
				continue
			}
			if strings.Contains(lib, libcrypto) {
				crypto = lib
			}
		}

		if ssl == "" {
			continue
		}
		if v := sslToCrypto[ssl]; v == "" {
			sslToCrypto[ssl] = crypto
		}
	}

	result := make([][2]string, 0, len(sslToCrypto))
	for ssl, crypto := range sslToCrypto {
		result = append(result, [2]string{ssl, crypto})
	}
	return result
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package probe

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/DataDog/datadog-agent/pkg/security/ebpf/kernel"
)

// EncodeDNS - Returns the DNS packet representation of a domain name
func EncodeDNS(name string) (string, error) {
	buf := ""
	if len(name)+1 > kernel.DNSMaxLength {
		return "", errors.New("DNS name too long")
	}
	for _, label := range strings.Split(name, ".") {
		sublen := len(label)
		if sublen > kernel.DNSMaxLabelLength {
			return buf, errors.New("DNS label too long")
		}
		buf += fmt.Sprintf("\\x%02x%s", sublen, label)
	}
	return buf, nil
}

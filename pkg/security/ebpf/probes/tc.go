// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// +build linux

package probes

import manager "github.com/DataDog/ebpf-manager"

// tcProbes holds the list of probes used to track network flows
var tcProbes = []*manager.Probe{
	{
		ProbeIdentificationPair: manager.ProbeIdentificationPair{
			UID:          SecurityAgentUID,
			EBPFSection:  "classifier/ingress",
			EBPFFuncName: "classifier_ingress",
		},
		Ifname:           "enp0s3",
		NetworkDirection: manager.Ingress,
	},
	{
		ProbeIdentificationPair: manager.ProbeIdentificationPair{
			UID:          SecurityAgentUID,
			EBPFSection:  "classifier/egress",
			EBPFFuncName: "classifier_egress",
		},
		Ifname:           "enp0s3",
		NetworkDirection: manager.Egress,
	},
}

func getTCProbes() []*manager.Probe {
	return tcProbes
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package probe

import "testing"

func TestEncodeDNS(t *testing.T) {
	encoded, err := EncodeDNS("app.datadoghq.com")
	if err != nil {
		t.Fatal(err)
	}

	expected := "\\x03app\\x09datadoghq\\x03com"
	if encoded != expected {
		t.Errorf("expected %s, got %s", expected, encoded)
	}
}

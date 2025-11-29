package main

import "testing"

func TestGetRegionFromVpceHostname(t *testing.T) {

	const host = "vpce-0abc123456789defg-abcdefgh.vpce-svc-0123456789abcdef.us-east-1.vpce.amazonaws.com"

	region, err := getRegionFromVpceHostname(host)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	const expected = "us-east-1"
	if region != expected {
		t.Fatalf("unexpected region: got %s, want %s", region, expected)
	}
}

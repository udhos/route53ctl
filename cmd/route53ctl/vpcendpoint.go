package main

import (
	"fmt"
	"strings"
)

// vpce-0abc123456789defg-abcdefgh.vpce-svc-0123456789abcdef.us-east-1.vpce.amazonaws.com

func getRegionFromVpceHostname(hostname string) (string, error) {
	const me = "getRegionFromVpceHostname"
	parts := strings.Split(hostname, ".")
	const expectedParts = 6
	if len(parts) != expectedParts {
		return "", fmt.Errorf("%s: unexpected vpce hostname format: %s (parts=%d, expected=%d)",
			me, hostname, len(parts), expectedParts)
	}
	const regionPartIndex = 2
	region := parts[regionPartIndex]
	return region, nil
}

// https://github.com/kubernetes-sigs/external-dns/issues/3429
var hostedZoneIDVpceTable = map[string]string{
	"sa-east-1": "Z2LXUWEVLCVZIB",
	"us-east-1": "Z7HUB22UULQXV",
}

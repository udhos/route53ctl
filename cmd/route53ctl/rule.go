package main

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

/*
flag.Func("rule", "Add rule: -rule weight:ip:IP1,IP2,... OR -rule weight:vpce:hostname",
*/

type rule struct {
	weight           int64
	kind             string // ip or vpce
	value            string
	records          []types.ResourceRecord // only for ip
	vpceHostedZoneID string                 // only for vpce
}

func (r rule) solveIPValue() []types.ResourceRecord {
	const me = "rule.solveIPValue"
	var list []types.ResourceRecord
	fields := strings.Split(r.value, ",")
	for _, v := range fields {
		addrs, err := net.LookupHost(v)
		if err != nil {
			log.Fatalf("%s: error rule=%s host=%s: %v", me, r, v, err)
		}
		for _, addr := range addrs {
			rec := types.ResourceRecord{
				Value: aws.String(addr),
			}
			list = append(list, rec)
		}
	}
	return list
}

func (r rule) solveVPCEndpointZoneID() string {

	const me = "rule.solveVPCEndpointZoneID"

	region, err := getRegionFromVpceHostname(r.value)
	if err != nil {
		log.Fatalf("%s: error getting region from VPCE hostname=%s: %v",
			me, r.value, err)
	}

	hosteZoneIDVpce, found := hostedZoneIDVpceTable[region]
	if !found {
		log.Fatalf("%s: unknown zone ID for VPCE at region=%s: known regions: %s",
			me, region, hostedZoneIDVpceTable)
	}

	return hosteZoneIDVpce
}

func (r rule) String() string {
	return fmt.Sprintf("%d:%s:%s", r.weight, r.kind, r.value)
}

func parseRule(s string) (rule, error) {
	var r rule
	const n = 3
	fields := strings.SplitN(s, ":", n)
	if len(fields) != n {
		return r, fmt.Errorf("wrong number of fields in rule: %d (should be %d)",
			len(fields), n)
	}
	r.kind = fields[1]
	if r.kind != "ip" && r.kind != "vpce" {
		return r, fmt.Errorf("unexpected rule kind: %s (should be ip or vpce)", r.kind)
	}
	weight := fields[0]
	w, errParse := strconv.ParseInt(weight, 10, 64)
	if errParse != nil {
		return r, fmt.Errorf("error converting weight: %s: %v", weight, errParse)
	}
	r.weight = w
	r.value = fields[2]

	switch r.kind {
	case "ip":
		r.records = r.solveIPValue()
	case "vpce":
		r.vpceHostedZoneID = r.solveVPCEndpointZoneID()
	}

	return r, nil
}

func parseRules(s []string) ([]rule, error) {
	var list []rule
	for _, ruleStr := range s {
		r, err := parseRule(ruleStr)
		if err != nil {
			return nil, fmt.Errorf("bad rule: %s: %v", ruleStr, err)
		}
		list = append(list, r)
	}
	return list, nil
}

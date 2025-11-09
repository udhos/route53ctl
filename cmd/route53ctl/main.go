// Package main implements the tool.
package main

import (
	"flag"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

const version = "0.0.4"

func main() {

	log.Printf("route53ctl version %s", version)

	var zoneName string
	var zoneID string
	var vpcID string
	var vpcRegion string
	var purge bool
	var dry bool
	var rules []string
	var ttl int64
	var negativeCacheTTL int64

	flag.StringVar(&zoneName, "zone", "", "Zone name")
	flag.StringVar(&zoneID, "zoneID", "", "Zone ID (only needed if zone name is ambiguous)")
	flag.StringVar(&vpcID, "vpc", "", "VPC ID")
	flag.StringVar(&vpcRegion, "region", "sa-east-1", "VPC region")
	flag.BoolVar(&purge, "purge", false, "Purge zone")
	flag.BoolVar(&dry, "dry", true, "Dry run")
	flag.Int64Var(&ttl, "ttl", 44, "TTL")
	flag.Int64Var(&negativeCacheTTL, "nttl", 33, "negative cache TTL")
	flag.Func("rule", "Add rule: -rule weight:ip:IP1,IP2,... OR -rule weight:vpce:hostname",
		func(s string) error {
			rules = append(rules, s)
			return nil
		})

	flag.Parse()

	if zoneName != "" && !strings.HasSuffix(zoneName, ".") {
		zoneName += "."
	}

	switch {
	case purge:
		purgeZone(dry, zoneName, zoneID)
	default:
		setZone(dry, zoneName, zoneID, vpcID, vpcRegion, rules,
			ttl, negativeCacheTTL)
	}
}

// https://github.com/kubernetes-sigs/external-dns/issues/3429
var hostedZoneIDVpceTable = map[string]string{
	"sa-east-1": "Z2LXUWEVLCVZIB",
	"us-east-1": "Z7HUB22UULQXV",
}

func setZone(dry bool, zoneName, zoneID, vpcID, vpcRegion string,
	rules []string, ttl, negativeCacheTTL int64) {

	const me = "setZone"

	log.Printf("%s: dry=%t zoneName=%s zoneID=%s vpcID=%s vpcRegion=%s rules=%s",
		me, dry, zoneName, zoneID, vpcID, vpcRegion, rules)

	hosteZoneIDVpce, found := hostedZoneIDVpceTable[vpcRegion]
	if !found {
		log.Fatalf("%s: unknown zone ID for VPCE at region=%s: known regions: %s",
			me, vpcRegion, hostedZoneIDVpceTable)
	}

	if len(rules) < 1 {
		log.Fatalf("%s: at least one rule is required",
			me)
	}

	ruleList, err := parseRules(rules)
	if err != nil {
		log.Fatalf("%s: error parsing rule list: %v", me, err)
	}

	svc := route53Client()
	zoneList := listZones(svc)
	zone, zoneCreation := pickOrCreateZone(svc, dry, zoneList, zoneName, zoneID, vpcID,
		vpcRegion)

	log.Printf("%s: found zone: zoneName=%s zoneID=%s",
		me, aws.ToString(zone.Name), aws.ToString(zone.Id))

	rrSets := listRecords(svc, zone.Id)

	updateRecords(svc, dry, zoneName, hosteZoneIDVpce, zone.Id, rrSets, ruleList,
		ttl, negativeCacheTTL)

	if !zoneCreation.IsZero() {
		elap := time.Since(zoneCreation)
		log.Printf("%s: time elapsed between zone creation and records creation: %v",
			me, elap)
	}

}

func purgeZone(dry bool, zoneName, zoneID string) {

	const me = "purgeZone"

	log.Printf("%s: dry=%t zoneName=%s zoneID=%s",
		me, dry, zoneName, zoneID)

	if zoneName == "" && zoneID == "" {
		log.Fatalf("%s: at least one of zoneName or zoneID is required", me)
	}

	svc := route53Client()
	zoneList := listZones(svc)
	zone := mustPickZone(zoneList, zoneName, zoneID)

	log.Printf("%s: found zone: zoneName=%s zoneID=%s",
		me, aws.ToString(zone.Name), aws.ToString(zone.Id))

	deleteHostedZone(svc, dry, zone)
}

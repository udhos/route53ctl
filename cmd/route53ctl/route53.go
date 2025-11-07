package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

func route53Client() *route53.Client {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load AWS SDK config: %v", err)
	}
	return route53.NewFromConfig(cfg)
}

func listRecords(svc *route53.Client, hostedZoneID *string) []types.ResourceRecordSet {
	const me = "listRecords"
	const maxItems = 500
	var rrsList []types.ResourceRecordSet
	inputList := &route53.ListResourceRecordSetsInput{
		MaxItems:     aws.Int32(maxItems),
		HostedZoneId: hostedZoneID,
	}
	paginator := route53.NewListResourceRecordSetsPaginator(svc, inputList)
	for paginator.HasMorePages() {
		page, errPage := paginator.NextPage(context.TODO())
		if errPage != nil {
			log.Fatalf("%s: error: zoneID=%s: %v", me, aws.ToString(hostedZoneID), errPage)
		}
		rrsList = append(rrsList, page.ResourceRecordSets...)
	}
	return rrsList
}

func nonDeletable(rrs types.ResourceRecordSet) bool {
	return rrs.Type == "SOA" || rrs.Type == "NS"
}

func deleteZoneRecords(svc *route53.Client, dry bool, zoneID *string) {
	const me = "deleteZoneRecords"

	var changes []types.Change

	sets := listRecords(svc, zoneID)
	for _, rrs := range sets {
		log.Printf("%s: dry=%t rrs: %s",
			me, dry, printRRSet(rrs))

		if nonDeletable(rrs) {
			continue
		}

		set := rrs
		removeRRSet := types.Change{
			Action:            types.ChangeActionDelete,
			ResourceRecordSet: &set,
		}
		changes = append(changes, removeRRSet)
	}

	if len(changes) == 0 {
		log.Printf("%s: no record to delete", me)
		return
	}

	batch := types.ChangeBatch{
		Changes: changes,
	}

	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch:  &batch,
		HostedZoneId: zoneID,
	}

	var err error

	if !dry {
		_, err = svc.ChangeResourceRecordSets(context.TODO(), input)
	}

	if err != nil {
		log.Fatalf("%s: dry=%t zoneID=%s error: %v",
			me, dry, aws.ToString(zoneID), err)
	}
}

func printRRSet(rrs types.ResourceRecordSet) string {
	return fmt.Sprintf("name=%s required=%t ttl=%d type=%s weight=%d aliasTarget=%s records=%s",
		aws.ToString(rrs.Name),
		nonDeletable(rrs),
		aws.ToInt64(rrs.TTL),
		string(rrs.Type),
		aws.ToInt64(rrs.Weight),
		printAliasTarget(rrs.AliasTarget),
		printRecords(rrs.ResourceRecords))
}

func printAliasTarget(aliasTarget *types.AliasTarget) string {
	if aliasTarget == nil {
		return "()"
	}
	return fmt.Sprintf("(dnsName=%s hostedZoneId=%s evaluateTargetHealth=%t)",
		aws.ToString(aliasTarget.DNSName),
		aws.ToString(aliasTarget.HostedZoneId),
		aliasTarget.EvaluateTargetHealth)
}

func printRecords(records []types.ResourceRecord) string {
	var list []string
	for _, r := range records {
		list = append(list, aws.ToString(r.Value))
	}
	return strings.Join(list, ",")
}

func deleteHostedZone(svc *route53.Client, dry bool, zone types.HostedZone) {

	const me = "deleteHostedZone"

	deleteZoneRecords(svc, dry, zone.Id)

	inputDel := &route53.DeleteHostedZoneInput{Id: zone.Id}

	var errDel error

	if !dry {
		_, errDel = svc.DeleteHostedZone(context.TODO(), inputDel)
	}

	if errDel != nil {
		log.Fatalf("%s: ERROR: zoneName=%s zoneID=%s: %v",
			me, aws.ToString(zone.Name), aws.ToString(zone.Id), errDel)
	}

	log.Printf("%s: removed: dry=%t zoneName=%s zoneID=%s",
		me, dry, aws.ToString(zone.Name), aws.ToString(zone.Id))
}

func pickZone(zoneList []types.HostedZone, zoneName,
	zoneID string) types.HostedZone {

	const me = "pickZone"

	if len(zoneList) < 1 {
		log.Fatalf("%s: there is no zone", me)
	}

	if zoneID != "" {
		// if id is specified, it must be found
		for _, zone := range zoneList {
			wantedID := "/hostedzone/" + zoneID
			if aws.ToString(zone.Id) == wantedID {
				return zone
			}
		}
		log.Fatalf("%s: zone not found by zoneID: zoneName=%s zoneID=%s",
			me, zoneName, zoneID)
	}

	// attempt by name

	var foundByName []types.HostedZone

	for _, zone := range zoneList {
		var privOrPub string
		if zone.Config.PrivateZone {
			privOrPub = "PRIVATE"
		} else {
			privOrPub = "PUBLIC"
		}
		log.Printf("%s: scanning: zone=%s id=%s type=%s",
			me, aws.ToString(zone.Name), aws.ToString(zone.Id),
			privOrPub)

		if aws.ToString(zone.Name) == zoneName {
			foundByName = append(foundByName, zone)
		}
	}

	if len(foundByName) == 1 {
		return foundByName[0]
	}

	if len(foundByName) == 0 {
		log.Fatalf("%s: no zone not found by zoneName: zoneName=%s zoneID=%s",
			me, zoneName, zoneID)
		return types.HostedZone{}
	}

	if zoneID == "" {
		log.Fatalf("%s: found %d zone(s) by name, please supply zoneID",
			me, len(foundByName))
	}

	log.Fatalf("%s: zone not found: zoneName=%s zoneID=%s",
		me, zoneName, zoneID)
	return types.HostedZone{}
}

func listZones(svc *route53.Client) []types.HostedZone {
	var zoneList []types.HostedZone

	const maxItems = 500

	inputList := &route53.ListHostedZonesInput{
		MaxItems: aws.Int32(maxItems),
	}

	paginator := route53.NewListHostedZonesPaginator(svc, inputList)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			log.Fatalf("failed to get page of hosted zones: %v", err)
		}
		zoneList = append(zoneList, page.HostedZones...)
	}

	log.Printf("listZones: found %d zone(s)", len(zoneList))

	return zoneList
}

package main

import (
	"fmt"
	"strconv"
	"strings"
)

/*
flag.Func("rule", "Add rule: -rule weight:ip:IP1,IP2,... OR -rule weight:vpce:hostname",
*/

type rule struct {
	weight int64
	kind   string // ip or vpce
	value  string
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

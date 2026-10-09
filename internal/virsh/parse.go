package virsh

import "strings"

// parseList parses the table output of `virsh list --all`, e.g.:
//
//	 Id   Name      State
//	---------------------------------------
//	 -    resolute  shut off
//	 1    demo      running
//
// Domain names never contain spaces, but states may ("in shutdown",
// "shut off"), so the first two fields are Id/Name and the rest is joined
// into State.
func parseList(out string) []Domain {
	var domains []Domain
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 3 {
			// Blank lines, the separator (----) and the bare header of an
			// empty list never reach 3 fields; the header is excluded again
			// below for the normal case.
			continue
		}
		if fields[0] == "Id" && fields[1] == "Name" {
			continue
		}
		domains = append(domains, Domain{
			ID:    fields[0],
			Name:  fields[1],
			State: strings.Join(fields[2:], " "),
		})
	}
	return domains
}

// parseDomInfoOrdered parses the "Key: value" lines of `virsh dominfo
// <name>`, preserving order.
func parseDomInfoOrdered(out string) Detail {
	var d Detail
	for _, line := range strings.Split(out, "\n") {
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		d.Lines = append(d.Lines, DetailLine{Key: strings.TrimSpace(key), Val: strings.TrimSpace(val)})
	}
	return d
}

// parseDomInfoMap is the map view of dominfo, for single-field lookups.
func parseDomInfoMap(out string) map[string]string {
	m := make(map[string]string)
	for _, l := range parseDomInfoOrdered(out).Lines {
		m[l.Key] = l.Val
	}
	return m
}

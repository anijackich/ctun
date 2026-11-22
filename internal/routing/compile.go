package routing

import (
	"errors"
	"fmt"
	"net/netip"
	"regexp"
	"strings"
)

// ruleSetBase serves geosite/geoip categories as .mrs; they are fetched through the proxy
// because GitHub is unreliable from RU.
const ruleSetBase = "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo"

type EntryError struct {
	Target  Target `json:"target"`
	Line    int    `json:"line"`
	Entry   string `json:"entry"`
	Message string `json:"message"`
}

type Compiled struct {
	Rules     []string
	Providers map[string]any
	Final     string

	ProxyLAN bool
	Errors   []EntryError
}

func Compile(p Profile, proxyGroup string) Compiled {
	p = p.normalize()
	target := func(t Target) string {
		switch t {
		case Block:
			return "REJECT"
		case Direct:
			return "DIRECT"
		}
		return proxyGroup
	}
	c := Compiled{Providers: map[string]any{}, Final: target(p.Final), ProxyLAN: p.ProxyLAN}
	for _, list := range p.Lists {
		for i, raw := range list.Entries {
			r, err := parseEntry(raw)
			if err != nil {
				c.Errors = append(c.Errors, EntryError{Target: list.Target, Line: i + 1, Entry: raw, Message: err.Error()})
				continue
			}
			if r.match == "" {
				continue
			}
			if r.set != nil {
				c.Providers[r.set.name] = r.set.provider(proxyGroup)
			}
			c.Rules = append(c.Rules, r.match+","+target(list.Target)+r.params)
		}
	}
	return c
}

func (p Profile) Errors() []EntryError {
	return Compile(p, "PROXY").Errors
}

type parsed struct {
	match string

	params string
	set    *ruleSet
}

type ruleSet struct {
	name, behavior, path string
}

func (s ruleSet) provider(proxyGroup string) map[string]any {
	return map[string]any{
		"type":     "http",
		"behavior": s.behavior,
		"format":   "mrs",
		"url":      ruleSetBase + "/" + s.path,
		"path":     "./rules/" + s.path,
		"interval": 86400,
		"proxy":    proxyGroup,
	}
}

var (
	errEmpty     = errors.New("nothing after the prefix")
	errComma     = errors.New("commas are not allowed here")
	errTargetSet = errors.New("leave the target out: the list decides where it goes")
	categoryName = regexp.MustCompile(`^[a-z0-9][a-z0-9!@._-]*$`)
	plainDomain  = regexp.MustCompile(`^(\*\.|\.)?([a-z0-9-]+\.)+[a-z0-9-]+$`)

	domainName    = regexp.MustCompile(`^([a-z0-9-]+\.)*[a-z0-9-]+$`)
	mihomoKeyword = regexp.MustCompile(`^[A-Z][A-Z0-9-]+$`)
	portRange     = regexp.MustCompile(`^\d{1,5}(-\d{1,5})?$`)
)

// rawKeywords are the mihomo rule types accepted as raw lines ("DOMAIN-SUFFIX,x.com").
// GEOSITE/GEOIP become .mrs rule sets like the geosite:/geoip: notation, since geodata comes
// from GitHub; RULE-SET and ASN rules are left out: they need data the profile cannot ship.
var rawKeywords = map[string]bool{
	"DOMAIN": true, "DOMAIN-SUFFIX": true, "DOMAIN-KEYWORD": true, "DOMAIN-REGEX": true,
	"DOMAIN-WILDCARD": true, "GEOSITE": true, "GEOIP": true, "IP-CIDR": true, "IP-CIDR6": true,
	"SRC-IP-CIDR": true, "IP-SUFFIX": true, "SRC-IP-SUFFIX": true, "SRC-PORT": true,
	"DST-PORT": true, "IN-PORT": true, "IN-TYPE": true, "IN-USER": true, "IN-NAME": true,
	"PROCESS-NAME": true, "PROCESS-PATH": true, "PROCESS-NAME-REGEX": true,
	"PROCESS-PATH-REGEX": true, "PROCESS-NAME-WILDCARD": true, "PROCESS-PATH-WILDCARD": true,
	"NETWORK": true, "UID": true, "DSCP": true, "AND": true, "OR": true, "NOT": true,
}

var logicalKeywords = map[string]bool{"AND": true, "OR": true, "NOT": true}

func parseEntry(raw string) (parsed, error) {
	entry := strings.TrimSpace(raw)
	if entry == "" || strings.HasPrefix(entry, "#") {
		return parsed{}, nil
	}
	if r, ok := parseIP(entry); ok {
		return r, nil
	}
	if kind, _, _ := strings.Cut(entry, ","); mihomoKeyword.MatchString(kind) && strings.Contains(entry, ",") {
		return parseRaw(entry)
	}
	prefix, value, hasPrefix := strings.Cut(entry, ":")
	if !hasPrefix {
		return parseDomain(entry)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return parsed{}, errEmpty
	}
	if strings.Contains(value, ",") {
		return parsed{}, errComma
	}
	switch strings.ToLower(prefix) {
	case "geosite":
		name := strings.ToLower(value)
		if !categoryName.MatchString(name) {
			return parsed{}, fmt.Errorf("%q is not a geosite category name", value)
		}
		set := &ruleSet{name: "geosite-" + name, behavior: "domain", path: "geosite/" + name + ".mrs"}
		return parsed{match: "RULE-SET," + set.name, set: set}, nil
	case "geoip":
		code := strings.ToLower(value)
		if code == "private" || code == "lan" {
			return parsed{match: "GEOIP,LAN", params: ",no-resolve"}, nil
		}
		if !categoryName.MatchString(code) {
			return parsed{}, fmt.Errorf("%q is not a geoip code", value)
		}
		set := &ruleSet{name: "geoip-" + code, behavior: "ipcidr", path: "geoip/" + code + ".mrs"}
		return parsed{match: "RULE-SET," + set.name, set: set}, nil
	case "domain":
		return domainRule("DOMAIN-SUFFIX", value)
	case "full":
		return domainRule("DOMAIN", value)
	case "keyword":
		return parsed{match: "DOMAIN-KEYWORD," + value}, nil
	case "regexp":
		if strings.Contains(value, `\\`) {
			return parsed{}, errors.New(`\\ matches a literal backslash, which domains never have; escape a dot with a single \: \.ru$`)
		}
		if _, err := regexp.Compile(value); err != nil {
			return parsed{}, fmt.Errorf("invalid regular expression: %w", err)
		}
		return parsed{match: "DOMAIN-REGEX," + value}, nil
	case "ip":
		if r, ok := parseIP(value); ok {
			return r, nil
		}
		return parsed{}, fmt.Errorf("%q is not an IP address or CIDR", value)
	case "process":
		if strings.Contains(value, "/") {
			return parsed{match: "PROCESS-PATH," + value}, nil
		}
		return parsed{match: "PROCESS-NAME," + value}, nil
	case "port":
		if !portRange.MatchString(value) {
			return parsed{}, fmt.Errorf("%q is not a port or a range like 8000-9000", value)
		}
		return parsed{match: "DST-PORT," + value}, nil
	}
	return parsed{}, fmt.Errorf("unknown prefix %q", prefix)
}

func parseDomain(entry string) (parsed, error) {
	domain := strings.ToLower(entry)
	if !plainDomain.MatchString(domain) {
		return parsed{}, fmt.Errorf("%q is not a domain, IP or known notation", entry)
	}
	return domainRule("DOMAIN-SUFFIX", strings.TrimPrefix(strings.TrimPrefix(domain, "*."), "."))
}

func domainRule(kind, value string) (parsed, error) {
	value = strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(value, "*."), "."))
	if !domainName.MatchString(value) {
		return parsed{}, fmt.Errorf("%q is not a domain", value)
	}
	return parsed{match: kind + "," + value}, nil
}

func parseIP(value string) (parsed, bool) {
	prefix, err := netip.ParsePrefix(value)
	if err != nil {
		addr, addrErr := netip.ParseAddr(value)
		if addrErr != nil {
			return parsed{}, false
		}
		prefix = netip.PrefixFrom(addr, addr.BitLen())
	}
	kind := "IP-CIDR"
	if prefix.Addr().Is6() && !prefix.Addr().Is4In6() {
		kind = "IP-CIDR6"
	}
	return parsed{match: kind + "," + prefix.Masked().String(), params: ",no-resolve"}, true
}

func parseRaw(entry string) (parsed, error) {
	kind, rest, _ := strings.Cut(entry, ",")
	if !rawKeywords[kind] {
		return parsed{}, fmt.Errorf("unknown rule type %s", kind)
	}
	var params string
	for _, p := range []string{",no-resolve", ",src"} {
		if strings.HasSuffix(rest, p) {
			rest, params = strings.TrimSuffix(rest, p), p+params
		}
	}
	rest = strings.TrimSpace(rest)
	switch {
	case rest == "":
		return parsed{}, errEmpty
	case kind == "GEOSITE" || kind == "GEOIP":
		return parseEntry(strings.ToLower(kind) + ":" + rest)
	case logicalKeywords[kind] && !strings.HasSuffix(rest, ")"):
		return parsed{}, errTargetSet
	case !logicalKeywords[kind] && strings.Contains(rest, ","):
		return parsed{}, errTargetSet
	}
	return parsed{match: kind + "," + rest, params: params}, nil
}

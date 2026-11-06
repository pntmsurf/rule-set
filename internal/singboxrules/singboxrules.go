package singboxrules

import (
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"strings"
)

const DefaultSRSVersion = 2

type SRS struct {
	Version int       `json:"version"`
	Rules   []SRSRule `json:"rules"`
}

type SRSRule struct {
	DomainSuffix  []string `json:"domain_suffix,omitempty"`
	DomainKeyword []string `json:"domain_keyword,omitempty"`
	DomainRegex   []string `json:"domain_regex,omitempty"`
	Domain        []string `json:"domain,omitempty"`
	IPCIDR        []string `json:"ip_cidr,omitempty"`
}

func (r SRSRule) IsEmpty() bool {
	return len(r.DomainSuffix) == 0 &&
		len(r.DomainKeyword) == 0 &&
		len(r.DomainRegex) == 0 &&
		len(r.Domain) == 0 &&
		len(r.IPCIDR) == 0
}

func RuleFromDomainLines(lines []string) SRSRule {
	var rule SRSRule
	for _, line := range lines {
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if parts := strings.Fields(line); len(parts) > 0 {
			line = parts[0]
		}
		if line == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "full:"):
			rule.Domain = append(rule.Domain, strings.TrimPrefix(line, "full:"))
		case strings.HasPrefix(line, "regexp:"):
			rule.DomainRegex = append(rule.DomainRegex, strings.TrimPrefix(line, "regexp:"))
		case strings.HasPrefix(line, "keyword:"):
			rule.DomainKeyword = append(rule.DomainKeyword, strings.TrimPrefix(line, "keyword:"))
		case strings.HasPrefix(line, "domain:"):
			bare := strings.TrimPrefix(line, "domain:")
			rule.DomainSuffix = append(rule.DomainSuffix, strings.TrimPrefix(bare, "."))
		case strings.HasPrefix(line, "+."):
			bare := strings.TrimPrefix(line, "+.")
			if strings.Contains(bare, "*") {
				regexStr := "^(.*\\.)?" + strings.ReplaceAll(strings.ReplaceAll(bare, ".", `\.`), "*", ".*") + "$"
				rule.DomainRegex = append(rule.DomainRegex, regexStr)
			} else {
				rule.DomainSuffix = append(rule.DomainSuffix, bare)
			}
		case strings.HasPrefix(line, "."):
			bare := strings.TrimPrefix(line, ".")
			rule.DomainSuffix = append(rule.DomainSuffix, bare)
		case strings.HasPrefix(line, "^") || strings.HasPrefix(line, `(`) || strings.HasPrefix(line, `.+\.`):
			rule.DomainRegex = append(rule.DomainRegex, line)
		case strings.Contains(line, "*"):
			regexStr := "^" + strings.ReplaceAll(strings.ReplaceAll(line, ".", `\.`), "*", ".*") + "$"
			rule.DomainRegex = append(rule.DomainRegex, regexStr)
		default:
			rule.Domain = append(rule.Domain, line)
		}
	}
	return rule
}

func RuleFromIPLines(lines []string) SRSRule {
	var valid []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if _, _, err := net.ParseCIDR(line); err == nil {
			valid = append(valid, line)
			continue
		}
		if ip := net.ParseIP(line); ip != nil {
			if ip.To4() != nil {
				valid = append(valid, ip.String()+"/32")
			} else {
				valid = append(valid, ip.String()+"/128")
			}
		}
	}
	return SRSRule{IPCIDR: valid}
}

func BuildDomainSRS(domains []string, version int) (SRS, bool) {
	if version <= 0 {
		version = DefaultSRSVersion
	}
	rule := RuleFromDomainLines(domains)
	if rule.IsEmpty() {
		return SRS{}, false
	}
	return SRS{Version: version, Rules: []SRSRule{rule}}, true
}

func BuildIPSRS(ips []string, version int) (SRS, bool) {
	if version <= 0 {
		version = DefaultSRSVersion
	}
	rule := RuleFromIPLines(ips)
	if rule.IsEmpty() {
		return SRS{}, false
	}
	return SRS{Version: version, Rules: []SRSRule{rule}}, true
}

func Compile(singboxBin string, srs SRS, jsonPath, outSRSPath string) error {
	b, err := json.MarshalIndent(srs, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(jsonPath, b, 0644); err != nil {
		return err
	}
	defer os.Remove(jsonPath)

	cmd := exec.Command(singboxBin, "rule-set", "compile", jsonPath, "-o", outSRSPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

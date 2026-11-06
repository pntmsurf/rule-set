package singboxrules

import (
	"reflect"
	"testing"
)

func TestRuleFromDomainLines(t *testing.T) {
	lines := []string{
		"+.t.me",
		"domain:telegram.org",
		"full:oauth2.googleapis.com",
		"regexp:^[a-z]+$",
		"keyword:google",
		"chatgpt-async-webps-prod-*-*.webpubsub.azure.com",
		".chrome",
		"example.com",
	}

	rule := RuleFromDomainLines(lines)
	wantSuffix := []string{"t.me", "telegram.org", "chrome"}
	if !reflect.DeepEqual(rule.DomainSuffix, wantSuffix) {
		t.Fatalf("DomainSuffix = %v, want %v", rule.DomainSuffix, wantSuffix)
	}
}

func TestRuleFromIPLines(t *testing.T) {
	lines := []string{
		"149.154.160.0/20",
		"8.8.8.8",
		"2001:67c:4e8::/48",
		"2001:db8::1",
		"invalid",
	}
	rule := RuleFromIPLines(lines)
	want := []string{
		"149.154.160.0/20",
		"8.8.8.8/32",
		"2001:67c:4e8::/48",
		"2001:db8::1/128",
	}
	if !reflect.DeepEqual(rule.IPCIDR, want) {
		t.Fatalf("IPCIDR = %v, want %v", rule.IPCIDR, want)
	}
}

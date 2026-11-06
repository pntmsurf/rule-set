package mihomobuild

import (
	"reflect"
	"testing"
)

func TestSanitizeDomainLines(t *testing.T) {
	lines := []string{
		"+.t.me",
		"domain:telegram.org",
		"full:oauth2.googleapis.com",
		".chrome",
		"*.example.com",
		"chatgpt-async-webps-prod-*-*.webpubsub.azure.com",
		"regexp:^[a-z]+$",
		"^r+[0-9]+\\.googlevideo\\.com$",
		"keyword:google",
	}

	got := SanitizeDomainLines(lines)
	want := []string{
		"+.t.me",
		"+.telegram.org",
		"oauth2.googleapis.com",
		"+.chrome",
		"*.example.com",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SanitizeDomainLines = %v, want %v", got, want)
	}
}

func TestSanitizeIPLines(t *testing.T) {
	lines := []string{
		"149.154.160.0/20",
		"8.8.8.8",
		"2001:67c:4e8::/48",
		"invalid",
	}
	got := SanitizeIPLines(lines)
	want := []string{
		"149.154.160.0/20",
		"8.8.8.8/32",
		"2001:67c:4e8::/48",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SanitizeIPLines = %v, want %v", got, want)
	}
}

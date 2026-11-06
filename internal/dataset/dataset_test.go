package dataset

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseTOMLFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "telegram.toml")
	content := `[domains]
# https://github.com/itdoginfo/allow-domains/blob/main/Services/telegram.lst
+.t.me
+.telegram.org

[cidr]
149.154.160.0/20
2001:67c:4e8::/48
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cat, err := ParseTOMLFile(path, "shared", "telegram")
	if err != nil {
		t.Fatalf("ParseTOMLFile error: %v", err)
	}
	if cat.TagName != "telegram" {
		t.Fatalf("TagName = %s, want telegram", cat.TagName)
	}
	wantDomains := []string{"+.t.me", "+.telegram.org"}
	if !reflect.DeepEqual(cat.Domains, wantDomains) {
		t.Fatalf("Domains = %v, want %v", cat.Domains, wantDomains)
	}
	wantCIDR := []string{"149.154.160.0/20", "2001:67c:4e8::/48"}
	if !reflect.DeepEqual(cat.CIDR, wantCIDR) {
		t.Fatalf("CIDR = %v, want %v", cat.CIDR, wantCIDR)
	}
}

func TestRepositoryResolveAndManifest(t *testing.T) {
	tmp := t.TempDir()
	sharedDir := filepath.Join(tmp, "shared")
	ruDir := filepath.Join(tmp, "ru")
	usDir := filepath.Join(tmp, "us")
	for _, d := range []string{sharedDir, ruDir, usDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}

	os.WriteFile(filepath.Join(sharedDir, "telegram.toml"), []byte(`[domains]
+.t.me

[cidr]
149.154.160.0/20
2001:67c:4e8::/48
`), 0644)

	os.WriteFile(filepath.Join(sharedDir, "whatsapp.toml"), []byte(`[domains]
+.wa.me

[cidr]
`), 0644)

	os.WriteFile(filepath.Join(sharedDir, "messengers.toml"), []byte(`[domains]
@telegram
@whatsapp

[cidr]
@telegram
`), 0644)

	os.WriteFile(filepath.Join(sharedDir, "claude.toml"), []byte(`[domains]
+.claude.ai

[cidr]
`), 0644)

	os.WriteFile(filepath.Join(ruDir, "neuro.toml"), []byte(`[domains]
+.giga.chat
+.alice.yandex.ru

[cidr]
`), 0644)

	os.WriteFile(filepath.Join(ruDir, "ru.toml"), []byte(`[domains]
+.ru

[cidr]
`), 0644)

	os.WriteFile(filepath.Join(ruDir, "manifest.yaml"), []byte(`shared:
  - telegram
  - messengers
`), 0644)

	os.WriteFile(filepath.Join(usDir, "gov.toml"), []byte(`[domains]
+.gov

[cidr]
`), 0644)

	os.WriteFile(filepath.Join(usDir, "manifest.yaml"), []byte(`shared:
  - telegram
  - claude
`), 0644)

	repo, err := LoadRepository(tmp)
	if err != nil {
		t.Fatalf("LoadRepository error: %v", err)
	}

	ruCats, err := repo.ResolveRegionCategories("ru")
	if err != nil {
		t.Fatalf("ResolveRegionCategories(ru) error: %v", err)
	}

	gotTags := map[string]*Category{}
	for _, c := range ruCats {
		gotTags[c.TagName] = c
	}

	if _, ok := gotTags["claude"]; ok {
		t.Fatalf("claude should not be in ru categories")
	}
	for _, expected := range []string{"telegram", "messengers", "ru-neuro", "ru"} {
		if _, ok := gotTags[expected]; !ok {
			t.Fatalf("missing expected tag %q in ru categories", expected)
		}
	}

	msg := gotTags["messengers"]
	wantMsgDomains := []string{"+.t.me", "+.wa.me"}
	if !reflect.DeepEqual(msg.Domains, wantMsgDomains) {
		t.Fatalf("messengers domains = %v, want %v", msg.Domains, wantMsgDomains)
	}
	wantMsgIPs := []string{"149.154.160.0/20", "2001:67c:4e8::/48"}
	if !reflect.DeepEqual(msg.IPs, wantMsgIPs) {
		t.Fatalf("messengers IPs = %v, want %v", msg.IPs, wantMsgIPs)
	}
}

func TestCycleDetection(t *testing.T) {
	tmp := t.TempDir()
	sharedDir := filepath.Join(tmp, "shared")
	os.MkdirAll(sharedDir, 0755)

	os.WriteFile(filepath.Join(sharedDir, "a.toml"), []byte("[domains]\n@b\n\n[cidr]\n"), 0644)
	os.WriteFile(filepath.Join(sharedDir, "b.toml"), []byte("[domains]\n@a\n\n[cidr]\n"), 0644)

	repo, err := LoadRepository(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ResolveAllCategories(); err == nil {
		t.Fatal("expected cycle error, got nil")
	}
}

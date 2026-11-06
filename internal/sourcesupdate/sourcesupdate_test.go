package sourcesupdate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTargets(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "sources.yaml")
	content := `shared:
  ads:
    domains:
      type: adguard
      sources:
        - https://small.oisd.nl
ru:
  blocked:
    domains:
      type: plain
      sources:
        - https://community.antifilter.download/list/domains.lst
    ips:
      type: cidr-plain
      sources:
        - https://community.antifilter.download/list/community.lst
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	targets, err := LoadTargets(cfgPath)
	if err != nil {
		t.Fatalf("LoadTargets error: %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("len(targets) = %d, want 2", len(targets))
	}
}

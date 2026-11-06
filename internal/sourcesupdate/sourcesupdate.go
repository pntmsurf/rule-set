package sourcesupdate

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/pntmsurf/rule-set/internal/dataset"
)

type Config map[string]map[string]CategoryConfig

type CategoryConfig struct {
	Domains *SourceSpec `yaml:"domains,omitempty"`
	IPs     *SourceSpec `yaml:"ips,omitempty"`
}

type SourceSpec struct {
	Type    string   `yaml:"type"`
	Sources []string `yaml:"sources"`
}

type Target struct {
	Scope    string
	Category string
	File     string
	Domains  *SourceSpec
	IPs      *SourceSpec
}

func LoadTargets(configPath string) ([]Target, error) {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", configPath, err)
	}

	var config Config
	if err := yaml.Unmarshal(raw, &config); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", configPath, err)
	}

	var targets []Target
	for scope, categories := range config {
		for categoryName, catCfg := range categories {
			targets = append(targets, Target{
				Scope:    scope,
				Category: categoryName,
				File:     filepath.Join("data", scope, "pending", categoryName+".toml"),
				Domains:  catCfg.Domains,
				IPs:      catCfg.IPs,
			})
		}
	}

	sort.Slice(targets, func(i, j int) bool {
		return targets[i].File < targets[j].File
	})

	return targets, nil
}

var httpClient = &http.Client{}

func fetchURL(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/plain,text/html,application/octet-stream,*/*")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

var (
	adguardDomainRE    = regexp.MustCompile(`^\|\|([a-zA-Z0-9][a-zA-Z0-9\-.]*\.[a-zA-Z]{2,})\^(\$|$)`)
	domainLooksValidRE = regexp.MustCompile(`^[a-z0-9*]([a-z0-9\-*]*[a-z0-9*])?(\.[a-z0-9*]([a-z0-9\-*]*[a-z0-9*])?)+$`)
	cidrLooksValidRE   = regexp.MustCompile(`^[0-9a-fA-F:.]+/[0-9]{1,3}$`)
	clashPayloadLineRE = regexp.MustCompile(`(?i)^-\s*(DOMAIN|DOMAIN-SUFFIX)\s*,\s*([^,\s]+)`)
)

func parseAdGuard(data []byte) map[string]bool {
	out := make(map[string]bool)
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	sc.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "!") || strings.HasPrefix(line, "[") || strings.HasPrefix(line, "@@") || !strings.HasPrefix(line, "||") {
			continue
		}

		if m := adguardDomainRE.FindStringSubmatch(line); m != nil {
			domain := strings.ToLower(strings.TrimPrefix(m[1], "*."))
			out[domain] = true
		}
	}
	return out
}

func parsePlainDomainList(data []byte) map[string]bool {
	out := make(map[string]bool)
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	sc.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for sc.Scan() {
		line := strings.ToLower(strings.TrimSpace(sc.Text()))
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") || strings.HasPrefix(line, ";") {
			continue
		}
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if line == "" {
			continue
		}

		line = strings.TrimPrefix(line, "*.")
		line = strings.TrimSuffix(line, ".")

		if domainLooksValidRE.MatchString(line) {
			out[line] = true
		}
	}
	return out
}

func parseCIDRPlain(data []byte) map[string]bool {
	out := make(map[string]bool)
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	sc.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		for _, tok := range strings.Fields(line) {
			if _, _, err := net.ParseCIDR(tok); err == nil && cidrLooksValidRE.MatchString(tok) {
				out[tok] = true
			}
		}
	}
	return out
}

func parseClashPayload(data []byte) map[string]bool {
	out := make(map[string]bool)
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	sc.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if m := clashPayloadLineRE.FindStringSubmatch(line); m != nil {
			domain := strings.ToLower(strings.TrimSuffix(m[2], "."))
			if domainLooksValidRE.MatchString(domain) {
				out[domain] = true
			}
		}
	}
	return out
}

func parseHosts(data []byte) map[string]bool {
	out := make(map[string]bool)
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	sc.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		if ip := net.ParseIP(fields[0]); ip != nil {
			domain := strings.ToLower(fields[1])
			if domain == "localhost" || domain == "localhost.localdomain" {
				continue
			}
			if domainLooksValidRE.MatchString(domain) {
				out[domain] = true
			}
		}
	}
	return out
}

func parseSourceData(url, sourceType string) (map[string]bool, error) {
	data, err := fetchURL(url)
	if err != nil {
		return nil, err
	}

	switch sourceType {
	case "adguard":
		return parseAdGuard(data), nil
	case "plain":
		return parsePlainDomainList(data), nil
	case "hosts":
		return parseHosts(data), nil
	case "cidr-plain":
		return parseCIDRPlain(data), nil
	case "clash-payload":
		return parseClashPayload(data), nil
	default:
		return nil, fmt.Errorf("unknown type: %q", sourceType)
	}
}

func normalizeDomainForCompare(line string) string {
	line = strings.TrimSpace(line)
	for _, comment := range []string{"#", "//"} {
		if idx := strings.Index(line, " "+comment); idx > 0 {
			line = line[:idx]
		}
	}
	line = strings.TrimSpace(line)
	for _, prefix := range []string{"full:", "regexp:", "keyword:", "domain:"} {
		line = strings.TrimPrefix(line, prefix)
	}
	line = strings.TrimPrefix(line, "+.")
	line = strings.TrimPrefix(line, ".")
	return strings.ToLower(strings.TrimSpace(line))
}

func loadExistingTOML(path, scope, name string) (*dataset.RawCategory, map[string]bool, map[string]bool, error) {
	domSet := make(map[string]bool)
	ipSet := make(map[string]bool)

	raw, err := dataset.ParseTOMLFile(path, scope, name)
	if os.IsNotExist(err) {
		return &dataset.RawCategory{
			Scope:   scope,
			Name:    name,
			TagName: dataset.CanonicalTagName(scope, name),
			Path:    path,
		}, domSet, ipSet, nil
	}
	if err != nil {
		return nil, nil, nil, err
	}

	for _, d := range raw.Domains {
		if !strings.HasPrefix(d, "@") {
			domSet[normalizeDomainForCompare(d)] = true
		}
	}
	for _, ip := range raw.CIDR {
		if !strings.HasPrefix(ip, "@") {
			ipSet[strings.TrimSpace(ip)] = true
		}
	}
	return raw, domSet, ipSet, nil
}

func fetchPromotedEntries(scopeDir string) (map[string]bool, map[string]bool, error) {
	domOut := make(map[string]bool)
	ipOut := make(map[string]bool)

	entries, err := os.ReadDir(scopeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return domOut, ipOut, nil
		}
		return nil, nil, err
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		_, dSet, iSet, err := loadExistingTOML(filepath.Join(scopeDir, e.Name()), filepath.Base(scopeDir), strings.TrimSuffix(e.Name(), ".toml"))
		if err != nil {
			return nil, nil, err
		}
		for k := range dSet {
			domOut[k] = true
		}
		for k := range iSet {
			ipOut[k] = true
		}
	}
	return domOut, ipOut, nil
}

func collectFresh(spec *SourceSpec, existing, categorized map[string]bool) ([]string, []string) {
	if spec == nil || len(spec.Sources) == 0 {
		return nil, nil
	}

	combined := make(map[string]bool)
	var labels []string
	for _, url := range spec.Sources {
		fmt.Printf("  fetch %s\n", url)
		found, err := parseSourceData(url, spec.Type)
		if err != nil {
			fmt.Printf("    skip: %v\n", err)
			continue
		}
		fmt.Printf("    got: %d\n", len(found))
		for k := range found {
			combined[k] = true
		}
		labels = append(labels, url)
	}

	var fresh []string
	for k := range combined {
		if existing[k] || categorized[k] {
			continue
		}
		fresh = append(fresh, k)
	}
	sort.Strings(fresh)
	fmt.Printf("  stats (%s): fetch %d, exist %d, categ %d, new %d\n",
		spec.Type, len(combined), len(existing), len(categorized), len(fresh))
	return fresh, labels
}

func processTarget(t Target, dryRun bool) error {
	raw, existDom, existIP, err := loadExistingTOML(t.File, t.Scope, t.Category)
	if err != nil {
		return fmt.Errorf("read %s: %w", t.File, err)
	}

	scopeDir := filepath.Dir(filepath.Dir(t.File))
	catDom, catIP, err := fetchPromotedEntries(scopeDir)
	if err != nil {
		return fmt.Errorf("categorized for %s: %w", t.File, err)
	}

	if t.Scope != dataset.SharedScope {
		sharedDir := filepath.Join(filepath.Dir(scopeDir), dataset.SharedScope)
		sDom, sIP, err := fetchPromotedEntries(sharedDir)
		if err == nil {
			for k := range sDom {
				catDom[k] = true
			}
			for k := range sIP {
				catIP[k] = true
			}
		}
	}

	freshDom, domLabels := collectFresh(t.Domains, existDom, catDom)
	freshIPs, ipLabels := collectFresh(t.IPs, existIP, catIP)

	if len(freshDom) == 0 && len(freshIPs) == 0 {
		return nil
	}

	if dryRun {
		for i, d := range freshDom {
			if i >= 10 {
				fmt.Printf("    ... and %d more domains\n", len(freshDom)-10)
				break
			}
			fmt.Println("    + domain:", d)
		}
		for i, ip := range freshIPs {
			if i >= 10 {
				fmt.Printf("    ... and %d more CIDRs\n", len(freshIPs)-10)
				break
			}
			fmt.Println("    + cidr:", ip)
		}
		return nil
	}

	for _, d := range freshDom {
		raw.Domains = append(raw.Domains, "+."+d)
	}
	raw.CIDR = append(raw.CIDR, freshIPs...)

	if err := os.MkdirAll(filepath.Dir(t.File), 0755); err != nil {
		return err
	}

	f, err := os.Create(t.File)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	fmt.Fprintln(w, "[domains]")
	for _, u := range domLabels {
		fmt.Fprintf(w, "# %s\n", u)
	}
	for _, d := range raw.Domains {
		fmt.Fprintln(w, d)
	}
	fmt.Fprintln(w, "\n[cidr]")
	for _, u := range ipLabels {
		fmt.Fprintf(w, "# %s\n", u)
	}
	for _, ip := range raw.CIDR {
		fmt.Fprintln(w, ip)
	}

	if err := w.Flush(); err != nil {
		return err
	}

	fmt.Printf("  updated %s (+%d domains, +%d CIDRs)\n", t.File, len(freshDom), len(freshIPs))
	return nil
}

func Run(targets []Target, dryRun bool, only string) {
	for _, t := range targets {
		if only != "" && !strings.Contains(t.File, only) {
			continue
		}
		fmt.Printf("\n=== %s ===\n", t.File)
		if err := processTarget(t, dryRun); err != nil {
			fmt.Printf("  error: %v\n", err)
		}
	}
}

package dataset

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const SharedScope = "shared"

// RawCategory represents an unexpanded .toml category file on disk.
type RawCategory struct {
	Scope   string // "shared" or region code ("ru", "cn", "ir", "us", "gb", etc.)
	Name    string // filename without .toml (e.g., "telegram", "gov", "ru")
	TagName string // canonical tag/rule-set name: "telegram" for shared, "ru-gov" or "ru" for regional
	Path    string
	Domains []string
	CIDR    []string
}

// Category represents a fully resolved category with all @ references expanded and deduplicated.
type Category struct {
	Scope   string
	Name    string
	TagName string
	Path    string
	Domains []string
	IPs     []string
}

// Manifest represents data/<region>/manifest.yaml.
type Manifest struct {
	Shared []string `yaml:"shared"`
}

// Repository holds all loaded shared and regional categories and manifests.
type Repository struct {
	DataDir   string
	Shared    map[string]*RawCategory
	Regions   map[string]map[string]*RawCategory
	Manifests map[string]Manifest
	ByTag     map[string]*RawCategory
}

// CanonicalTagName returns the canonical rule-set / geosite / geoip tag name.
// For shared scope: Name ("telegram", "neuro").
// For regional scope: if Name == Scope (e.g. data/ru/ru.toml), returns "ru";
// otherwise returns "<region>-<name>" (e.g. "ru-gov", "ru-neuro").
func CanonicalTagName(scope, name string) string {
	scope = strings.ToLower(strings.TrimSpace(scope))
	name = strings.ToLower(strings.TrimSpace(name))
	if scope == "" || scope == SharedScope {
		return name
	}
	if name == scope || strings.HasPrefix(name, scope+"-") {
		return name
	}
	return scope + "-" + name
}

// ParseTOMLFile parses a rule-set .toml file with [domains] and [cidr] sections
// (also accepting [ip], [ips], [ipv4], [ipv6] as aliases for [cidr]).
func ParseTOMLFile(path, scope, name string) (*RawCategory, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cat := &RawCategory{
		Scope:   scope,
		Name:    name,
		TagName: CanonicalTagName(scope, name),
		Path:    path,
	}

	section := ""
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 16*1024*1024)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Strip inline comments preceded by whitespace
		if idx := strings.Index(line, " #"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			continue
		}

		switch section {
		case "domains", "domain":
			cat.Domains = append(cat.Domains, line)
		case "cidr", "ip", "ips", "ipv4", "ipv6":
			cat.CIDR = append(cat.CIDR, line)
		case "":
			if looksLikeIPOrCIDR(line) {
				cat.CIDR = append(cat.CIDR, line)
			} else {
				cat.Domains = append(cat.Domains, line)
			}
		default:
			return nil, fmt.Errorf("%s: unknown section [%s]", path, section)
		}
	}

	if err := sc.Err(); err != nil {
		return nil, err
	}
	return cat, nil
}

func looksLikeIPOrCIDR(s string) bool {
	if strings.HasPrefix(s, "@") {
		return false
	}
	if _, _, err := net.ParseCIDR(s); err == nil {
		return true
	}
	return net.ParseIP(s) != nil
}

// LoadManifest reads data/<region>/manifest.yaml.
func LoadManifest(path string) (Manifest, error) {
	var m Manifest
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return m, nil
		}
		return m, err
	}
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return m, fmt.Errorf("parse %s: %w", path, err)
	}
	for i := range m.Shared {
		m.Shared[i] = strings.ToLower(strings.TrimSpace(strings.TrimSuffix(m.Shared[i], ".toml")))
	}
	return m, nil
}

// LoadRepository scans data/shared and all regional directories in data/.
func LoadRepository(dataDir string) (*Repository, error) {
	repo := &Repository{
		DataDir:   dataDir,
		Shared:    make(map[string]*RawCategory),
		Regions:   make(map[string]map[string]*RawCategory),
		Manifests: make(map[string]Manifest),
		ByTag:     make(map[string]*RawCategory),
	}

	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirName := entry.Name()
		if dirName == "pending" || strings.HasPrefix(dirName, ".") {
			continue
		}

		subDir := filepath.Join(dataDir, dirName)
		if dirName == SharedScope {
			cats, err := loadDirCategories(subDir, SharedScope)
			if err != nil {
				return nil, err
			}
			for _, c := range cats {
				repo.Shared[c.Name] = c
				repo.ByTag[c.TagName] = c
			}
			continue
		}

		region := strings.ToLower(dirName)
		cats, err := loadDirCategories(subDir, region)
		if err != nil {
			return nil, err
		}
		repo.Regions[region] = make(map[string]*RawCategory)
		for _, c := range cats {
			repo.Regions[region][c.Name] = c
			repo.ByTag[c.TagName] = c
		}

		manifestPath := filepath.Join(subDir, "manifest.yaml")
		manifest, err := LoadManifest(manifestPath)
		if err != nil {
			return nil, err
		}
		repo.Manifests[region] = manifest
	}

	return repo, nil
}

func loadDirCategories(dir, scope string) ([]*RawCategory, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var out []*RawCategory
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fname := e.Name()
		if fname == "manifest.yaml" || fname == "manifest.yml" || strings.HasPrefix(fname, ".") {
			continue
		}
		ext := strings.ToLower(filepath.Ext(fname))
		if ext != ".toml" && ext != "" && ext != ".list" && ext != ".txt" {
			continue
		}
		name := strings.TrimSuffix(fname, filepath.Ext(fname))
		if ext == "" {
			name = fname
		}
		name = strings.ToLower(name)

		cat, err := ParseTOMLFile(filepath.Join(dir, fname), scope, name)
		if err != nil {
			return nil, err
		}
		out = append(out, cat)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].TagName < out[j].TagName
	})
	return out, nil
}

// RegionNames returns sorted list of region codes present in data/.
func (r *Repository) RegionNames() []string {
	var regions []string
	for reg := range r.Regions {
		regions = append(regions, reg)
	}
	sort.Strings(regions)
	return regions
}

// ResolveCategory expands all @ references in [domains] and [cidr] for a given RawCategory.
func (r *Repository) ResolveCategory(raw *RawCategory) (*Category, error) {
	domVisited := make(map[string]bool)
	domains, err := r.expandDomains(raw, domVisited)
	if err != nil {
		return nil, err
	}

	ipVisited := make(map[string]bool)
	ips, err := r.expandCIDR(raw, ipVisited)
	if err != nil {
		return nil, err
	}

	return &Category{
		Scope:   raw.Scope,
		Name:    raw.Name,
		TagName: raw.TagName,
		Path:    raw.Path,
		Domains: dedupPreserveOrder(domains),
		IPs:     dedupPreserveOrder(ips),
	}, nil
}

func (r *Repository) findRefTarget(from *RawCategory, ref string) (*RawCategory, error) {
	refName := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(ref, "@")))
	refName = strings.TrimSuffix(refName, ".toml")
	if refName == "" {
		return nil, fmt.Errorf("%s: empty @ reference", from.Path)
	}

	if strings.Contains(refName, "/") {
		parts := strings.SplitN(refName, "/", 2)
		scope, name := parts[0], parts[1]
		if scope == SharedScope {
			if target, ok := r.Shared[name]; ok {
				return target, nil
			}
		} else if regMap, ok := r.Regions[scope]; ok {
			if target, ok := regMap[name]; ok {
				return target, nil
			}
		}
		return nil, fmt.Errorf("%s: reference @%s not found", from.Path, refName)
	}

	if target, ok := r.Shared[refName]; ok {
		return target, nil
	}

	if from.Scope != SharedScope {
		if regMap, ok := r.Regions[from.Scope]; ok {
			if target, ok := regMap[refName]; ok {
				return target, nil
			}
		}
	}

	if target, ok := r.ByTag[refName]; ok {
		return target, nil
	}

	return nil, fmt.Errorf("%s: reference @%s not found", from.Path, refName)
}

func (r *Repository) expandDomains(raw *RawCategory, visited map[string]bool) ([]string, error) {
	key := raw.Scope + ":" + raw.Name
	if visited[key] {
		return nil, fmt.Errorf("cyclic @ reference detected involving %s (%s)", raw.TagName, raw.Path)
	}
	visited[key] = true
	defer func() { delete(visited, key) }()

	var out []string
	for _, entry := range raw.Domains {
		if strings.HasPrefix(entry, "@") {
			target, err := r.findRefTarget(raw, entry)
			if err != nil {
				return nil, err
			}
			sub, err := r.expandDomains(target, visited)
			if err != nil {
				return nil, err
			}
			out = append(out, sub...)
		} else {
			out = append(out, entry)
		}
	}
	return out, nil
}

func (r *Repository) expandCIDR(raw *RawCategory, visited map[string]bool) ([]string, error) {
	key := raw.Scope + ":" + raw.Name
	if visited[key] {
		return nil, fmt.Errorf("cyclic @ reference detected involving %s (%s)", raw.TagName, raw.Path)
	}
	visited[key] = true
	defer func() { delete(visited, key) }()

	var out []string
	for _, entry := range raw.CIDR {
		if strings.HasPrefix(entry, "@") {
			target, err := r.findRefTarget(raw, entry)
			if err != nil {
				return nil, err
			}
			sub, err := r.expandCIDR(target, visited)
			if err != nil {
				return nil, err
			}
			out = append(out, sub...)
		} else {
			out = append(out, entry)
		}
	}
	return out, nil
}

// ResolveRegionCategories returns all resolved categories for a specific region:
// 1. All shared categories explicitly listed in data/<region>/manifest.yaml (whitelist)
// 2. All regional categories in data/<region>/*.toml
func (r *Repository) ResolveRegionCategories(region string) ([]*Category, error) {
	region = strings.ToLower(strings.TrimSpace(region))
	regMap, ok := r.Regions[region]
	if !ok {
		return nil, fmt.Errorf("region %q not found in %s", region, r.DataDir)
	}

	manifest := r.Manifests[region]
	seenTags := make(map[string]bool)
	var out []*Category

	for _, sharedName := range manifest.Shared {
		raw, ok := r.Shared[sharedName]
		if !ok {
			return nil, fmt.Errorf("region %q manifest references unknown shared category %q", region, sharedName)
		}
		if seenTags[raw.TagName] {
			continue
		}
		resolved, err := r.ResolveCategory(raw)
		if err != nil {
			return nil, err
		}
		seenTags[resolved.TagName] = true
		out = append(out, resolved)
	}

	for _, name := range regMapKeysSorted(regMap) {
		raw := regMap[name]
		if seenTags[raw.TagName] {
			continue
		}
		resolved, err := r.ResolveCategory(raw)
		if err != nil {
			return nil, err
		}
		seenTags[resolved.TagName] = true
		out = append(out, resolved)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].TagName < out[j].TagName
	})
	return out, nil
}

// ResolveAllCategories returns all resolved categories across shared/ and all regions,
// sorted deterministically by TagName.
func (r *Repository) ResolveAllCategories() ([]*Category, error) {
	var rawList []*RawCategory
	for _, c := range r.Shared {
		rawList = append(rawList, c)
	}
	for _, regMap := range r.Regions {
		for _, c := range regMap {
			rawList = append(rawList, c)
		}
	}
	sort.Slice(rawList, func(i, j int) bool {
		return rawList[i].TagName < rawList[j].TagName
	})

	var out []*Category
	seen := make(map[string]bool)
	for _, raw := range rawList {
		if seen[raw.TagName] {
			continue
		}
		seen[raw.TagName] = true
		resolved, err := r.ResolveCategory(raw)
		if err != nil {
			return nil, err
		}
		out = append(out, resolved)
	}
	return out, nil
}

func regMapKeysSorted(m map[string]*RawCategory) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func dedupPreserveOrder(in []string) []string {
	seen := make(map[string]bool, len(in))
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

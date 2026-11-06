package mihomobuild

import (
	"net"
	"os"
	"os/exec"
	"strings"
)

type Kind string

const (
	KindDomain Kind = "domain"
	KindIPCIDR Kind = "ipcidr"
)

// SanitizeDomainLines filters and normalizes domain rules into Mihomo `behavior: domain` text format.
// Mihomo's binary .mrs format supports exact domains, `+.` / `.` suffix wildcards, and full-label
// `*` wildcards (e.g. `*.example.com`). It rejects `regexp:`, `keyword:`, and partial-label `*` wildcards
// (e.g. `foo-*-bar.example.com`, see MetaCubeX/mihomo#3116).
func SanitizeDomainLines(lines []string) []string {
	var out []string
	seen := make(map[string]bool, len(lines))

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

		// Skip explicit regex and keyword prefixes first
		if strings.HasPrefix(line, "regexp:") ||
			strings.HasPrefix(line, "keyword:") ||
			strings.HasPrefix(line, "^") ||
			strings.HasPrefix(line, "(") ||
			strings.HasPrefix(line, `.+\.`) {
			continue
		}

		hasPlusPrefix := false
		switch {
		case strings.HasPrefix(line, "+."):
			hasPlusPrefix = true
			line = strings.TrimPrefix(line, "+.")
		case strings.HasPrefix(line, "domain:"):
			hasPlusPrefix = true
			line = strings.TrimPrefix(strings.TrimPrefix(line, "domain:"), ".")
		case strings.HasPrefix(line, "."):
			hasPlusPrefix = true
			line = strings.TrimPrefix(line, ".")
		case strings.HasPrefix(line, "full:"):
			line = strings.TrimPrefix(line, "full:")
		}

		// After stripping valid leading `+.`, check for regex metacharacters
		if strings.ContainsAny(line, `\^$+?()[]{}|`) {
			continue
		}

		if hasPlusPrefix {
			line = "+." + line
		}

		if !isValidMihomoDomain(line) {
			continue
		}

		if !seen[line] {
			seen[line] = true
			out = append(out, line)
		}
	}
	return out
}

// isValidMihomoDomain checks that any `*` wildcard occupies an entire domain label
// as required by Mihomo >= v1.19.30.
func isValidMihomoDomain(d string) bool {
	bare := strings.TrimPrefix(d, "+.")
	bare = strings.TrimPrefix(bare, ".")
	if bare == "" {
		return false
	}
	labels := strings.Split(bare, ".")
	for _, label := range labels {
		if label == "" {
			return false
		}
		if strings.Contains(label, "*") && label != "*" {
			return false
		}
	}
	return true
}

// SanitizeIPLines filters and normalizes IP/CIDR lines for Mihomo `behavior: ipcidr`.
func SanitizeIPLines(lines []string) []string {
	var out []string
	seen := make(map[string]bool, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var normalized string
		if _, _, err := net.ParseCIDR(line); err == nil {
			normalized = line
		} else if ip := net.ParseIP(line); ip != nil {
			if ip.To4() != nil {
				normalized = ip.String() + "/32"
			} else {
				normalized = ip.String() + "/128"
			}
		} else {
			continue
		}

		if !seen[normalized] {
			seen[normalized] = true
			out = append(out, normalized)
		}
	}
	return out
}

func ConvertLines(mihomoBin string, kind Kind, lines []string, tmpPath, outPath string) error {
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		return err
	}
	defer os.Remove(tmpPath)

	return Convert(mihomoBin, kind, tmpPath, outPath)
}

func Convert(mihomoBin string, kind Kind, inPath, outPath string) error {
	cmd := exec.Command(mihomoBin, "convert-ruleset", string(kind), "text", inPath, outPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

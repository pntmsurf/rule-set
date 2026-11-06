package xraygeo

import (
	"net"
	"os"
	"strings"

	"github.com/v2fly/v2ray-core/v4/app/router"
	"google.golang.org/protobuf/proto"

	"github.com/pntmsurf/rule-set/internal/dataset"
)

func BuildGeoSiteFromCategories(categories []*dataset.Category) (*router.GeoSiteList, error) {
	var list router.GeoSiteList
	for _, cat := range categories {
		if len(cat.Domains) == 0 {
			continue
		}

		geosite := &router.GeoSite{
			CountryCode: strings.ToUpper(cat.TagName),
		}

		for _, line := range cat.Domains {
			d, ok := ParseDomainLine(line)
			if !ok {
				continue
			}
			geosite.Domain = append(geosite.Domain, d)
		}

		if len(geosite.Domain) > 0 {
			list.Entry = append(list.Entry, geosite)
		}
	}
	return &list, nil
}

func ParseDomainLine(line string) (*router.Domain, bool) {
	if idx := strings.Index(line, "#"); idx >= 0 {
		line = strings.TrimSpace(line[:idx])
	}
	if line == "" {
		return nil, false
	}

	var attrs []*router.Domain_Attribute
	parts := strings.Fields(line)
	if len(parts) > 1 {
		line = parts[0]
		for _, tok := range parts[1:] {
			if strings.HasPrefix(tok, "@") && len(tok) > 1 {
				attrs = append(attrs, &router.Domain_Attribute{
					Key:        strings.ToLower(tok[1:]),
					TypedValue: &router.Domain_Attribute_BoolValue{BoolValue: true},
				})
			}
		}
	}

	domainType := router.Domain_Domain
	switch {
	case strings.HasPrefix(line, "+."):
		domainType = router.Domain_Domain
		line = strings.TrimPrefix(line, "+.")
	case strings.HasPrefix(line, "domain:"):
		domainType = router.Domain_Domain
		line = strings.TrimPrefix(line, "domain:")
	case strings.HasPrefix(line, "full:"):
		domainType = router.Domain_Full
		line = strings.TrimPrefix(line, "full:")
	case strings.HasPrefix(line, "regexp:"):
		domainType = router.Domain_Regex
		line = strings.TrimPrefix(line, "regexp:")
	case strings.HasPrefix(line, "keyword:"):
		domainType = router.Domain_Plain
		line = strings.TrimPrefix(line, "keyword:")
	case strings.HasPrefix(line, "^") || strings.HasPrefix(line, `(`) || strings.HasPrefix(line, `.+\.`):
		domainType = router.Domain_Regex
	case strings.HasPrefix(line, "."):
		domainType = router.Domain_Domain
		line = strings.TrimPrefix(line, ".")
	}

	if strings.Contains(line, "*") && domainType == router.Domain_Domain {
		domainType = router.Domain_Regex
		line = "^" + strings.ReplaceAll(strings.ReplaceAll(line, ".", `\.`), "*", ".*") + "$"
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return nil, false
	}

	return &router.Domain{
		Type:      domainType,
		Value:     line,
		Attribute: attrs,
	}, true
}

func BuildGeoIPFromCategories(categories []*dataset.Category) (*router.GeoIPList, error) {
	var list router.GeoIPList
	for _, cat := range categories {
		if len(cat.IPs) == 0 {
			continue
		}

		geoip := &router.GeoIP{
			CountryCode: strings.ToUpper(cat.TagName),
		}

		for _, line := range cat.IPs {
			cidr, ok := ParseCIDRLine(line)
			if !ok {
				continue
			}
			geoip.Cidr = append(geoip.Cidr, cidr)
		}

		if len(geoip.Cidr) > 0 {
			list.Entry = append(list.Entry, geoip)
		}
	}
	return &list, nil
}

func ParseCIDRLine(line string) (*router.CIDR, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return nil, false
	}

	ip, ipnet, err := net.ParseCIDR(line)
	if err != nil {
		ip = net.ParseIP(line)
		if ip == nil {
			return nil, false
		}
		if ip4 := ip.To4(); ip4 != nil {
			ip = ip4
			ipnet = &net.IPNet{IP: ip4, Mask: net.CIDRMask(32, 32)}
		} else {
			ip16 := ip.To16()
			ip = ip16
			ipnet = &net.IPNet{IP: ip16, Mask: net.CIDRMask(128, 128)}
		}
	}

	ones, _ := ipnet.Mask.Size()
	networkIP := ipnet.IP
	if networkIP == nil {
		networkIP = ip
	}
	ipBytes := networkIP.To4()
	if ipBytes == nil {
		ipBytes = networkIP.To16()
	}

	return &router.CIDR{
		Ip:     ipBytes,
		Prefix: uint32(ones),
	}, true
}

func WriteGeoSite(path string, list *router.GeoSiteList) error {
	data, err := proto.Marshal(list)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func WriteGeoIP(path string, list *router.GeoIPList) error {
	data, err := proto.Marshal(list)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

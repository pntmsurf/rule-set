package xraygeo

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/v2fly/v2ray-core/v4/app/router"
	"google.golang.org/protobuf/proto"

	"github.com/pntmsurf/rule-set/internal/dataset"
)

func TestBuildGeoSiteFromCategories(t *testing.T) {
	cats := []*dataset.Category{
		{
			Scope:   "shared",
			Name:    "telegram",
			TagName: "telegram",
			Domains: []string{
				"+.t.me",
				"domain:telegram.org",
				"full:core.telegram.org",
				"regexp:^.*\\.telegra\\.ph$",
				"keyword:tgproxy",
				"*.tdesktop.com @ads",
			},
		},
		{
			Scope:   "ru",
			Name:    "neuro",
			TagName: "ru-neuro",
			Domains: []string{
				"+.giga.chat",
			},
		},
		{
			Scope:   "ru",
			Name:    "porn",
			TagName: "ru-porn",
			Domains: nil,
		},
	}

	list, err := BuildGeoSiteFromCategories(cats)
	if err != nil {
		t.Fatalf("BuildGeoSiteFromCategories error: %v", err)
	}
	if len(list.Entry) != 2 {
		t.Fatalf("len(list.Entry) = %d, want 2", len(list.Entry))
	}

	tg := list.Entry[0]
	if tg.CountryCode != "TELEGRAM" {
		t.Fatalf("CountryCode = %s, want TELEGRAM", tg.CountryCode)
	}
	if len(tg.Domain) != 6 {
		t.Fatalf("len(tg.Domain) = %d, want 6", len(tg.Domain))
	}
}

func TestBuildGeoIPFromCategories(t *testing.T) {
	cats := []*dataset.Category{
		{
			Scope:   "shared",
			Name:    "telegram",
			TagName: "telegram",
			IPs: []string{
				"149.154.160.0/20",
				"2001:67c:4e8::/48",
				"8.8.8.8",
				"2001:db8::1",
				"invalid-ip",
			},
		},
	}

	list, err := BuildGeoIPFromCategories(cats)
	if err != nil {
		t.Fatalf("BuildGeoIPFromCategories error: %v", err)
	}
	if len(list.Entry) != 1 {
		t.Fatalf("len(list.Entry) = %d, want 1", len(list.Entry))
	}
	entry := list.Entry[0]
	if entry.CountryCode != "TELEGRAM" || len(entry.Cidr) != 4 {
		t.Fatalf("entry = %+v", entry)
	}
	if entry.Cidr[0].Prefix != 20 || net.IP(entry.Cidr[0].Ip).String() != "149.154.160.0" {
		t.Fatalf("cidr[0] = %+v", entry.Cidr[0])
	}
}

func TestWriteGeoSiteAndGeoIP(t *testing.T) {
	dir := t.TempDir()
	sitePath := filepath.Join(dir, "ru-geosite.dat")
	ipPath := filepath.Join(dir, "ru-geoip.dat")

	siteList := &router.GeoSiteList{
		Entry: []*router.GeoSite{
			{
				CountryCode: "TELEGRAM",
				Domain:      []*router.Domain{{Type: router.Domain_Domain, Value: "t.me"}},
			},
		},
	}
	if err := WriteGeoSite(sitePath, siteList); err != nil {
		t.Fatal(err)
	}
	rawSite, err := os.ReadFile(sitePath)
	if err != nil {
		t.Fatal(err)
	}
	var gotSite router.GeoSiteList
	if err := proto.Unmarshal(rawSite, &gotSite); err != nil {
		t.Fatal(err)
	}

	ipList := &router.GeoIPList{
		Entry: []*router.GeoIP{
			{
				CountryCode: "TELEGRAM",
				Cidr:        []*router.CIDR{{Ip: net.ParseIP("149.154.160.0").To4(), Prefix: 20}},
			},
		},
	}
	if err := WriteGeoIP(ipPath, ipList); err != nil {
		t.Fatal(err)
	}
}

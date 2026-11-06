package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/pntmsurf/rule-set/internal/dataset"
)

func main() {
	root := flag.String("root", ".", "repo root with data folder")
	flag.Parse()

	dataDir := filepath.Join(*root, "data")
	repo, err := dataset.LoadRepository(dataDir)
	if err != nil {
		log.Fatalf("load repository %s: %v", dataDir, err)
	}

	var errs []string
	cats, err := repo.ResolveAllCategories()
	if err != nil {
		log.Fatalf("resolve categories: %v", err)
	}

	for _, cat := range cats {
		if len(cat.Domains) == 0 && len(cat.IPs) == 0 {
			errs = append(errs, fmt.Sprintf("%s (%s): category has 0 domains and 0 CIDRs", cat.TagName, cat.Path))
		}
		for _, ipStr := range cat.IPs {
			if _, _, err := net.ParseCIDR(ipStr); err != nil {
				if net.ParseIP(ipStr) == nil {
					errs = append(errs, fmt.Sprintf("%s (%s): invalid IP/CIDR %q", cat.TagName, cat.Path, ipStr))
				}
			}
		}
	}

	for _, region := range repo.RegionNames() {
		regCats, err := repo.ResolveRegionCategories(region)
		if err != nil {
			errs = append(errs, fmt.Sprintf("region %s: %v", region, err))
			continue
		}
		domCount := 0
		ipCount := 0
		for _, c := range regCats {
			if len(c.Domains) > 0 {
				domCount++
			}
			if len(c.IPs) > 0 {
				ipCount++
			}
		}
		log.Printf("region %-4s: %3d total categories (%3d geosite tags, %2d geoip tags)", region, len(regCats), domCount, ipCount)
	}

	if len(errs) > 0 {
		fmt.Fprintln(os.Stderr, "\nValidation failed:")
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
		os.Exit(1)
	}

	log.Printf("validation passed (%d total unique categories across shared and %d regions)", len(cats), len(repo.RegionNames()))
}

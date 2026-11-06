package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pntmsurf/rule-set/internal/dataset"
	"github.com/pntmsurf/rule-set/internal/xraygeo"
)

func main() {
	root := flag.String("root", ".", "repo root with data folder")
	out := flag.String("out", "", "output folder for <region>-geosite.dat and <region>-geoip.dat, default root/release")
	onlyRegion := flag.String("region", "", "optional single region to build (e.g. ru, cn, ir, us, gb)")
	flag.Parse()

	outDir := *out
	if outDir == "" {
		outDir = filepath.Join(*root, "release")
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Fatalf("cannot create %s: %v", outDir, err)
	}

	dataDir := filepath.Join(*root, "data")
	repo, err := dataset.LoadRepository(dataDir)
	if err != nil {
		log.Fatalf("load repository %s: %v", dataDir, err)
	}

	regions := repo.RegionNames()
	if *onlyRegion != "" {
		regions = []string{strings.ToLower(strings.TrimSpace(*onlyRegion))}
	}

	for _, region := range regions {
		cats, err := repo.ResolveRegionCategories(region)
		if err != nil {
			log.Fatalf("resolve region %s: %v", region, err)
		}

		geosite, err := xraygeo.BuildGeoSiteFromCategories(cats)
		if err != nil {
			log.Fatalf("build %s-geosite: %v", region, err)
		}
		geositePath := filepath.Join(outDir, region+"-geosite.dat")
		if err := xraygeo.WriteGeoSite(geositePath, geosite); err != nil {
			log.Fatalf("write %s: %v", geositePath, err)
		}
		log.Printf("%s (%d categories)", geositePath, len(geosite.Entry))

		geoip, err := xraygeo.BuildGeoIPFromCategories(cats)
		if err != nil {
			log.Fatalf("build %s-geoip: %v", region, err)
		}
		geoipPath := filepath.Join(outDir, region+"-geoip.dat")
		if err := xraygeo.WriteGeoIP(geoipPath, geoip); err != nil {
			log.Fatalf("write %s: %v", geoipPath, err)
		}
		log.Printf("%s (%d categories)", geoipPath, len(geoip.Entry))
	}
}

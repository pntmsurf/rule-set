package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/pntmsurf/rule-set/internal/dataset"
	"github.com/pntmsurf/rule-set/internal/singboxrules"
)

func main() {
	root := flag.String("root", ".", "repo root with data folder")
	out := flag.String("out", "", "output folder for .srs files, default is root/release")
	singboxBin := flag.String("singbox-bin", "sing-box", "path to sing-box binary or name in PATH")
	version := flag.Int("version", singboxrules.DefaultSRSVersion, "sing-box rule-set version (1, 2, or 3)")
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

	categories, err := repo.ResolveAllCategories()
	if err != nil {
		log.Fatalf("resolve categories: %v", err)
	}

	built := 0
	for _, cat := range categories {
		if srs, ok := singboxrules.BuildDomainSRS(cat.Domains, *version); ok {
			jsonPath := filepath.Join(outDir, cat.TagName+".json")
			srsPath := filepath.Join(outDir, cat.TagName+".srs")
			if err := singboxrules.Compile(*singboxBin, srs, jsonPath, srsPath); err != nil {
				log.Printf("  fail domain %s: %v", cat.TagName, err)
			} else {
				built++
			}
		}

		if srs, ok := singboxrules.BuildIPSRS(cat.IPs, *version); ok {
			jsonPath := filepath.Join(outDir, cat.TagName+"-ip.json")
			ipPath := filepath.Join(outDir, cat.TagName+"-ip.srs")
			if err := singboxrules.Compile(*singboxBin, srs, jsonPath, ipPath); err != nil {
				log.Printf("  fail ip %s: %v", cat.TagName, err)
			} else {
				built++
			}
		}
	}

	log.Printf("built %d .srs rule-sets in %s", built, outDir)
}

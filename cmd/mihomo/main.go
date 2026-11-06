package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/pntmsurf/rule-set/internal/dataset"
	"github.com/pntmsurf/rule-set/internal/mihomobuild"
)

func main() {
	root := flag.String("root", ".", "repo root with data folder")
	out := flag.String("out", "", "output folder for .mrs files, default is root/release")
	mihomoBin := flag.String("mihomo-bin", "mihomo", "path to mihomo binary or name in PATH")
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
		domLines := mihomobuild.SanitizeDomainLines(cat.Domains)
		if len(domLines) > 0 {
			tmpPath := filepath.Join(outDir, cat.TagName+".domain.txt")
			outPath := filepath.Join(outDir, cat.TagName+".mrs")
			if err := mihomobuild.ConvertLines(*mihomoBin, mihomobuild.KindDomain, domLines, tmpPath, outPath); err != nil {
				log.Printf("  fail domain %s: %v", cat.TagName, err)
			} else {
				built++
			}
		}

		ipLines := mihomobuild.SanitizeIPLines(cat.IPs)
		if len(ipLines) > 0 {
			tmpPath := filepath.Join(outDir, cat.TagName+".ip.txt")
			ipPath := filepath.Join(outDir, cat.TagName+"-ip.mrs")
			if err := mihomobuild.ConvertLines(*mihomoBin, mihomobuild.KindIPCIDR, ipLines, tmpPath, ipPath); err != nil {
				log.Printf("  fail ip %s: %v", cat.TagName, err)
			} else {
				built++
			}
		}
	}

	log.Printf("built %d .mrs rule-sets in %s", built, outDir)
}

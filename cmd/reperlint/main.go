package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/rusl222/scada/analyzer"
	"github.com/rusl222/scada/reperdb"

	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("get working directory: %v", err)
	}

	root, err := analyzer.FindProjectRoot(wd)
	if err != nil {
		log.Fatal(err)
	}

	dbPath := filepath.Join(root, "sqlite.db")

	repo, err := reperdb.Open(dbPath)
	if err != nil {
		log.Fatalf("open reper database %q: %v", dbPath, err)
	}

	cfg := analyzer.Config{
		// TODO: заменить на реальный import path.
		VarPackagePath: "example.com/reperlint/scada",

		VarTypeName:     "Var",
		BindMethodName:  "Bind",
		ReperRepository: repo,
	}

	singlechecker.Main(
		analyzer.NewAnalyzer(cfg),
	)
}

package puritycheck

import (
	"fmt"
	"log"
	"os"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/archcheck"
)

type Finding = archcheck.Finding

func Check(root, goCommand string, platform archcheck.Platform, pure func(string) bool) ([]Finding, error) {
	p, err := load(root, goCommand, platform, pure)
	if err != nil {
		return []Finding{{Category: "package-error", Path: root, Detail: err.Error()}}, err
	}
	findings := analyze(p, time.Now(), log.New(os.Stderr, "", 0))
	for _, finding := range findings {
		if finding.Category == "package-error" {
			return findings, fmt.Errorf("%s", finding.Detail)
		}
	}
	return findings, nil
}

package archcheck

import (
	"fmt"
	"log"
	"os"
	"time"
)

func CheckPurity(root, goCommand string, platform Platform, pure func(string) bool) ([]Finding, error) {
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

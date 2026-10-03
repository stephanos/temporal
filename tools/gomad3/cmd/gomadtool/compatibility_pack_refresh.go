package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"

	"go.temporal.io/server/tools/gomad3/internal/compatibilitypack/authoring"
	capabilityanalysis "go.temporal.io/server/tools/gomad3/qualification/analysis"
	"go.temporal.io/server/tools/gomad3/upgrade"
)

type packTarget struct {
	platform string
	working  string
}

func runCompatibilityPackRefresh(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomadtool compatibility-pack refresh", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "", "Gomad v3 module root")
	impactPath := flags.String("impact-report", "", "pin-impact JSON report for the bumped checkout")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || *root == "" || *impactPath == "" {
		return 2
	}
	resolvedRoot, err := filepath.Abs(*root)
	if err != nil {
		return refreshInputError(stderr, err)
	}
	compatibilityRoot := filepath.Join(resolvedRoot, "internal", "compatibilitypack")
	targets, err := readPackTargets(filepath.Join(compatibilityRoot, "targets.tsv"), resolvedRoot)
	if err != nil {
		return refreshInputError(stderr, err)
	}
	contents, err := readCompatibilityPackFile(*impactPath, 16<<20)
	if err != nil {
		return refreshInputError(stderr, err)
	}
	var impact upgrade.PinImpact
	if err := json.Unmarshal(contents, &impact); err != nil || impact.Schema != "gomad3.pin-impact/v1" {
		return refreshInputError(stderr, errors.New("invalid pin-impact report"))
	}
	ids, err := invalidatedPackRequests(impact, targets)
	if err != nil {
		return refreshInputError(stderr, err)
	}
	return refreshCompatibilityPacks(resolvedRoot, compatibilityRoot, runtime.GOOS+"/"+runtime.GOARCH, ids, targets, discoverPackRequest, stdout, stderr)
}

func refreshInputError(stderr io.Writer, err error) int {
	if _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {
		return 3
	}
	return 2
}

type packDiscover func(authoring.Request, string, string) (authoring.Request, string, error)

func discoverPackRequest(request authoring.Request, working, toolchain string) (authoring.Request, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), compatibilityPackTimeout)
	defer cancel()
	prepared, err := capabilityanalysis.PrepareCapabilityReview(ctx, request.ReviewSpec(working, toolchain))
	if err != nil {
		return authoring.Request{}, "", err
	}
	fresh, digest, err := authoring.Discover(request, prepared.Review)
	return fresh, digest, errors.Join(err, prepared.Close())
}

func refreshCompatibilityPacks(root, compatibilityRoot, platform string, ids []string, targets map[string]packTarget, discover packDiscover, stdout, stderr io.Writer) int {
	failed := false
	for _, id := range ids {
		target := targets[id]
		requestPath := filepath.Join(compatibilityRoot, "requests", id+".json")
		request, status := readReviewedCompatibilityPackRequest(requestPath, stderr)
		if status != 0 || request.ID != id || len(request.Platforms) != 1 || request.Platforms[0] != target.platform {
			if _, err := fmt.Fprintf(stderr, "%s: invalid mapped request\n", id); err != nil {
				return 1
			}
			failed = true
			continue
		}
		if target.platform != platform {
			if _, err := fmt.Fprintf(stdout, "%s: not evaluable on %s; unchanged\n", id, platform); err != nil {
				return 1
			}
			continue
		}
		fresh, digest, err := discover(request, target.working, filepath.Join(root, ".toolchain"))
		if err != nil {
			if _, writeErr := fmt.Fprintf(stderr, "%s: %v\n", id, err); writeErr != nil {
				return 1
			}
			failed = true
			continue
		}
		if request.ApprovalSHA256 == digest {
			continue
		}
		_, reviewedDigest, err := authoring.RenderReview(fresh)
		if err != nil || reviewedDigest != digest {
			if err == nil {
				err = errors.New("discovery and review digests differ")
			}
			if _, writeErr := fmt.Fprintf(stderr, "%s: fresh review failed: %v\n", id, err); writeErr != nil {
				return 1
			}
			failed = true
			continue
		}
		if !reflect.DeepEqual(request, fresh) {
			if err := authoring.PublishRequest(requestPath, fresh); err != nil {
				if _, writeErr := fmt.Fprintf(stderr, "%s: %v\n", id, err); writeErr != nil {
					return 1
				}
				failed = true
				continue
			}
		}
		if _, err := authoring.PublishReview(filepath.Join(compatibilityRoot, "reports", id+".md"), fresh); err != nil {
			if _, writeErr := fmt.Fprintf(stderr, "%s: %v\n", id, err); writeErr != nil {
				return 1
			}
			failed = true
			continue
		}
		if _, err := fmt.Fprintf(stdout, "%s: review %s\n", id, digest); err != nil {
			return 1
		}
	}
	if failed {
		return 1
	}
	return 0
}

func readPackTargets(path, root string) (map[string]packTarget, error) {
	contents, err := readCompatibilityPackFile(path, 64<<10)
	if err != nil {
		return nil, err
	}
	targets := make(map[string]packTarget)
	scanner := bufio.NewScanner(bytes.NewReader(contents))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 3 || fields[1] == "" || strings.ContainsAny(fields[1], "/\\.") ||
			fields[0] != "darwin/arm64" && fields[0] != "linux/amd64" ||
			filepath.IsAbs(fields[2]) {
			return nil, fmt.Errorf("invalid compatibility-pack target mapping: %q", scanner.Text())
		}
		if _, exists := targets[fields[1]]; exists {
			return nil, fmt.Errorf("duplicate compatibility-pack target mapping: %s", fields[1])
		}
		working := filepath.Clean(filepath.Join(root, fields[2]))
		if _, err := os.Stat(filepath.Join(working, "go.mod")); err != nil {
			return nil, fmt.Errorf("compatibility-pack target %s has no module: %w", fields[1], err)
		}
		targets[fields[1]] = packTarget{platform: fields[0], working: working}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return targets, nil
}

func invalidatedPackRequests(impact upgrade.PinImpact, targets map[string]packTarget) ([]string, error) {
	set := make(map[string]bool)
	for _, pin := range impact.Pins {
		if pin.Class != "pack_rule" || pin.Status != "invalidated" && pin.Status != "unknown" {
			continue
		}
		id, _, found := strings.Cut(pin.ID, ":")
		if !found || targets[id].working == "" {
			return nil, fmt.Errorf("invalidated compatibility-pack request %q has no target mapping", pin.ID)
		}
		set[id] = true
	}
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}

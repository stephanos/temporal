package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/format"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"go.temporal.io/server/common/dynamicconfig"
)

const registryHelperArgument = "--internal-registry-helper"

type helperCommand func(context.Context, string, ...string) ([]byte, []byte, error)

var productionFixtureGenerator func() ([]ResolutionFixture, error)

func run(ctx context.Context, moduleRoot string) (Catalog, error) {
	sites, err := discoverRegistrationSites(ctx, moduleRoot)
	if err != nil {
		return Catalog{}, err
	}
	packages := registrationPackages(sites)
	helperDirectory := filepath.Join(moduleRoot, "cmd", "tools", "genleandynamicconfig")
	catalog, err := runRegistryHelper(ctx, moduleRoot, helperDirectory, packages, executeHelper)
	if err != nil {
		return Catalog{}, err
	}
	catalog, err = reconcileDiscovery(catalog, sites)
	if err != nil {
		return Catalog{}, err
	}
	if err := validateFixtures(catalog.Fixtures); err != nil {
		return Catalog{}, fmt.Errorf("projection fixtures: %w", err)
	}
	identity, err := catalogIdentity(catalog)
	if err != nil {
		return Catalog{}, err
	}
	catalog.Identity = identity
	return catalog, nil
}

func runRegistryHelper(
	ctx context.Context,
	moduleRoot string,
	helperDirectory string,
	packages []string,
	runner helperCommand,
) (catalog Catalog, resultErr error) {
	source, err := registryHelperSource(packages)
	if err != nil {
		return Catalog{}, err
	}
	helper, err := os.CreateTemp(helperDirectory, "zz_genleandynamicconfig_helper_*.go")
	if err != nil {
		return Catalog{}, fmt.Errorf("helper create in %q: %w", helperDirectory, err)
	}
	helperPath := helper.Name()
	defer func() {
		if helper != nil {
			if closeErr := helper.Close(); closeErr != nil {
				resultErr = errors.Join(resultErr, fmt.Errorf("helper close %q: %w", helperPath, closeErr))
				catalog = Catalog{}
			}
		}
		if removeErr := os.Remove(helperPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			resultErr = errors.Join(resultErr, fmt.Errorf("helper remove %q: %w", helperPath, removeErr))
			catalog = Catalog{}
		}
	}()
	if err := helper.Chmod(0o600); err != nil {
		return Catalog{}, fmt.Errorf("helper chmod %q: %w", helperPath, err)
	}
	if _, err := helper.Write(source); err != nil {
		return Catalog{}, fmt.Errorf("helper write %q: %w", helperPath, err)
	}
	if err := helper.Close(); err != nil {
		return Catalog{}, fmt.Errorf("helper close %q: %w", helperPath, err)
	}
	helper = nil

	stdout, stderr, err := runner(
		ctx,
		moduleRoot,
		"run",
		"-tags=test_dep",
		"./cmd/tools/genleandynamicconfig",
		registryHelperArgument,
	)
	if err != nil {
		diagnostic := strings.TrimSpace(string(stderr))
		if panicIndex := strings.Index(diagnostic, "panic:"); panicIndex >= 0 {
			panicLine := strings.SplitN(diagnostic[panicIndex:], "\n", 2)[0]
			return Catalog{}, fmt.Errorf("helper initialization %s", panicLine)
		}
		if diagnostic == "" {
			return Catalog{}, fmt.Errorf("helper run: %w", err)
		}
		return Catalog{}, fmt.Errorf("helper run: %w: %s", err, diagnostic)
	}
	decoder := json.NewDecoder(bytes.NewReader(stdout))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&catalog); err != nil {
		return Catalog{}, fmt.Errorf("helper decode catalog: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return Catalog{}, errors.New("helper decode catalog: unexpected trailing JSON")
		}
		return Catalog{}, fmt.Errorf("helper decode catalog: %w", err)
	}
	return catalog, nil
}

func registryHelperSource(packages []string) ([]byte, error) {
	packages = slices.Clone(packages)
	slices.Sort(packages)
	packages = slices.Compact(packages)
	var source strings.Builder
	source.WriteString("package main\n\nimport (\n")
	for _, packagePath := range packages {
		if packagePath == dynamicConfigPackagePath || packagePath == "go.temporal.io/server/chasm/lib/callback" {
			continue
		}
		fmt.Fprintf(&source, "\t_ %s\n", strconv.Quote(packagePath))
	}
	source.WriteString(`
	callbackconfig "go.temporal.io/server/chasm/lib/callback"
	"go.temporal.io/server/common/dynamicconfig"
)

func init() {
	productionFixtureGenerator = func() ([]ResolutionFixture, error) {
		return computeProductionFixtures(productionSettings{
			Global:        dynamicconfig.AdminEnableListHistoryTasks,
			Namespace:     callbackconfig.MaxPerExecution,
			NamespaceID:   dynamicconfig.SkipReapplicationByNamespaceID,
			TaskQueue:     dynamicconfig.MatchingUpdateAckInterval,
			ShardID:       dynamicconfig.ReplicationTaskProcessorErrorRetryMaxAttempts,
			TaskType:      dynamicconfig.StandbyTaskMissingEventsResendDelay,
			Destination:   callbackconfig.RequestTimeout,
			ChasmTaskType: dynamicconfig.ChasmStandbyTaskDiscardDelay,
		})
	}
}
`)
	formatted, err := format.Source([]byte(source.String()))
	if err != nil {
		return nil, fmt.Errorf("helper format: %w", err)
	}
	return formatted, nil
}

func executeHelper(
	ctx context.Context,
	moduleRoot string,
	arguments ...string,
) (stdoutBytes []byte, stderrBytes []byte, resultErr error) {
	command := exec.CommandContext(ctx, "go", arguments...)
	command.Dir = moduleRoot
	command.Env = append(os.Environ(), "GOWORK=off")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	resultErr = command.Run()
	return stdout.Bytes(), stderr.Bytes(), resultErr
}

func writeRegistryCatalog(writer io.Writer) error {
	metadata, err := dynamicconfig.RegisteredSettingMetadata()
	if err != nil {
		return fmt.Errorf("helper snapshot: %w", err)
	}
	catalog, err := projectMetadata(metadata)
	if err != nil {
		return err
	}
	if productionFixtureGenerator == nil {
		return errors.New("helper fixtures: production fixture generator is not initialized")
	}
	fixtures, err := productionFixtureGenerator()
	if err != nil {
		return fmt.Errorf("helper fixtures: %w", err)
	}
	if err := validateFixtures(fixtures); err != nil {
		return fmt.Errorf("helper fixtures: %w", err)
	}
	catalog.Fixtures = fixtures
	identity, err := catalogIdentity(catalog)
	if err != nil {
		return fmt.Errorf("helper catalog identity: %w", err)
	}
	catalog.Identity = identity
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(catalog); err != nil {
		return fmt.Errorf("helper encode catalog: %w", err)
	}
	return nil
}

func registrationPackages(sites []RegistrationSite) []string {
	seen := make(map[string]struct{}, len(sites))
	result := make([]string, 0, len(sites))
	for _, site := range sites {
		if _, exists := seen[site.Package]; exists {
			continue
		}
		seen[site.Package] = struct{}{}
		result = append(result, site.Package)
	}
	slices.Sort(result)
	return result
}

func reconcileDiscovery(catalog Catalog, sites []RegistrationSite) (Catalog, error) {
	if len(catalog.Settings) == 0 {
		return Catalog{}, errors.New("reconcile: initialized registry catalog is empty")
	}
	sites = slices.Clone(sites)
	slices.SortFunc(sites, compareRegistrationSites)
	settingByKey := make(map[string]int, len(catalog.Settings))
	for index, setting := range catalog.Settings {
		settingByKey[setting.Key] = index
	}
	for _, site := range sites {
		index, exists := settingByKey[site.Key]
		if !exists {
			return Catalog{}, fmt.Errorf(
				"reconcile package %q key %q: discovered initializer was not registered",
				site.Package,
				site.Key,
			)
		}
		catalog.Settings[index].Provenance = append(catalog.Settings[index].Provenance, site)
	}
	for _, setting := range catalog.Settings {
		if len(setting.Provenance) == 0 {
			return Catalog{}, fmt.Errorf(
				"reconcile setting %q: initialized registry entry has no production initializer discovery",
				setting.Key,
			)
		}
	}
	for index := range catalog.Settings {
		slices.SortFunc(catalog.Settings[index].Provenance, compareRegistrationSites)
	}
	identity, err := catalogIdentity(catalog)
	if err != nil {
		return Catalog{}, err
	}
	catalog.Identity = identity
	return catalog, nil
}

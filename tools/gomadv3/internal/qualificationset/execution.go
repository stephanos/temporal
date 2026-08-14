package qualificationset

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/artifact"
	"go.temporal.io/server/tools/gomadv3/internal/capabilityanalysis"
	"go.temporal.io/server/tools/gomadv3/internal/choicewire"
	"go.temporal.io/server/tools/gomadv3/internal/qualify"
	"go.temporal.io/server/tools/gomadv3/internal/record"
)

func analysisCommand(config Config, manifest Manifest, suite Suite) Command {
	runTimeout, overallTimeout := suiteTimeouts(manifest, suite)
	args := []string{"analyze", "--format=json"}
	for _, value := range suite.BuildTags {
		args = append(args, "--build-tag="+value)
	}
	args = append(args, "go-test", suite.Package, "--", "-test.run=^"+regexp.QuoteMeta(suite.Test)+"$")
	return Command{
		Executable: config.GomadPath, Args: args, Dir: config.WorkingDir,
		Timeout: overallTimeout + manifestGrace(manifest) + 10*time.Second, Grace: min(runTimeout, manifestGrace(manifest)),
	}
}

func retainedAnalysis(result CommandResult) (capabilityanalysis.Report, string, error) {
	if result.Err != nil {
		classification := retainedErrorClassification(result, result.Err)
		return capabilityanalysis.Report{}, classification, result.Err
	}
	if len(result.Stderr) != 0 {
		return capabilityanalysis.Report{}, "runner_failure", errors.New("JSON capability analysis wrote to stderr")
	}
	if len(result.Stdout) == 0 || result.Stdout[len(result.Stdout)-1] != '\n' || bytes.Count(result.Stdout, []byte{'\n'}) != 1 {
		return capabilityanalysis.Report{}, "runner_failure", errors.New("capability analysis output framing is invalid")
	}
	report, err := capabilityanalysis.Decode(result.Stdout[:len(result.Stdout)-1])
	if err != nil {
		return capabilityanalysis.Report{}, "runner_failure", err
	}
	expectedStatus := 0
	if report.Classification == capabilityanalysis.ClassificationUnsupported {
		expectedStatus = 1
	}
	if result.ExitCode != expectedStatus {
		return capabilityanalysis.Report{}, "runner_failure", fmt.Errorf("capability analysis status is %d, want %d", result.ExitCode, expectedStatus)
	}
	return report, string(report.Classification), nil
}

func mergeAnalysisIdentity(report *SetReport, analysis capabilityanalysis.Report) error {
	if report.AnalysisCompleted == 1 {
		report.Platform = PlatformIdentity{GOOS: analysis.Toolchain.TargetGOOS, GOARCH: analysis.Toolchain.TargetGOARCH}
		report.Toolchain = analysis.Toolchain
		report.IOProfile = analysis.IOProfile
		return nil
	}
	if report.Platform != (PlatformIdentity{GOOS: analysis.Toolchain.TargetGOOS, GOARCH: analysis.Toolchain.TargetGOARCH}) || !reflect.DeepEqual(report.Toolchain, analysis.Toolchain) || !reflect.DeepEqual(report.IOProfile, analysis.IOProfile) {
		return errors.New("capability analyses disagree on implementation identity")
	}
	return nil
}

func matchesUnsupportedAnalysis(expected Expectation, analysis capabilityanalysis.Report) bool {
	if expected.Classification != "unsupported_target" || len(analysis.Blockers) == 0 {
		return false
	}
	first := analysis.Blockers[0]
	return first.Package.ImportPath == expected.ImportPath && first.Capability == expected.Capability
}

func matchesSupportedExpectation(expected Expectation, suite SuiteReport) bool {
	if expected.Classification != suite.Classification || len(suite.Seeds) == 0 {
		return false
	}
	for _, seed := range suite.Seeds {
		if seed.Classification != expected.Classification {
			return false
		}
	}
	return true
}

func contextClassification(err error) string {
	if errors.Is(err, context.Canceled) {
		return "cancelled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "overall_timeout"
	}
	return "runner_failure"
}

func retainedErrorClassification(result CommandResult, err error) string {
	if result.Err != nil {
		if classification := contextClassification(result.Err); classification != "runner_failure" {
			return classification
		}
	}
	if classification := contextClassification(err); classification != "runner_failure" {
		return classification
	}
	if result.ExitCode == 2 {
		return "invalid_input"
	}
	return "runner_failure"
}

func projectSeedReport(report qualify.Report, classification string, seed uint64, suite Suite, analysis capabilityanalysis.Report) (SeedReport, error) {
	if uint64(report.Seed) != seed {
		return SeedReport{}, fmt.Errorf("qualification seed is %d, want %d", report.Seed, seed)
	}
	result := SeedReport{
		Seed: record.Uint64String(seed), Classification: classification, EvidenceSHA256: report.EvidenceDigest,
		ReplayMatch: true, Choice: emptyChoiceCoverage(),
	}
	for index, run := range report.Runs {
		if err := mergeQualificationRun(&result, run, index, seed, suite, analysis); err != nil {
			return SeedReport{}, err
		}
	}
	if !result.Replayed {
		result.ReplayMatch = false
	}
	if suite.ReplaySuccesses && (!result.Replayed || !result.ReplayMatch) {
		return SeedReport{}, errors.New("qualification did not complete exact successful replay")
	}
	if suite.ChoiceBytes != 0 && !result.Choice.Available {
		return SeedReport{}, errors.New("qualification did not retain required choice coverage")
	}
	return result, nil
}

func mergeQualificationRun(result *SeedReport, run qualify.RunReport, index int, seed uint64, suite Suite, analysis capabilityanalysis.Report) error {
	if suite.ReplaySuccesses && (run.ArtifactPath == "" || run.Replay == nil || !run.Replay.Attempted) {
		return fmt.Errorf("qualification run %d is missing required successful replay evidence", index)
	}
	if run.Replay != nil {
		result.Replayed = true
		if !run.Replay.Match {
			result.ReplayMatch = false
			if result.ReplayDivergence == "" {
				result.ReplayDivergence = run.Replay.Divergence
			}
		}
	}
	if run.ArtifactPath == "" {
		return nil
	}
	opened, err := artifact.Open(run.ArtifactPath)
	if err != nil {
		return fmt.Errorf("open retained qualification artifact: %w", err)
	}
	artifactErr := validateArtifactIdentity(opened, seed, analysis)
	var coverage ChoiceCoverage
	if artifactErr == nil {
		coverage, artifactErr = projectArtifactChoice(opened)
	}
	if artifactErr == nil {
		result.ElapsedNanos += record.Uint64String(opened.Manifest.Host.ElapsedNanos)
		result.ArtifactBytes += record.Uint64String(opened.StoredBytes)
		if opened.Manifest.ChoiceProfile != nil {
			result.TraceBytes += opened.Manifest.ChoiceProfile.Trace.Bytes
			result.Choice, artifactErr = mergeChoiceCoverage(result.Choice, coverage)
		}
	}
	return errors.Join(artifactErr, opened.Close())
}

func validateArtifactIdentity(opened artifact.Artifact, seed uint64, analysis capabilityanalysis.Report) error {
	manifest := opened.Manifest
	if uint64(manifest.Seed) != seed || manifest.Toolchain.BuildKey != analysis.Toolchain.BuildKey || manifest.Toolchain.GoVersion != analysis.Toolchain.GoVersion || manifest.Toolchain.TargetGOOS != analysis.Toolchain.TargetGOOS || manifest.Toolchain.TargetGOARCH != analysis.Toolchain.TargetGOARCH {
		return errors.New("retained artifact identity does not match capability analysis")
	}
	if manifest.IOProfile.Name != analysis.IOProfile.Name || manifest.IOProfile.ImplementationSHA256 != analysis.IOProfile.ImplementationSHA256 || manifest.IOProfile.InventorySHA256 != analysis.IOProfile.InventorySHA256 {
		return errors.New("retained artifact I/O profile does not match capability analysis")
	}
	return nil
}

func projectArtifactChoice(opened artifact.Artifact) (ChoiceCoverage, error) {
	profile := opened.Manifest.ChoiceProfile
	if profile == nil {
		return emptyChoiceCoverage(), nil
	}
	payload, err := artifact.ReadPayload(opened, profile.Trace.File, uint64(profile.Trace.Limit))
	if err != nil {
		return ChoiceCoverage{}, err
	}
	targetIdentity, err := opened.Manifest.Target.SHA256.Bytes()
	if err != nil {
		return ChoiceCoverage{}, err
	}
	traceIdentity, err := profile.Trace.SHA256.Bytes()
	if err != nil {
		return ChoiceCoverage{}, err
	}
	projection, err := choicewire.ProjectComplete(payload, choicewire.CompleteMetadata{
		Limit: uint64(profile.Trace.Limit), Records: uint64(profile.Trace.Records), SHA256: traceIdentity,
	}, targetIdentity)
	if err != nil {
		return ChoiceCoverage{}, fmt.Errorf("project retained choice trace: %w", err)
	}
	return ChoiceCoverage{
		Available: true, Profile: profile.Name, ImplementationSHA256: profile.ImplementationSHA256,
		Limit: profile.Trace.Limit, Features: append([]choicewire.Feature(nil), projection.Features.Values...),
		AdjacentPairsObserved:  record.Uint64String(projection.Features.AdjacentPairsObserved),
		AdjacentPairsTruncated: projection.Features.AdjacentPairsTruncated,
	}, nil
}

func emptyChoiceCoverage() ChoiceCoverage {
	return ChoiceCoverage{Features: []choicewire.Feature{}}
}

func aggregateChoiceCoverage(seeds []SeedReport) (ChoiceCoverage, error) {
	result := emptyChoiceCoverage()
	for _, seed := range seeds {
		var err error
		result, err = mergeChoiceCoverage(result, seed.Choice)
		if err != nil {
			return ChoiceCoverage{}, err
		}
	}
	return result, nil
}

func mergeChoiceCoverage(left, right ChoiceCoverage) (ChoiceCoverage, error) {
	if !right.Available {
		return left, nil
	}
	if !left.Available {
		right.Features = append([]choicewire.Feature(nil), right.Features...)
		return right, nil
	}
	if left.Profile != right.Profile || left.ImplementationSHA256 != right.ImplementationSHA256 || left.Limit != right.Limit {
		return ChoiceCoverage{}, errors.New("retained choice coverage identities disagree")
	}
	features := append(append([]choicewire.Feature(nil), left.Features...), right.Features...)
	sort.Slice(features, func(i, j int) bool {
		if features[i].Kind != features[j].Kind {
			return features[i].Kind < features[j].Kind
		}
		return features[i].Value < features[j].Value
	})
	deduplicated := features[:0]
	for _, feature := range features {
		if len(deduplicated) == 0 || deduplicated[len(deduplicated)-1] != feature {
			deduplicated = append(deduplicated, feature)
		}
	}
	left.Features = deduplicated
	left.AdjacentPairsObserved += right.AdjacentPairsObserved
	left.AdjacentPairsTruncated = left.AdjacentPairsTruncated || right.AdjacentPairsTruncated
	return left, nil
}

func addSeedTotals(report *SetReport, seed SeedReport) {
	report.ElapsedNanos += seed.ElapsedNanos
	report.ArtifactBytes += seed.ArtifactBytes
	report.TraceBytes += seed.TraceBytes
	if seed.Replayed {
		report.Replayed++
	}
	if seed.Replayed && !seed.ReplayMatch {
		report.ReplayDiverged++
	}
}

func suiteTimeouts(manifest Manifest, suite Suite) (runTimeout time.Duration, overallTimeout time.Duration) {
	runTimeout, _ = time.ParseDuration(manifest.RunTimeout)
	overallTimeout, _ = time.ParseDuration(manifest.OverallTimeout)
	if suite.RunTimeout != "" {
		runTimeout, _ = time.ParseDuration(suite.RunTimeout)
	}
	if suite.OverallTimeout != "" {
		overallTimeout, _ = time.ParseDuration(suite.OverallTimeout)
	}
	return runTimeout, overallTimeout
}

func manifestGrace(manifest Manifest) time.Duration {
	grace, _ := time.ParseDuration(manifest.TerminateGrace)
	return grace
}

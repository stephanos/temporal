package generate

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"

	"go.temporal.io/server/tools/umpire3/checker/finite"
	"go.temporal.io/server/tools/umpire3/checker/veil"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tools/umpire3/protocol/checker"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

type checkerCoverageInputs struct {
	nativeCertificate string
	nativeReceipt     string
	nativeBenchmark   string
	veilBinding       string
	veilResults       []string
}

func exportCheckerCoverage(inputs checkerCoverageInputs, writer io.Writer) error {
	manifest, err := buildCheckerCoverage(inputs)
	if err != nil {
		return err
	}
	encoded, err := manifest.CanonicalJSON()
	if err != nil {
		return err
	}
	if _, err := writer.Write(append(encoded, '\n')); err != nil {
		return fmt.Errorf("write checker coverage: %w", err)
	}
	return nil
}

func buildCheckerCoverage(inputs checkerCoverageInputs) (protocolchecker.CheckerCoverageManifest, error) {
	composition, err := protocolcatalog.DefaultComposition()
	if err != nil {
		return protocolchecker.CheckerCoverageManifest{}, err
	}
	catalog, err := protocolcatalog.DefaultCatalog()
	if err != nil {
		return protocolchecker.CheckerCoverageManifest{}, err
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return protocolchecker.CheckerCoverageManifest{}, err
	}
	finiteCatalog, err := finite.DefaultFiniteReplayCatalog()
	if err != nil {
		return protocolchecker.CheckerCoverageManifest{}, err
	}
	nexusView, found, err := finite.DefaultFirstOrderView(
		protocolcatalog.TargetIDNexusCancellation, "sound")
	if err != nil {
		return protocolchecker.CheckerCoverageManifest{}, err
	}
	if !found {
		return protocolchecker.CheckerCoverageManifest{}, errors.New("sound Nexus first-order view is unavailable")
	}
	nativeEntry, err := checkedNativeCoverage(inputs, nexusView)
	if err != nil {
		return protocolchecker.CheckerCoverageManifest{}, err
	}
	veilEntry, err := checkedVeilCoverage(inputs, nexusView)
	if err != nil {
		return protocolchecker.CheckerCoverageManifest{}, err
	}

	manifest := protocolchecker.CheckerCoverageManifest{
		FormatVersion: protocolchecker.CheckerCoverageFormatVersion,
		CatalogHash:   catalogHash, CompositionSemanticHash: composition.SemanticHash,
		Entries: []protocolchecker.CheckerCoverageEntry{},
	}
	for _, target := range composition.Targets {
		for _, property := range target.Properties {
			exact, err := exactCoverage(finiteCatalog, nexusView, target.Identifier, property)
			if err != nil {
				return protocolchecker.CheckerCoverageManifest{}, err
			}
			manifest.Entries = append(manifest.Entries, exact)
			if target.Identifier == protocolcatalog.TargetIDNexusCancellation &&
				property == protocolcatalog.PropertyIDNexusCancellationWonExcludesSuccess {
				manifest.Entries = append(manifest.Entries, nativeEntry, veilEntry)
				continue
			}
			manifest.Entries = append(manifest.Entries,
				unsupportedCoverage(target.Identifier, property, protocolchecker.CheckerNative,
					"no native certificate view is declared for this target/property"),
				unsupportedCoverage(target.Identifier, property, protocolchecker.CheckerVeil,
					"the owning Lean family does not import a Veil declaration"),
			)
		}
	}
	slices.SortFunc(manifest.Entries, func(left, right protocolchecker.CheckerCoverageEntry) int {
		if comparison := compareText(string(left.Target), string(right.Target)); comparison != 0 {
			return comparison
		}
		if comparison := compareText(string(left.Property), string(right.Property)); comparison != 0 {
			return comparison
		}
		return compareText(string(left.Checker), string(right.Checker))
	})
	if err := manifest.Validate(); err != nil {
		return protocolchecker.CheckerCoverageManifest{}, err
	}
	return manifest, nil
}

func exactCoverage(
	finiteCatalog protocolchecker.FiniteReplayCatalog,
	nexus protocolchecker.FirstOrderView,
	target protocolcatalog.TargetID,
	property protocolcatalog.PropertyID,
) (protocolchecker.CheckerCoverageEntry, error) {
	entry := protocolchecker.CheckerCoverageEntry{
		Target: target, Property: property, Checker: protocolchecker.CheckerExact,
		Status: protocolchecker.CheckerCoverageChecked,
		Claims: []protocolchecker.CheckerClaim{{
			Job: "exhaustive", ResultClass: protocolcatalog.ResultClassFiniteExhaustive,
			TrustBadge: protocolcatalog.TrustBadgeCheckedCertificate, Exact: true,
			Bounds: protocolchecker.BackendBounds{}, Omissions: []string{},
		}},
	}
	if target == nexus.Target && property == nexus.Property {
		encoded, err := nexus.CanonicalJSON()
		if err != nil {
			return protocolchecker.CheckerCoverageEntry{}, err
		}
		entry.World = nexus.World
		entry.Variant = nexus.Variant
		entry.SemanticHash = nexus.SemanticHash
		entry.Evidence = []protocolchecker.CheckerEvidence{{
			Kind: "first-order-exact-view", Digest: checkerDigest(encoded),
			Declaration: nexus.Relation.Declaration,
		}}
		return entry, nil
	}
	view, found := finiteCatalog.Target(target, property)
	if !found {
		return protocolchecker.CheckerCoverageEntry{},
			fmt.Errorf("no exact finite replay view for %q/%q", target, property)
	}
	encoded, err := json.Marshal(view)
	if err != nil {
		return protocolchecker.CheckerCoverageEntry{}, err
	}
	entry.World = view.World
	entry.Variant = view.Variant
	entry.SemanticHash = view.SemanticHash
	entry.Evidence = []protocolchecker.CheckerEvidence{{
		Kind: "finite-replay-exact-view", Digest: checkerDigest(encoded),
		Declaration: view.Relation.Declaration,
	}}
	return entry, nil
}

func checkedNativeCoverage(
	inputs checkerCoverageInputs,
	view protocolchecker.FirstOrderView,
) (protocolchecker.CheckerCoverageEntry, error) {
	certificateBytes, err := os.ReadFile(inputs.nativeCertificate)
	if err != nil {
		return protocolchecker.CheckerCoverageEntry{}, fmt.Errorf("read native certificate: %w", err)
	}
	certificate, err := finite.DecodeCertificate(bytes.NewReader(certificateBytes),
		protocolexperiment.DefaultDecodeLimit, view)
	if err != nil {
		return protocolchecker.CheckerCoverageEntry{}, fmt.Errorf("validate native certificate: %w", err)
	}
	receiptBytes, err := os.ReadFile(inputs.nativeReceipt)
	if err != nil {
		return protocolchecker.CheckerCoverageEntry{}, fmt.Errorf("read native receipt: %w", err)
	}
	receipt, err := finite.DecodeReceipt(bytes.NewReader(receiptBytes),
		protocolexperiment.DefaultDecodeLimit, certificate)
	if err != nil {
		return protocolchecker.CheckerCoverageEntry{}, fmt.Errorf("validate native receipt: %w", err)
	}
	canonicalCertificate, err := certificate.CanonicalJSON(view)
	if err != nil {
		return protocolchecker.CheckerCoverageEntry{}, err
	}
	canonicalReceipt, err := receipt.CanonicalJSON(certificate)
	if err != nil {
		return protocolchecker.CheckerCoverageEntry{}, err
	}
	benchmarkBytes, err := os.ReadFile(inputs.nativeBenchmark)
	if err != nil {
		return protocolchecker.CheckerCoverageEntry{}, fmt.Errorf("read native benchmark: %w", err)
	}
	benchmark, err := finite.DecodeBenchmarkReport(bytes.NewReader(benchmarkBytes),
		protocolexperiment.DefaultDecodeLimit, view, certificate, receipt)
	if err != nil {
		return protocolchecker.CheckerCoverageEntry{}, fmt.Errorf("validate native benchmark: %w", err)
	}
	canonicalBenchmark, err := benchmark.CanonicalJSON(view, certificate, receipt)
	if err != nil {
		return protocolchecker.CheckerCoverageEntry{}, err
	}
	evidence := []protocolchecker.CheckerEvidence{
		{Kind: "native-certificate", Digest: checkerDigest(canonicalCertificate)},
		{Kind: "native-lean-receipt", Digest: checkerDigest(canonicalReceipt)},
		{Kind: "native-scale-benchmark", Digest: checkerDigest(canonicalBenchmark)},
	}
	slices.SortFunc(evidence, compareCoverageEvidence)
	return protocolchecker.CheckerCoverageEntry{
		Target: view.Target, Property: view.Property, Checker: protocolchecker.CheckerNative,
		Status: protocolchecker.CheckerCoverageChecked, World: view.World, Variant: view.Variant,
		SemanticHash: view.SemanticHash,
		Claims: []protocolchecker.CheckerClaim{{
			Job: "exhaustive", ResultClass: receipt.ResultClass, TrustBadge: receipt.TrustBadge,
			Exact: true, Bounds: protocolchecker.BackendBounds{}, Omissions: []string{},
		}},
		Evidence: evidence,
	}, nil
}

func checkedVeilCoverage(
	inputs checkerCoverageInputs,
	view protocolchecker.FirstOrderView,
) (protocolchecker.CheckerCoverageEntry, error) {
	bindingBytes, err := os.ReadFile(inputs.veilBinding)
	if err != nil {
		return protocolchecker.CheckerCoverageEntry{}, fmt.Errorf("read Veil binding: %w", err)
	}
	binding, err := veil.DecodeBindingArtifact(bytes.NewReader(bindingBytes), protocolexperiment.DefaultDecodeLimit)
	if err != nil {
		return protocolchecker.CheckerCoverageEntry{}, fmt.Errorf("validate Veil binding: %w", err)
	}
	if err := binding.ValidateAgainst(view); err != nil {
		return protocolchecker.CheckerCoverageEntry{}, err
	}
	canonicalBinding, err := binding.CanonicalJSON()
	if err != nil {
		return protocolchecker.CheckerCoverageEntry{}, err
	}
	evidence := []protocolchecker.CheckerEvidence{{
		Kind: "veil-semantic-binding", Digest: checkerDigest(canonicalBinding),
		Declaration: binding.Binding.SemanticBinding.Declaration,
	}}
	claims := make([]protocolchecker.CheckerClaim, 0, len(inputs.veilResults))
	for _, path := range inputs.veilResults {
		if path == "" {
			return protocolchecker.CheckerCoverageEntry{}, errors.New("veil result path cannot be empty")
		}
		resultBytes, err := os.ReadFile(path)
		if err != nil {
			return protocolchecker.CheckerCoverageEntry{}, fmt.Errorf("read Veil result %q: %w", path, err)
		}
		result, err := protocolchecker.DecodeBackendResult(
			bytes.NewReader(resultBytes), protocolexperiment.DefaultDecodeLimit)
		if err != nil {
			return protocolchecker.CheckerCoverageEntry{}, fmt.Errorf("validate Veil result %q: %w", path, err)
		}
		if result.Target != view.Target || result.Property != view.Property ||
			result.World != view.World || result.Variant != view.Variant ||
			result.SemanticHash != view.SemanticHash ||
			result.BindingArtifactDigest != binding.ArtifactDigest {
			return protocolchecker.CheckerCoverageEntry{},
				fmt.Errorf("veil result %q does not match its checked binding and view", path)
		}
		canonical, err := result.CanonicalJSON()
		if err != nil {
			return protocolchecker.CheckerCoverageEntry{}, err
		}
		claims = append(claims, protocolchecker.CheckerClaim{
			Job: string(result.Job), ResultClass: result.ResultClass, TrustBadge: result.TrustBadge,
			Exact: result.Exact, Bounds: result.Bounds,
			Omissions: append([]string{}, result.Omissions...),
		})
		evidence = append(evidence, protocolchecker.CheckerEvidence{
			Kind: "veil-" + string(result.Job) + "-result", Digest: checkerDigest(canonical),
		})
	}
	slices.SortFunc(claims, func(left, right protocolchecker.CheckerClaim) int {
		return compareText(left.Job, right.Job)
	})
	slices.SortFunc(evidence, compareCoverageEvidence)
	return protocolchecker.CheckerCoverageEntry{
		Target: view.Target, Property: view.Property, Checker: protocolchecker.CheckerVeil,
		Status: protocolchecker.CheckerCoverageChecked, World: view.World, Variant: view.Variant,
		SemanticHash: view.SemanticHash, Claims: claims, Evidence: evidence,
	}, nil
}

func unsupportedCoverage(
	target protocolcatalog.TargetID,
	property protocolcatalog.PropertyID,
	checker protocolchecker.CheckerKind,
	reason string,
) protocolchecker.CheckerCoverageEntry {
	return protocolchecker.CheckerCoverageEntry{
		Target: target, Property: property, Checker: checker,
		Status: protocolchecker.CheckerCoverageNotSupported,
		Claims: []protocolchecker.CheckerClaim{}, Evidence: []protocolchecker.CheckerEvidence{}, Reason: reason,
	}
}

func checkerDigest(encoded []byte) string {
	digest := sha256.Sum256(encoded)
	return fmt.Sprintf("sha256:%x", digest)
}

func compareCoverageEvidence(left, right protocolchecker.CheckerEvidence) int {
	if comparison := compareText(left.Kind, right.Kind); comparison != 0 {
		return comparison
	}
	if comparison := compareText(left.Digest, right.Digest); comparison != 0 {
		return comparison
	}
	return compareText(left.Declaration, right.Declaration)
}

func compareText(left, right string) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

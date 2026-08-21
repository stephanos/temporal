package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"

	"go.temporal.io/server/tests/umpire3/model-checkers/native"
	"go.temporal.io/server/tests/umpire3/model-checkers/veil"
	"go.temporal.io/server/tests/umpire3/protocol"
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

func buildCheckerCoverage(inputs checkerCoverageInputs) (protocol.CheckerCoverageManifest, error) {
	composition, err := protocol.DefaultComposition()
	if err != nil {
		return protocol.CheckerCoverageManifest{}, err
	}
	catalog, err := protocol.DefaultCatalog()
	if err != nil {
		return protocol.CheckerCoverageManifest{}, err
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return protocol.CheckerCoverageManifest{}, err
	}
	finite, err := protocol.DefaultFiniteReplayCatalog()
	if err != nil {
		return protocol.CheckerCoverageManifest{}, err
	}
	nexusView, found, err := protocol.DefaultFirstOrderView(
		protocol.TargetIDNexusCancellation, "sound")
	if err != nil {
		return protocol.CheckerCoverageManifest{}, err
	}
	if !found {
		return protocol.CheckerCoverageManifest{}, errors.New("sound Nexus first-order view is unavailable")
	}
	nativeEntry, err := checkedNativeCoverage(inputs, nexusView)
	if err != nil {
		return protocol.CheckerCoverageManifest{}, err
	}
	veilEntry, err := checkedVeilCoverage(inputs, nexusView)
	if err != nil {
		return protocol.CheckerCoverageManifest{}, err
	}

	manifest := protocol.CheckerCoverageManifest{
		FormatVersion: protocol.CheckerCoverageFormatVersion,
		CatalogHash:   catalogHash, CompositionSemanticHash: composition.SemanticHash,
		Entries: []protocol.CheckerCoverageEntry{},
	}
	for _, target := range composition.Targets {
		for _, property := range target.Properties {
			exact, err := exactCoverage(finite, nexusView, target.Identifier, property)
			if err != nil {
				return protocol.CheckerCoverageManifest{}, err
			}
			manifest.Entries = append(manifest.Entries, exact)
			if target.Identifier == protocol.TargetIDNexusCancellation &&
				property == protocol.PropertyIDNexusCancellationWonExcludesSuccess {
				manifest.Entries = append(manifest.Entries, nativeEntry, veilEntry)
				continue
			}
			manifest.Entries = append(manifest.Entries,
				unsupportedCoverage(target.Identifier, property, protocol.CheckerNative,
					"no native certificate view is declared for this target/property"),
				unsupportedCoverage(target.Identifier, property, protocol.CheckerVeil,
					"the owning Lean family does not import a Veil declaration"),
			)
		}
	}
	slices.SortFunc(manifest.Entries, func(left, right protocol.CheckerCoverageEntry) int {
		if comparison := compareText(string(left.Target), string(right.Target)); comparison != 0 {
			return comparison
		}
		if comparison := compareText(string(left.Property), string(right.Property)); comparison != 0 {
			return comparison
		}
		return compareText(string(left.Checker), string(right.Checker))
	})
	if err := manifest.Validate(); err != nil {
		return protocol.CheckerCoverageManifest{}, err
	}
	return manifest, nil
}

func exactCoverage(
	finite protocol.FiniteReplayCatalog,
	nexus protocol.FirstOrderView,
	target protocol.TargetID,
	property protocol.PropertyID,
) (protocol.CheckerCoverageEntry, error) {
	entry := protocol.CheckerCoverageEntry{
		Target: target, Property: property, Checker: protocol.CheckerExact,
		Status: protocol.CheckerCoverageChecked,
		Claims: []protocol.CheckerClaim{{
			Job: "exhaustive", ResultClass: protocol.ResultClassFiniteExhaustive,
			TrustBadge: protocol.TrustBadgeCheckedCertificate, Exact: true,
			Bounds: protocol.BackendBounds{}, Omissions: []string{},
		}},
	}
	if target == nexus.Target && property == nexus.Property {
		encoded, err := nexus.CanonicalJSON()
		if err != nil {
			return protocol.CheckerCoverageEntry{}, err
		}
		entry.World = nexus.World
		entry.Variant = nexus.Variant
		entry.SemanticHash = nexus.SemanticHash
		entry.Evidence = []protocol.CheckerEvidence{{
			Kind: "first-order-exact-view", Digest: checkerDigest(encoded),
			Declaration: nexus.Relation.Declaration,
		}}
		return entry, nil
	}
	view, found := finite.Target(target, property)
	if !found {
		return protocol.CheckerCoverageEntry{},
			fmt.Errorf("no exact finite replay view for %q/%q", target, property)
	}
	encoded, err := json.Marshal(view)
	if err != nil {
		return protocol.CheckerCoverageEntry{}, err
	}
	entry.World = view.World
	entry.Variant = view.Variant
	entry.SemanticHash = view.SemanticHash
	entry.Evidence = []protocol.CheckerEvidence{{
		Kind: "finite-replay-exact-view", Digest: checkerDigest(encoded),
		Declaration: view.Relation.Declaration,
	}}
	return entry, nil
}

func checkedNativeCoverage(
	inputs checkerCoverageInputs,
	view protocol.FirstOrderView,
) (protocol.CheckerCoverageEntry, error) {
	certificateBytes, err := os.ReadFile(inputs.nativeCertificate)
	if err != nil {
		return protocol.CheckerCoverageEntry{}, fmt.Errorf("read native certificate: %w", err)
	}
	certificate, err := native.DecodeCertificate(bytes.NewReader(certificateBytes),
		protocol.DefaultDecodeLimit, view)
	if err != nil {
		return protocol.CheckerCoverageEntry{}, fmt.Errorf("validate native certificate: %w", err)
	}
	receiptBytes, err := os.ReadFile(inputs.nativeReceipt)
	if err != nil {
		return protocol.CheckerCoverageEntry{}, fmt.Errorf("read native receipt: %w", err)
	}
	receipt, err := native.DecodeReceipt(bytes.NewReader(receiptBytes),
		protocol.DefaultDecodeLimit, certificate)
	if err != nil {
		return protocol.CheckerCoverageEntry{}, fmt.Errorf("validate native receipt: %w", err)
	}
	canonicalCertificate, err := certificate.CanonicalJSON(view)
	if err != nil {
		return protocol.CheckerCoverageEntry{}, err
	}
	canonicalReceipt, err := receipt.CanonicalJSON(certificate)
	if err != nil {
		return protocol.CheckerCoverageEntry{}, err
	}
	benchmarkBytes, err := os.ReadFile(inputs.nativeBenchmark)
	if err != nil {
		return protocol.CheckerCoverageEntry{}, fmt.Errorf("read native benchmark: %w", err)
	}
	benchmark, err := native.DecodeBenchmarkReport(bytes.NewReader(benchmarkBytes),
		protocol.DefaultDecodeLimit, view, certificate, receipt)
	if err != nil {
		return protocol.CheckerCoverageEntry{}, fmt.Errorf("validate native benchmark: %w", err)
	}
	canonicalBenchmark, err := benchmark.CanonicalJSON(view, certificate, receipt)
	if err != nil {
		return protocol.CheckerCoverageEntry{}, err
	}
	evidence := []protocol.CheckerEvidence{
		{Kind: "native-certificate", Digest: checkerDigest(canonicalCertificate)},
		{Kind: "native-lean-receipt", Digest: checkerDigest(canonicalReceipt)},
		{Kind: "native-scale-benchmark", Digest: checkerDigest(canonicalBenchmark)},
	}
	slices.SortFunc(evidence, compareCoverageEvidence)
	return protocol.CheckerCoverageEntry{
		Target: view.Target, Property: view.Property, Checker: protocol.CheckerNative,
		Status: protocol.CheckerCoverageChecked, World: view.World, Variant: view.Variant,
		SemanticHash: view.SemanticHash,
		Claims: []protocol.CheckerClaim{{
			Job: "exhaustive", ResultClass: receipt.ResultClass, TrustBadge: receipt.TrustBadge,
			Exact: true, Bounds: protocol.BackendBounds{}, Omissions: []string{},
		}},
		Evidence: evidence,
	}, nil
}

func checkedVeilCoverage(
	inputs checkerCoverageInputs,
	view protocol.FirstOrderView,
) (protocol.CheckerCoverageEntry, error) {
	bindingBytes, err := os.ReadFile(inputs.veilBinding)
	if err != nil {
		return protocol.CheckerCoverageEntry{}, fmt.Errorf("read Veil binding: %w", err)
	}
	binding, err := veil.DecodeBindingArtifact(bytes.NewReader(bindingBytes), protocol.DefaultDecodeLimit)
	if err != nil {
		return protocol.CheckerCoverageEntry{}, fmt.Errorf("validate Veil binding: %w", err)
	}
	if err := binding.ValidateAgainst(view); err != nil {
		return protocol.CheckerCoverageEntry{}, err
	}
	canonicalBinding, err := binding.CanonicalJSON()
	if err != nil {
		return protocol.CheckerCoverageEntry{}, err
	}
	evidence := []protocol.CheckerEvidence{{
		Kind: "veil-semantic-binding", Digest: checkerDigest(canonicalBinding),
		Declaration: binding.Binding.SemanticBinding.Declaration,
	}}
	claims := make([]protocol.CheckerClaim, 0, len(inputs.veilResults))
	for _, path := range inputs.veilResults {
		if path == "" {
			return protocol.CheckerCoverageEntry{}, errors.New("Veil result path cannot be empty")
		}
		resultBytes, err := os.ReadFile(path)
		if err != nil {
			return protocol.CheckerCoverageEntry{}, fmt.Errorf("read Veil result %q: %w", path, err)
		}
		result, err := protocol.DecodeBackendResult(
			bytes.NewReader(resultBytes), protocol.DefaultDecodeLimit)
		if err != nil {
			return protocol.CheckerCoverageEntry{}, fmt.Errorf("validate Veil result %q: %w", path, err)
		}
		if result.Target != view.Target || result.Property != view.Property ||
			result.World != view.World || result.Variant != view.Variant ||
			result.SemanticHash != view.SemanticHash ||
			result.GeneratedArtifactDigest != binding.ArtifactDigest {
			return protocol.CheckerCoverageEntry{},
				fmt.Errorf("Veil result %q does not match its checked binding and view", path)
		}
		canonical, err := result.CanonicalJSON()
		if err != nil {
			return protocol.CheckerCoverageEntry{}, err
		}
		claims = append(claims, protocol.CheckerClaim{
			Job: string(result.Job), ResultClass: result.ResultClass, TrustBadge: result.TrustBadge,
			Exact: result.Exact, Bounds: result.Bounds,
			Omissions: append([]string{}, result.Omissions...),
		})
		evidence = append(evidence, protocol.CheckerEvidence{
			Kind: "veil-" + string(result.Job) + "-result", Digest: checkerDigest(canonical),
		})
	}
	slices.SortFunc(claims, func(left, right protocol.CheckerClaim) int {
		return compareText(left.Job, right.Job)
	})
	slices.SortFunc(evidence, compareCoverageEvidence)
	return protocol.CheckerCoverageEntry{
		Target: view.Target, Property: view.Property, Checker: protocol.CheckerVeil,
		Status: protocol.CheckerCoverageChecked, World: view.World, Variant: view.Variant,
		SemanticHash: view.SemanticHash, Claims: claims, Evidence: evidence,
	}, nil
}

func unsupportedCoverage(
	target protocol.TargetID,
	property protocol.PropertyID,
	checker protocol.CheckerKind,
	reason string,
) protocol.CheckerCoverageEntry {
	return protocol.CheckerCoverageEntry{
		Target: target, Property: property, Checker: checker,
		Status: protocol.CheckerCoverageNotSupported,
		Claims: []protocol.CheckerClaim{}, Evidence: []protocol.CheckerEvidence{}, Reason: reason,
	}
}

func checkerDigest(encoded []byte) string {
	digest := sha256.Sum256(encoded)
	return fmt.Sprintf("sha256:%x", digest)
}

func compareCoverageEvidence(left, right protocol.CheckerEvidence) int {
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

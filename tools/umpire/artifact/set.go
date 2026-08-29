package artifact

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

const (
	artifactSetFormat         = "umpire-artifact-set/v2"
	artifactSetIdentityDomain = "umpire.artifact-set-identity/v2"
	artifactSetChecksumDomain = "umpire.artifact-set/v2"
)

var artifactSetPaths = [...]string{
	"artifacts/experiment.json",
	"artifacts/runtime-configuration.json",
	"artifacts/experiment-run.json",
	"artifacts/raw-evidence.json",
	"artifacts/evidence.json",
	"artifacts/result.json",
}

// SetMember is one exact canonical Artifact document at its manifest path.
type SetMember struct {
	Path    string
	Encoded []byte
}

type artifactSetManifestMember struct {
	Path                string `json:"path"`
	FormatVersion       string `json:"formatVersion"`
	ArtifactChecksum    string `json:"artifactChecksum"`
	BehaviorFingerprint string `json:"behaviorFingerprint"`
	ProvenanceChecksum  string `json:"provenanceChecksum"`
}

type artifactSetManifestPreimage struct {
	FormatVersion       string                      `json:"formatVersion"`
	ArtifactSetIdentity string                      `json:"artifactSetIdentity"`
	Members             []artifactSetManifestMember `json:"members"`
}

type artifactSetManifest struct {
	FormatVersion       string                      `json:"formatVersion"`
	ArtifactSetIdentity string                      `json:"artifactSetIdentity"`
	Members             []artifactSetManifestMember `json:"members"`
	ArtifactSetChecksum string                      `json:"artifactSetChecksum"`
}

var artifactSetManifestDecoder = Decoder[artifactSetManifest]{
	Format:   artifactSetFormat,
	Bounds:   Bounds{CollectionLimit: artifactSetManifestCollectionLimit, StringLimit: artifactSetManifestStringLimit},
	Validate: validateArtifactSetManifest,
	Canonical: func(manifest artifactSetManifest) ([]byte, error) {
		return CanonicalPretty(manifest)
	},
	ArtifactChecksum: verifyArtifactSetManifestChecksum,
	Closure:          validateArtifactSetManifestClosure,
}

// AdmittedSet is an exact, closed Artifact set and its deterministic manifest.
type AdmittedSet struct {
	members        []SetMember
	manifest       artifactSetManifest
	manifestBytes  []byte
	manifestSHA256 string
	executable     *admittedExecutableValues
}

type admittedExecutableValues struct {
	experiment           artifactv2.Experiment
	runtimeConfiguration artifactv2.RuntimeConfiguration
}

// ExecutableSet is the exact typed two-member prefix retained during Artifact set admission.
type ExecutableSet struct {
	admitted AdmittedSet
}

type admittedExecutionMembers struct {
	run         artifactv2.ExperimentRun
	rawEvidence artifactv2.RawEvidence
	rows        []artifactSetManifestMember
}

type artifactSetFormatClassifier func([]byte) error

var artifactSetMemberFormatClassifiers = [...]artifactSetFormatClassifier{
	func(encoded []byte) error { return classifyArtifactSetFormat(experimentV2Decoder, encoded) },
	func(encoded []byte) error { return classifyArtifactSetFormat(runtimeConfigurationV2Decoder, encoded) },
	func(encoded []byte) error { return classifyArtifactSetFormat(experimentRunV2Decoder, encoded) },
	func(encoded []byte) error { return classifyArtifactSetFormat(rawEvidenceV2Decoder, encoded) },
	func(encoded []byte) error { return classifyArtifactSetFormat(evidenceV2Decoder, encoded) },
	func(encoded []byte) error { return classifyArtifactSetFormat(resultV2Decoder, encoded) },
}

// Identity returns the content-derived Artifact set identity.
func (s AdmittedSet) Identity() string {
	return s.manifest.ArtifactSetIdentity
}

// Checksum returns the checksum of the manifest preimage.
func (s AdmittedSet) Checksum() string {
	return s.manifest.ArtifactSetChecksum
}

// ManifestSHA256 returns the raw SHA-256 of the complete manifest bytes.
func (s AdmittedSet) ManifestSHA256() string {
	return s.manifestSHA256
}

// ManifestBytes returns a copy of the exact deterministic manifest bytes.
func (s AdmittedSet) ManifestBytes() []byte {
	return append([]byte(nil), s.manifestBytes...)
}

// Executable returns the already-admitted typed values only for an exact two-member set.
func (s AdmittedSet) Executable() (ExecutableSet, bool) {
	if s.executable == nil || len(s.members) != 2 {
		return ExecutableSet{}, false
	}
	return ExecutableSet{admitted: s}, true
}

// AdmittedSet returns the exact admitted set associated with this typed projection.
func (s ExecutableSet) AdmittedSet() AdmittedSet {
	return cloneAdmittedSet(s.admitted)
}

// Experiment returns an immutable copy of the admitted ExperimentSpec value.
func (s ExecutableSet) Experiment() artifactv2.Experiment {
	if s.admitted.executable == nil {
		return artifactv2.Experiment{}
	}
	return cloneExperiment(s.admitted.executable.experiment)
}

// RuntimeConfiguration returns an immutable copy of the admitted RuntimeConfiguration value.
func (s ExecutableSet) RuntimeConfiguration() artifactv2.RuntimeConfiguration {
	if s.admitted.executable == nil {
		return artifactv2.RuntimeConfiguration{}
	}
	return cloneRuntimeConfiguration(s.admitted.executable.runtimeConfiguration)
}

// AdmitSet admits only an exact executable, execution, or evaluation closure.
func AdmitSet(members []SetMember) (AdmittedSet, error) {
	if err := classifyArtifactSetMemberFormats(members); err != nil {
		return AdmittedSet{}, err
	}
	if len(members) != 2 && len(members) != 4 && len(members) != 6 {
		return AdmittedSet{}, wrapAdmission(ErrorClosure,
			fmt.Errorf("artifact set has %d members; expected 2, 4, or 6", len(members)))
	}

	experiment, err := DecodeExperimentV2(members[0].Encoded)
	if err != nil {
		return AdmittedSet{}, err
	}
	runtimeConfiguration, err := DecodeRuntimeConfigurationV2(members[1].Encoded)
	if err != nil {
		return AdmittedSet{}, err
	}
	if err := ValidateRuntimeConfigurationV2Closure(runtimeConfiguration, experiment); err != nil {
		return AdmittedSet{}, err
	}

	rows := make([]artifactSetManifestMember, 0, len(members))
	experimentBinding, err := artifactv2.ExperimentArtifactBinding(experiment)
	if err != nil {
		return AdmittedSet{}, wrapAdmission(ErrorClosure, err)
	}
	rows = append(rows,
		manifestMember(members[0].Path, experimentBinding),
		manifestMember(members[1].Path, artifactv2.RuntimeConfigurationArtifactBinding(runtimeConfiguration)),
	)

	if len(members) >= 4 {
		execution, executionErr := admitExecutionMembers(members, experiment, runtimeConfiguration)
		if executionErr != nil {
			return AdmittedSet{}, executionErr
		}
		rows = append(rows, execution.rows...)
		if len(members) == 6 {
			evaluationRows, evaluationErr := admitEvaluationMembers(
				members, experiment, runtimeConfiguration, execution,
			)
			if evaluationErr != nil {
				return AdmittedSet{}, evaluationErr
			}
			rows = append(rows, evaluationRows...)
		}
	}

	if err := validateArtifactSetPaths(members); err != nil {
		return AdmittedSet{}, wrapAdmission(ErrorClosure, err)
	}
	return buildAdmittedSet(members, rows, experiment, runtimeConfiguration)
}

func admitExecutionMembers(
	members []SetMember,
	experiment artifactv2.Experiment,
	runtimeConfiguration artifactv2.RuntimeConfiguration,
) (admittedExecutionMembers, error) {
	run, err := DecodeExperimentRunV2(members[2].Encoded)
	if err != nil {
		return admittedExecutionMembers{}, err
	}
	rawEvidence, err := DecodeRawEvidenceV2(members[3].Encoded)
	if err != nil {
		return admittedExecutionMembers{}, err
	}
	if err := ValidateRawEvidenceV2Closure(
		rawEvidence, experiment, runtimeConfiguration, run,
	); err != nil {
		return admittedExecutionMembers{}, err
	}
	return admittedExecutionMembers{
		run:         run,
		rawEvidence: rawEvidence,
		rows: []artifactSetManifestMember{
			manifestMember(members[2].Path, artifactv2.ExperimentRunArtifactBinding(run)),
			manifestMember(members[3].Path, artifactv2.RawEvidenceArtifactBinding(rawEvidence)),
		},
	}, nil
}

func admitEvaluationMembers(
	members []SetMember,
	experiment artifactv2.Experiment,
	runtimeConfiguration artifactv2.RuntimeConfiguration,
	execution admittedExecutionMembers,
) ([]artifactSetManifestMember, error) {
	evidence, err := DecodeEvidenceV2(members[4].Encoded)
	if err != nil {
		return nil, err
	}
	result, err := DecodeResultV2(members[5].Encoded)
	if err != nil {
		return nil, err
	}
	if err := ValidateResultV2Closure(
		result, experiment, runtimeConfiguration, execution.run, execution.rawEvidence, evidence,
	); err != nil {
		return nil, err
	}
	if result.ImplementationLink.SourceTarget.DefinitionID != experiment.Plan.TargetDefinitionID ||
		result.ImplementationLink.SourceTarget.BehaviorFingerprint !=
			experiment.Plan.TargetBehaviorFingerprint {
		return nil, wrapAdmission(ErrorClosure,
			errors.New("result Implementation Link source target does not match ExperimentSpec"))
	}
	return []artifactSetManifestMember{
		manifestMember(members[4].Path, artifactv2.EvidenceArtifactBinding(evidence)),
		manifestMember(members[5].Path, artifactv2.ArtifactBinding{
			FormatVersion:       result.FormatVersion,
			ArtifactChecksum:    result.ArtifactChecksum,
			BehaviorFingerprint: result.BehaviorFingerprint,
			ProvenanceChecksum:  result.ProvenanceChecksum,
		}),
	}, nil
}

// AdmitSetManifest admits an existing manifest only when it is the exact manifest for the members.
func AdmitSetManifest(encodedManifest []byte, members []SetMember) (AdmittedSet, error) {
	manifestFormatErr := classifyArtifactSetFormat(artifactSetManifestDecoder, encodedManifest)
	memberFormatErr := classifyArtifactSetMemberFormats(members)
	if err := earlierArtifactSetFormatError(manifestFormatErr, memberFormatErr); err != nil {
		return AdmittedSet{}, err
	}
	admitted, err := AdmitSet(members)
	if err != nil {
		return AdmittedSet{}, err
	}
	manifest, err := artifactSetManifestDecoder.Decode(encodedManifest)
	if err != nil {
		return AdmittedSet{}, err
	}
	canonical, err := CanonicalPretty(manifest)
	if err != nil {
		return AdmittedSet{}, wrapAdmission(ErrorMalformedValue, err)
	}
	if !bytes.Equal(canonical, admitted.manifestBytes) {
		return AdmittedSet{}, wrapAdmission(ErrorClosure,
			errors.New("artifact set manifest does not match its exact member closure"))
	}
	return admitted, nil
}

func classifyArtifactSetMemberFormats(members []SetMember) error {
	var selected error
	for index := 0; index < len(members) && index < len(artifactSetMemberFormatClassifiers); index++ {
		err := artifactSetMemberFormatClassifiers[index](members[index].Encoded)
		selected = earlierArtifactSetFormatError(selected, err)
	}
	return selected
}

func classifyArtifactSetFormat[T any](decoder Decoder[T], encoded []byte) error {
	_, err := inspectAdmissionFormat(decoder, encoded, standardStructuralLimits)
	return err
}

func earlierArtifactSetFormatError(first error, second error) error {
	if first == nil {
		return second
	}
	if second == nil {
		return first
	}
	firstCode, firstOK := CodeOf(first)
	secondCode, secondOK := CodeOf(second)
	if secondOK && (!firstOK || artifactSetFormatErrorRank(secondCode) < artifactSetFormatErrorRank(firstCode)) {
		return second
	}
	return first
}

func artifactSetFormatErrorRank(code ErrorCode) int {
	switch code {
	case ErrorByteLimit:
		return 0
	case ErrorSyntax:
		return 1
	case ErrorTokenLimit:
		return 2
	case ErrorDepthLimit:
		return 3
	case ErrorDuplicateKey:
		return 4
	case ErrorCaseCollision:
		return 5
	case ErrorUnsupportedFormat:
		return 6
	case ErrorWrongFamily:
		return 7
	default:
		return 8
	}
}

func manifestMember(path string, binding artifactv2.ArtifactBinding) artifactSetManifestMember {
	return artifactSetManifestMember{
		Path:                path,
		FormatVersion:       binding.FormatVersion,
		ArtifactChecksum:    binding.ArtifactChecksum,
		BehaviorFingerprint: binding.BehaviorFingerprint,
		ProvenanceChecksum:  binding.ProvenanceChecksum,
	}
}

func validateArtifactSetPaths(members []SetMember) error {
	seen := make(map[string]struct{}, len(members))
	for index, member := range members {
		if _, duplicate := seen[member.Path]; duplicate {
			return fmt.Errorf("artifact set path %q occurs more than once", member.Path)
		}
		seen[member.Path] = struct{}{}
		if member.Path != artifactSetPaths[index] {
			return fmt.Errorf("artifact set member %d has path %q; expected %q",
				index, member.Path, artifactSetPaths[index])
		}
	}
	return nil
}

func buildAdmittedSet(
	members []SetMember,
	rows []artifactSetManifestMember,
	experiment artifactv2.Experiment,
	runtimeConfiguration artifactv2.RuntimeConfiguration,
) (AdmittedSet, error) {
	memberRows, err := CanonicalPretty(rows)
	if err != nil {
		return AdmittedSet{}, wrapAdmission(ErrorMalformedValue, err)
	}
	identity := "umpire.artifact-set." + rawSHA256(artifactSetIdentityDomain, memberRows)
	preimage := artifactSetManifestPreimage{
		FormatVersion:       artifactSetFormat,
		ArtifactSetIdentity: identity,
		Members:             rows,
	}
	preimageBytes, err := CanonicalPretty(preimage)
	if err != nil {
		return AdmittedSet{}, wrapAdmission(ErrorMalformedValue, err)
	}
	checksum := "sha256:" + rawSHA256(artifactSetChecksumDomain, preimageBytes)
	manifest := artifactSetManifest{
		FormatVersion:       preimage.FormatVersion,
		ArtifactSetIdentity: preimage.ArtifactSetIdentity,
		Members:             preimage.Members,
		ArtifactSetChecksum: checksum,
	}
	manifestBytes, err := CanonicalPretty(manifest)
	if err != nil {
		return AdmittedSet{}, wrapAdmission(ErrorMalformedValue, err)
	}
	admitted := AdmittedSet{
		members:        cloneSetMembers(members),
		manifest:       manifest,
		manifestBytes:  manifestBytes,
		manifestSHA256: "sha256:" + rawSHA256("", manifestBytes),
	}
	if len(members) == 2 {
		admitted.executable = &admittedExecutableValues{
			experiment:           cloneExperiment(experiment),
			runtimeConfiguration: cloneRuntimeConfiguration(runtimeConfiguration),
		}
	}
	return admitted, nil
}

func artifactSetManifestCollectionLimit(path JSONPath, kind CollectionKind) int {
	if path == "$.members" && kind == CollectionArray {
		return len(artifactSetPaths)
	}
	return 0
}

func artifactSetManifestStringLimit(JSONPath) int {
	return MaximumIdentityBytes
}

func validateArtifactSetManifest(manifest artifactSetManifest) error {
	if manifest.FormatVersion != artifactSetFormat {
		return fmt.Errorf("unsupported format %q", manifest.FormatVersion)
	}
	if !validArtifactSetIdentity(manifest.ArtifactSetIdentity) ||
		!artifactv2.ValidDigest(manifest.ArtifactSetChecksum) {
		return errors.New("artifact set identity or checksum is malformed")
	}
	for _, member := range manifest.Members {
		if !artifactv2.ValidDigest(member.ArtifactChecksum) ||
			!artifactv2.ValidDigest(member.BehaviorFingerprint) ||
			!artifactv2.ValidDigest(member.ProvenanceChecksum) {
			return fmt.Errorf("artifact set member %q has a malformed digest", member.Path)
		}
	}
	return nil
}

func verifyArtifactSetManifestChecksum(manifest artifactSetManifest) error {
	preimage := artifactSetManifestPreimage{
		FormatVersion:       manifest.FormatVersion,
		ArtifactSetIdentity: manifest.ArtifactSetIdentity,
		Members:             manifest.Members,
	}
	encoded, err := CanonicalPretty(preimage)
	if err != nil {
		return err
	}
	expected := "sha256:" + rawSHA256(artifactSetChecksumDomain, encoded)
	if manifest.ArtifactSetChecksum != expected {
		return fmt.Errorf("artifact set checksum mismatch: got %q, want %q",
			manifest.ArtifactSetChecksum, expected)
	}
	return nil
}

func validateArtifactSetManifestClosure(manifest artifactSetManifest) error {
	if len(manifest.Members) != 2 && len(manifest.Members) != 4 && len(manifest.Members) != 6 {
		return fmt.Errorf("artifact set manifest has %d members; expected 2, 4, or 6",
			len(manifest.Members))
	}
	for index, member := range manifest.Members {
		if member.Path != artifactSetPaths[index] {
			return fmt.Errorf("artifact set manifest member %d has path %q; expected %q",
				index, member.Path, artifactSetPaths[index])
		}
	}
	memberRows, err := CanonicalPretty(manifest.Members)
	if err != nil {
		return err
	}
	expectedIdentity := "umpire.artifact-set." + rawSHA256(artifactSetIdentityDomain, memberRows)
	if manifest.ArtifactSetIdentity != expectedIdentity {
		return fmt.Errorf("artifact set identity mismatch: got %q, want %q",
			manifest.ArtifactSetIdentity, expectedIdentity)
	}
	return nil
}

func validArtifactSetIdentity(identity string) bool {
	const prefix = "umpire.artifact-set."
	digest := strings.TrimPrefix(identity, prefix)
	if digest == identity || len(digest) != sha256.Size*2 {
		return false
	}
	for _, character := range digest {
		if character < '0' || (character > '9' && character < 'a') || character > 'f' {
			return false
		}
	}
	return true
}

func rawSHA256(domain string, canonical []byte) string {
	hash := sha256.New()
	if domain != "" {
		_, _ = hash.Write([]byte(domain))
		_, _ = hash.Write([]byte{'\n'})
	}
	_, _ = hash.Write(canonical)
	return hex.EncodeToString(hash.Sum(nil))
}

func cloneSetMembers(members []SetMember) []SetMember {
	cloned := make([]SetMember, len(members))
	for index, member := range members {
		cloned[index] = SetMember{Path: member.Path, Encoded: append([]byte(nil), member.Encoded...)}
	}
	return cloned
}

func cloneAdmittedSet(admitted AdmittedSet) AdmittedSet {
	cloned := admitted
	cloned.members = cloneSetMembers(admitted.members)
	cloned.manifest.Members = append([]artifactSetManifestMember(nil), admitted.manifest.Members...)
	cloned.manifestBytes = append([]byte(nil), admitted.manifestBytes...)
	if admitted.executable != nil {
		cloned.executable = &admittedExecutableValues{
			experiment:           cloneExperiment(admitted.executable.experiment),
			runtimeConfiguration: cloneRuntimeConfiguration(admitted.executable.runtimeConfiguration),
		}
	}
	return cloned
}

func cloneExperiment(document artifactv2.Experiment) artifactv2.Experiment {
	cloned := document
	cloned.Properties = append([]artifactv2.Property(nil), document.Properties...)
	for index := range cloned.Properties {
		cloned.Properties[index].RequirementDefinitionIDs = append(
			[]string(nil), document.Properties[index].RequirementDefinitionIDs...,
		)
	}
	cloned.ObservationRequirementDefinitionIDs = append(
		[]string(nil), document.ObservationRequirementDefinitionIDs...,
	)
	cloned.Provenance = cloneProvenance(document.Provenance)
	cloned.Plan = cloneDrivePlan(document.Plan)
	return cloned
}

func cloneDrivePlan(plan artifactv2.DrivePlan) artifactv2.DrivePlan {
	cloned := plan
	cloned.Bindings = append([]artifactv2.Binding(nil), plan.Bindings...)
	cloned.SymbolicRoles = append([]artifactv2.Role(nil), plan.SymbolicRoles...)
	cloned.ModelPreconditions = append([]artifactv2.Precondition(nil), plan.ModelPreconditions...)
	for index := range cloned.ModelPreconditions {
		cloned.ModelPreconditions[index].Left.Value = cloneModelValuePointer(
			plan.ModelPreconditions[index].Left.Value,
		)
		cloned.ModelPreconditions[index].Right.Value = cloneModelValuePointer(
			plan.ModelPreconditions[index].Right.Value,
		)
	}
	cloned.RequestedActions = append([]artifactv2.ModelValue(nil), plan.RequestedActions...)
	cloned.ModelOutcomes = append([]artifactv2.ModelValue(nil), plan.ModelOutcomes...)
	cloned.ResultingStates = append([]artifactv2.ModelValue(nil), plan.ResultingStates...)
	cloned.LinearExtension = append([]artifactv2.Occurrence(nil), plan.LinearExtension...)
	for index := range cloned.LinearExtension {
		cloned.LinearExtension[index].AuthoredDefinitionID = cloneStringPointer(
			plan.LinearExtension[index].AuthoredDefinitionID,
		)
	}
	cloned.SelectedChoices = append([]artifactv2.ModelValue(nil), plan.SelectedChoices...)
	cloned.SelectedVariants = append([]artifactv2.ModelValue(nil), plan.SelectedVariants...)
	cloned.RequestedFaults = append([]artifactv2.ModelValue(nil), plan.RequestedFaults...)
	cloned.CapabilityRequirementDefinitionIDs = append(
		[]string(nil), plan.CapabilityRequirementDefinitionIDs...,
	)
	cloned.Checkpoints = append([]artifactv2.Checkpoint(nil), plan.Checkpoints...)
	for index := range cloned.Checkpoints {
		cloned.Checkpoints[index].Observations = append(
			[]artifactv2.ModelValue(nil), plan.Checkpoints[index].Observations...,
		)
	}
	cloned.KnownGaps = cloneKnownGaps(plan.KnownGaps)
	cloned.Provenance = cloneProvenance(plan.Provenance)
	return cloned
}

func cloneRuntimeConfiguration(
	document artifactv2.RuntimeConfiguration,
) artifactv2.RuntimeConfiguration {
	cloned := document
	cloned.AuthorityProfile.RequiredCapabilityDefinitionIDs = append(
		[]string(nil), document.AuthorityProfile.RequiredCapabilityDefinitionIDs...,
	)
	cloned.PhaseLimits = append([]artifactv2.PhaseLimit(nil), document.PhaseLimits...)
	cloned.ParticipantBindings = append(
		[]artifactv2.ParticipantBinding(nil), document.ParticipantBindings...,
	)
	for index := range cloned.ParticipantBindings {
		cloned.ParticipantBindings[index].CapabilityDefinitionIDs = append(
			[]string(nil), document.ParticipantBindings[index].CapabilityDefinitionIDs...,
		)
	}
	cloned.KnownGaps = cloneKnownGaps(document.KnownGaps)
	cloned.Provenance = cloneProvenance(document.Provenance)
	return cloned
}

func cloneProvenance(provenance artifactv2.Provenance) artifactv2.Provenance {
	return artifactv2.Provenance{
		SourceDefinitionIDs: append([]string(nil), provenance.SourceDefinitionIDs...),
		SourceLocations:     append([]artifactv2.SourceLocation(nil), provenance.SourceLocations...),
	}
}

func cloneKnownGaps(gaps []artifactv2.KnownGap) []artifactv2.KnownGap {
	cloned := append([]artifactv2.KnownGap(nil), gaps...)
	for index := range cloned {
		cloned[index].Subject = cloneStringPointer(gaps[index].Subject)
		cloned[index].Detail = cloneStringPointer(gaps[index].Detail)
	}
	return cloned
}

func cloneModelValuePointer(value *artifactv2.ModelValue) *artifactv2.ModelValue {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

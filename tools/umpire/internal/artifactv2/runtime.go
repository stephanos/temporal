package artifactv2

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

const (
	RuntimeConfigurationFormat = "umpire-runtime-configuration/v2"
	ExperimentRunFormat        = "umpire-experiment-run/v2"
)

const (
	provenanceChecksumDomain           = "umpire.provenance/v2"
	runtimeConfigurationChecksumDomain = "umpire.runtime-configuration/v2"
	experimentRunChecksumDomain        = "umpire.experiment-run/v2"
)

var executionPhases = [...]string{
	"preparation",
	"realization",
	"observation",
	"isolation",
	"cleanup",
}

type ArtifactBinding struct {
	FormatVersion       string `json:"formatVersion"`
	ArtifactChecksum    string `json:"artifactChecksum"`
	BehaviorFingerprint string `json:"behaviorFingerprint"`
	ProvenanceChecksum  string `json:"provenanceChecksum"`
}

type AuthorityProfile struct {
	DefinitionID                    string   `json:"definitionId"`
	Version                         Natural  `json:"version"`
	BehaviorFingerprint             string   `json:"behaviorFingerprint"`
	RequiredCapabilityDefinitionIDs []string `json:"requiredCapabilityDefinitionIds"`
}

type PhaseLimit struct {
	Phase                string  `json:"phase"`
	DurationMilliseconds Natural `json:"durationMilliseconds"`
	MaxAttempts          Natural `json:"maxAttempts"`
	MaxRecords           Natural `json:"maxRecords"`
	MaxBytes             Natural `json:"maxBytes"`
}

type ObservationConfiguration struct {
	ProfileDefinitionID        string `json:"profileDefinitionId"`
	ProfileBehaviorFingerprint string `json:"profileBehaviorFingerprint"`
	ProgramDefinitionID        string `json:"programDefinitionId"`
	ProgramBehaviorFingerprint string `json:"programBehaviorFingerprint"`
	MappingDefinitionID        string `json:"mappingDefinitionId"`
	MappingBehaviorFingerprint string `json:"mappingBehaviorFingerprint"`
}

type ParticipantBinding struct {
	ParticipantDefinitionID    string   `json:"participantDefinitionId"`
	ProtocolDefinitionID       string   `json:"protocolDefinitionId"`
	ProtocolVersion            Natural  `json:"protocolVersion"`
	ProgramDefinitionID        string   `json:"programDefinitionId"`
	ProgramBehaviorFingerprint string   `json:"programBehaviorFingerprint"`
	CapabilityDefinitionIDs    []string `json:"capabilityDefinitionIds"`
}

type RuntimeConfiguration struct {
	FormatVersion             string                   `json:"formatVersion"`
	ConfigurationDefinitionID string                   `json:"configurationDefinitionId"`
	BehaviorFingerprint       string                   `json:"behaviorFingerprint"`
	Experiment                ArtifactBinding          `json:"experiment"`
	AuthorityProfile          AuthorityProfile         `json:"authorityProfile"`
	PhaseLimits               []PhaseLimit             `json:"phaseLimits"`
	Observation               ObservationConfiguration `json:"observation"`
	ParticipantBindings       []ParticipantBinding     `json:"participantBindings"`
	KnownGaps                 []KnownGap               `json:"knownGaps"`
	Provenance                Provenance               `json:"provenance"`
	ProvenanceChecksum        string                   `json:"provenanceChecksum"`
	ArtifactChecksum          string                   `json:"artifactChecksum,omitempty"`
}

type PhaseOutcome struct {
	Phase                string   `json:"phase"`
	Status               string   `json:"status"`
	StartedAtUnixMillis  *Natural `json:"startedAtUnixMillis"`
	FinishedAtUnixMillis *Natural `json:"finishedAtUnixMillis"`
	Code                 *string  `json:"code"`
}

type ControlAttempt struct {
	OccurrenceDefinitionID  string  `json:"occurrenceDefinitionId"`
	ActionDefinitionID      string  `json:"actionDefinitionId"`
	Attempt                 Natural `json:"attempt"`
	ReceiptFactDefinitionID *string `json:"receiptFactDefinitionId"`
	Status                  string  `json:"status"`
	Code                    *string `json:"code"`
}

type SourceClosure struct {
	SourceDefinitionID string  `json:"sourceDefinitionId"`
	Status             string  `json:"status"`
	RecordCount        Natural `json:"recordCount"`
	ByteCount          Natural `json:"byteCount"`
}

type CleanupOutcome struct {
	Status          string  `json:"status"`
	OpenHandleCount Natural `json:"openHandleCount"`
	Code            *string `json:"code"`
}

type ExperimentRun struct {
	FormatVersion        string           `json:"formatVersion"`
	RunIdentity          string           `json:"runIdentity"`
	BehaviorFingerprint  string           `json:"behaviorFingerprint"`
	Experiment           ArtifactBinding  `json:"experiment"`
	RuntimeConfiguration ArtifactBinding  `json:"runtimeConfiguration"`
	Attempt              Natural          `json:"attempt"`
	OperationalStatus    string           `json:"operationalStatus"`
	PhaseOutcomes        []PhaseOutcome   `json:"phaseOutcomes"`
	ControlAttempts      []ControlAttempt `json:"controlAttempts"`
	SourceClosures       []SourceClosure  `json:"sourceClosures"`
	Cleanup              CleanupOutcome   `json:"cleanup"`
	Limits               []PhaseLimit     `json:"limits"`
	KnownGaps            []KnownGap       `json:"knownGaps"`
	Provenance           Provenance       `json:"provenance"`
	ProvenanceChecksum   string           `json:"provenanceChecksum"`
	ArtifactChecksum     string           `json:"artifactChecksum,omitempty"`
}

func CanonicalRuntimeConfigurationBytes(document RuntimeConfiguration) ([]byte, error) {
	return encodeJSONLine(document)
}

func ExpectedProvenanceChecksum(provenance Provenance) (string, error) {
	encoded, err := encodeJSONLine(provenance)
	if err != nil {
		return "", err
	}
	return derive(provenanceChecksumDomain, encoded), nil
}

func ExpectedRuntimeConfigurationChecksum(document RuntimeConfiguration) (string, error) {
	document.ArtifactChecksum = ""
	encoded, err := encodeJSONLine(document)
	if err != nil {
		return "", err
	}
	return derive(runtimeConfigurationChecksumDomain, encoded), nil
}

func SealRuntimeConfiguration(document RuntimeConfiguration) (RuntimeConfiguration, error) {
	provenanceChecksum, err := ExpectedProvenanceChecksum(document.Provenance)
	if err != nil {
		return RuntimeConfiguration{}, err
	}
	document.ProvenanceChecksum = provenanceChecksum
	artifactChecksum, err := ExpectedRuntimeConfigurationChecksum(document)
	if err != nil {
		return RuntimeConfiguration{}, err
	}
	document.ArtifactChecksum = artifactChecksum
	return document, nil
}

func VerifyRuntimeConfigurationProvenanceChecksum(document RuntimeConfiguration) error {
	expected, err := ExpectedProvenanceChecksum(document.Provenance)
	if err != nil {
		return err
	}
	if document.ProvenanceChecksum != expected {
		return fmt.Errorf("RuntimeConfiguration provenance checksum mismatch: got %q, want %q",
			document.ProvenanceChecksum, expected)
	}
	return nil
}

func VerifyRuntimeConfigurationArtifactChecksum(document RuntimeConfiguration) error {
	expected, err := ExpectedRuntimeConfigurationChecksum(document)
	if err != nil {
		return err
	}
	if document.ArtifactChecksum != expected {
		return fmt.Errorf("RuntimeConfiguration artifact checksum mismatch: got %q, want %q",
			document.ArtifactChecksum, expected)
	}
	return nil
}

func ValidateRuntimeConfiguration(document RuntimeConfiguration) error {
	if document.FormatVersion != RuntimeConfigurationFormat {
		return fmt.Errorf("unsupported format %q", document.FormatVersion)
	}
	if !validDefinitionID(document.ConfigurationDefinitionID) {
		return fmt.Errorf("configuration definition ID %q is invalid", document.ConfigurationDefinitionID)
	}
	if !ValidDigest(document.BehaviorFingerprint) || !ValidDigest(document.ProvenanceChecksum) ||
		!ValidDigest(document.ArtifactChecksum) {
		return errors.New("RuntimeConfiguration checksum or behavior fingerprint is malformed")
	}
	if err := validateArtifactBinding("experiment", document.Experiment, ExperimentFormat); err != nil {
		return err
	}
	if err := validateAuthorityProfile(document.AuthorityProfile); err != nil {
		return err
	}
	if err := validatePhaseLimits(document.PhaseLimits); err != nil {
		return err
	}
	if err := validateObservationConfiguration(document.Observation); err != nil {
		return err
	}
	if err := validateParticipantBindings(document.ParticipantBindings); err != nil {
		return err
	}
	if document.KnownGaps == nil {
		return errors.New("RuntimeConfiguration known gaps must not be null")
	}
	if err := validateKnownGaps(document.KnownGaps); err != nil {
		return err
	}
	return validateProvenance(document.Provenance)
}

func validateArtifactBinding(label string, binding ArtifactBinding, format string) error {
	if binding.FormatVersion != format {
		return fmt.Errorf("%s binding format %q is invalid", label, binding.FormatVersion)
	}
	if !ValidDigest(binding.ArtifactChecksum) || !ValidDigest(binding.BehaviorFingerprint) ||
		!ValidDigest(binding.ProvenanceChecksum) {
		return fmt.Errorf("%s binding contains a malformed digest", label)
	}
	return nil
}

func validateAuthorityProfile(profile AuthorityProfile) error {
	if !validDefinitionID(profile.DefinitionID) || profile.Version.IsZero() ||
		!ValidDigest(profile.BehaviorFingerprint) {
		return errors.New("authority profile is malformed")
	}
	if profile.RequiredCapabilityDefinitionIDs == nil {
		return errors.New("authority profile required capability definition IDs must not be null")
	}
	return validateDefinitionIDSet("authority profile required capability definition ID",
		profile.RequiredCapabilityDefinitionIDs)
}

func validatePhaseLimits(limits []PhaseLimit) error {
	if len(limits) != len(executionPhases) {
		return fmt.Errorf("phase limits must contain exactly %d phases", len(executionPhases))
	}
	for index, phase := range executionPhases {
		limit := limits[index]
		if limit.Phase != phase {
			return fmt.Errorf("phase limit %d is %q; expected %q", index, limit.Phase, phase)
		}
		if limit.DurationMilliseconds.IsZero() || limit.MaxAttempts.IsZero() ||
			limit.MaxRecords.IsZero() || limit.MaxBytes.IsZero() {
			return fmt.Errorf("phase limit %q must use positive bounds", limit.Phase)
		}
	}
	return nil
}

func validateObservationConfiguration(observation ObservationConfiguration) error {
	for _, field := range []struct {
		label string
		value string
	}{
		{label: "observation profile definition ID", value: observation.ProfileDefinitionID},
		{label: "observation program definition ID", value: observation.ProgramDefinitionID},
		{label: "observation mapping definition ID", value: observation.MappingDefinitionID},
	} {
		if !validDefinitionID(field.value) {
			return fmt.Errorf("%s %q is invalid", field.label, field.value)
		}
	}
	for _, field := range []struct {
		label string
		value string
	}{
		{label: "observation profile behavior fingerprint", value: observation.ProfileBehaviorFingerprint},
		{label: "observation program behavior fingerprint", value: observation.ProgramBehaviorFingerprint},
		{label: "observation mapping behavior fingerprint", value: observation.MappingBehaviorFingerprint},
	} {
		if !ValidDigest(field.value) {
			return fmt.Errorf("%s %q is invalid", field.label, field.value)
		}
	}
	return nil
}

func validateParticipantBindings(bindings []ParticipantBinding) error {
	if len(bindings) == 0 {
		return errors.New("at least one participant binding is required")
	}
	if !slices.IsSortedFunc(bindings, func(left, right ParticipantBinding) int {
		return strings.Compare(left.ParticipantDefinitionID, right.ParticipantDefinitionID)
	}) {
		return errors.New("participant bindings are not in canonical order")
	}
	for index, binding := range bindings {
		if index > 0 && binding.ParticipantDefinitionID == bindings[index-1].ParticipantDefinitionID {
			return fmt.Errorf("duplicate participant binding %q", binding.ParticipantDefinitionID)
		}
		if !validDefinitionID(binding.ParticipantDefinitionID) ||
			!validDefinitionID(binding.ProtocolDefinitionID) ||
			binding.ProtocolVersion.IsZero() ||
			!validDefinitionID(binding.ProgramDefinitionID) ||
			!ValidDigest(binding.ProgramBehaviorFingerprint) {
			return fmt.Errorf("participant binding %q is malformed", binding.ParticipantDefinitionID)
		}
		if binding.CapabilityDefinitionIDs == nil {
			return fmt.Errorf("participant binding %q capability definition IDs must not be null",
				binding.ParticipantDefinitionID)
		}
		if err := validateDefinitionIDSet("participant capability definition ID",
			binding.CapabilityDefinitionIDs); err != nil {
			return err
		}
	}
	return nil
}

func ExperimentArtifactBinding(document Experiment) (ArtifactBinding, error) {
	provenanceChecksum, err := ExpectedProvenanceChecksum(document.Provenance)
	if err != nil {
		return ArtifactBinding{}, err
	}
	return ArtifactBinding{
		FormatVersion:       document.FormatVersion,
		ArtifactChecksum:    document.ArtifactChecksum,
		BehaviorFingerprint: document.QueryBehaviorFingerprint,
		ProvenanceChecksum:  provenanceChecksum,
	}, nil
}

func RuntimeConfigurationArtifactBinding(document RuntimeConfiguration) ArtifactBinding {
	return ArtifactBinding{
		FormatVersion:       document.FormatVersion,
		ArtifactChecksum:    document.ArtifactChecksum,
		BehaviorFingerprint: document.BehaviorFingerprint,
		ProvenanceChecksum:  document.ProvenanceChecksum,
	}
}

func ValidateRuntimeConfigurationExperimentClosure(document RuntimeConfiguration, experiment Experiment) error {
	expected, err := ExperimentArtifactBinding(experiment)
	if err != nil {
		return err
	}
	if document.Experiment != expected {
		return errors.New("RuntimeConfiguration experiment binding does not match ExperimentSpec")
	}
	capabilities := append([]string{}, document.AuthorityProfile.RequiredCapabilityDefinitionIDs...)
	for _, participant := range document.ParticipantBindings {
		capabilities = append(capabilities, participant.CapabilityDefinitionIDs...)
	}
	slices.Sort(capabilities)
	capabilities = slices.Compact(capabilities)
	if !slices.Equal(capabilities, experiment.Plan.CapabilityRequirementDefinitionIDs) {
		return errors.New("RuntimeConfiguration capabilities do not match ExperimentSpec requirements")
	}
	return nil
}

func CanonicalExperimentRunBytes(document ExperimentRun) ([]byte, error) {
	return encodeJSONLine(document)
}

func ExpectedExperimentRunChecksum(document ExperimentRun) (string, error) {
	document.ArtifactChecksum = ""
	encoded, err := encodeJSONLine(document)
	if err != nil {
		return "", err
	}
	return derive(experimentRunChecksumDomain, encoded), nil
}

func SealExperimentRun(document ExperimentRun) (ExperimentRun, error) {
	provenanceChecksum, err := ExpectedProvenanceChecksum(document.Provenance)
	if err != nil {
		return ExperimentRun{}, err
	}
	document.ProvenanceChecksum = provenanceChecksum
	artifactChecksum, err := ExpectedExperimentRunChecksum(document)
	if err != nil {
		return ExperimentRun{}, err
	}
	document.ArtifactChecksum = artifactChecksum
	return document, nil
}

func VerifyExperimentRunProvenanceChecksum(document ExperimentRun) error {
	expected, err := ExpectedProvenanceChecksum(document.Provenance)
	if err != nil {
		return err
	}
	if document.ProvenanceChecksum != expected {
		return fmt.Errorf("ExperimentRun provenance checksum mismatch: got %q, want %q",
			document.ProvenanceChecksum, expected)
	}
	return nil
}

func VerifyExperimentRunArtifactChecksum(document ExperimentRun) error {
	expected, err := ExpectedExperimentRunChecksum(document)
	if err != nil {
		return err
	}
	if document.ArtifactChecksum != expected {
		return fmt.Errorf("ExperimentRun artifact checksum mismatch: got %q, want %q",
			document.ArtifactChecksum, expected)
	}
	return nil
}

func ValidateExperimentRun(document ExperimentRun) error {
	if document.FormatVersion != ExperimentRunFormat {
		return fmt.Errorf("unsupported format %q", document.FormatVersion)
	}
	if !validDefinitionID(document.RunIdentity) || !ValidDigest(document.BehaviorFingerprint) ||
		!ValidDigest(document.ProvenanceChecksum) || !ValidDigest(document.ArtifactChecksum) ||
		document.Attempt.IsZero() {
		return errors.New("ExperimentRun identity, attempt, checksum, or behavior fingerprint is malformed")
	}
	if err := validateArtifactBinding("experiment", document.Experiment, ExperimentFormat); err != nil {
		return err
	}
	if err := validateArtifactBinding("runtime configuration", document.RuntimeConfiguration,
		RuntimeConfigurationFormat); err != nil {
		return err
	}
	if err := validatePhaseOutcomes(document.PhaseOutcomes); err != nil {
		return err
	}
	if err := validatePhaseProgression(document.PhaseOutcomes); err != nil {
		return err
	}
	if err := validateControlAttempts(document.ControlAttempts, document.Attempt); err != nil {
		return err
	}
	if err := validateSourceClosures(document.SourceClosures); err != nil {
		return err
	}
	if err := validateCleanupOutcome(document.Cleanup); err != nil {
		return err
	}
	if err := validatePhaseLimits(document.Limits); err != nil {
		return err
	}
	if document.KnownGaps == nil {
		return errors.New("ExperimentRun known gaps must not be null")
	}
	if err := validateKnownGaps(document.KnownGaps); err != nil {
		return err
	}
	if err := validateProvenance(document.Provenance); err != nil {
		return err
	}
	expected := expectedOperationalStatus(document)
	if document.OperationalStatus != expected {
		return fmt.Errorf("operational status %q is inconsistent with Run outcomes; expected %q",
			document.OperationalStatus, expected)
	}
	return nil
}

func validatePhaseOutcomes(outcomes []PhaseOutcome) error {
	if len(outcomes) != len(executionPhases) {
		return fmt.Errorf("phase outcomes must contain exactly %d phases", len(executionPhases))
	}
	for index, phase := range executionPhases {
		outcome := outcomes[index]
		if outcome.Phase != phase {
			return fmt.Errorf("phase outcome %d is %q; expected %q", index, outcome.Phase, phase)
		}
		switch outcome.Status {
		case "not-started":
			if outcome.StartedAtUnixMillis != nil || outcome.FinishedAtUnixMillis != nil || outcome.Code != nil {
				return fmt.Errorf("not-started phase %q must not have timestamps or code", outcome.Phase)
			}
		case "succeeded":
			if err := validateTerminalPhaseOutcome(outcome, false); err != nil {
				return err
			}
		case "failed", "timed-out", "canceled":
			if err := validateTerminalPhaseOutcome(outcome, true); err != nil {
				return err
			}
		default:
			return fmt.Errorf("phase status %q is invalid", outcome.Status)
		}
	}
	return nil
}

func validateTerminalPhaseOutcome(outcome PhaseOutcome, requiresCode bool) error {
	if outcome.StartedAtUnixMillis == nil || outcome.FinishedAtUnixMillis == nil ||
		compareNatural(*outcome.StartedAtUnixMillis, *outcome.FinishedAtUnixMillis) > 0 {
		return fmt.Errorf("terminal phase %q has invalid timestamps", outcome.Phase)
	}
	if requiresCode != (outcome.Code != nil) {
		return fmt.Errorf("terminal phase %q has an invalid code for status %q", outcome.Phase, outcome.Status)
	}
	if outcome.Code != nil && !validDefinitionID(*outcome.Code) {
		return fmt.Errorf("terminal phase %q code %q is invalid", outcome.Phase, *outcome.Code)
	}
	return nil
}

func validatePhaseProgression(outcomes []PhaseOutcome) error {
	preparation := outcomes[0].Status
	realization := outcomes[1].Status
	observation := outcomes[2].Status
	cleanup := outcomes[4].Status
	if preparation == "not-started" {
		return errors.New("preparation must start before an ExperimentRun can exist")
	}
	if preparation != "succeeded" && realization != "not-started" {
		return errors.New("realization cannot start before preparation succeeds")
	}
	if (realization == "not-started") != (observation == "not-started") {
		return errors.New("observation must start exactly when realization starts")
	}
	if cleanup == "not-started" {
		return errors.New("cleanup must start exactly once after preparation begins")
	}
	return nil
}

func validateControlAttempts(attempts []ControlAttempt, runAttempt Natural) error {
	if attempts == nil {
		return errors.New("control attempts must not be null")
	}
	if !slices.IsSortedFunc(attempts, compareControlAttempt) {
		return errors.New("control attempts are not in canonical order")
	}
	receipts := make(map[string]struct{}, len(attempts))
	for index, attempt := range attempts {
		if index > 0 && compareControlAttempt(attempts[index-1], attempt) == 0 {
			return fmt.Errorf("duplicate control attempt for occurrence %q", attempt.OccurrenceDefinitionID)
		}
		if !validDefinitionID(attempt.OccurrenceDefinitionID) || !validDefinitionID(attempt.ActionDefinitionID) ||
			attempt.Attempt.IsZero() || attempt.Attempt != runAttempt {
			return fmt.Errorf("control attempt for occurrence %q is malformed", attempt.OccurrenceDefinitionID)
		}
		switch attempt.Status {
		case "not-attempted":
			if attempt.ReceiptFactDefinitionID != nil || attempt.Code != nil {
				return fmt.Errorf("not-attempted control %q must not have receipt or code",
					attempt.OccurrenceDefinitionID)
			}
		case "accepted":
			if err := validateAttemptedControl(attempt, false, receipts); err != nil {
				return err
			}
		case "rejected", "unsupported", "failed", "canceled":
			if err := validateAttemptedControl(attempt, true, receipts); err != nil {
				return err
			}
		default:
			return fmt.Errorf("control status %q is invalid", attempt.Status)
		}
	}
	return nil
}

func validateAttemptedControl(attempt ControlAttempt, requiresCode bool, receipts map[string]struct{}) error {
	if attempt.ReceiptFactDefinitionID == nil || !validDefinitionID(*attempt.ReceiptFactDefinitionID) {
		return fmt.Errorf("attempted control %q must have one valid receipt fact definition ID",
			attempt.OccurrenceDefinitionID)
	}
	if _, duplicate := receipts[*attempt.ReceiptFactDefinitionID]; duplicate {
		return fmt.Errorf("duplicate receipt fact definition ID %q", *attempt.ReceiptFactDefinitionID)
	}
	receipts[*attempt.ReceiptFactDefinitionID] = struct{}{}
	if requiresCode != (attempt.Code != nil) {
		return fmt.Errorf("attempted control %q has an invalid code for status %q",
			attempt.OccurrenceDefinitionID, attempt.Status)
	}
	if attempt.Code != nil && !validDefinitionID(*attempt.Code) {
		return fmt.Errorf("control code %q is invalid", *attempt.Code)
	}
	return nil
}

func compareControlAttempt(left, right ControlAttempt) int {
	if comparison := strings.Compare(left.OccurrenceDefinitionID, right.OccurrenceDefinitionID); comparison != 0 {
		return comparison
	}
	return compareNatural(left.Attempt, right.Attempt)
}

func validateSourceClosures(closures []SourceClosure) error {
	if len(closures) == 0 {
		return errors.New("at least one source closure is required")
	}
	if !slices.IsSortedFunc(closures, func(left, right SourceClosure) int {
		return strings.Compare(left.SourceDefinitionID, right.SourceDefinitionID)
	}) {
		return errors.New("source closures are not in canonical order")
	}
	for index, closure := range closures {
		if index > 0 && closure.SourceDefinitionID == closures[index-1].SourceDefinitionID {
			return fmt.Errorf("duplicate source closure %q", closure.SourceDefinitionID)
		}
		if !validDefinitionID(closure.SourceDefinitionID) {
			return fmt.Errorf("source closure definition ID %q is invalid", closure.SourceDefinitionID)
		}
		switch closure.Status {
		case "closed", "partial", "failed":
		default:
			return fmt.Errorf("source closure status %q is invalid", closure.Status)
		}
	}
	return nil
}

func validateCleanupOutcome(cleanup CleanupOutcome) error {
	switch cleanup.Status {
	case "complete":
		if !cleanup.OpenHandleCount.IsZero() || cleanup.Code != nil {
			return errors.New("complete cleanup must have zero open handles and no code")
		}
	case "incomplete", "failed":
		if cleanup.Code == nil || !validDefinitionID(*cleanup.Code) {
			return fmt.Errorf("cleanup status %q requires one valid code", cleanup.Status)
		}
	default:
		return fmt.Errorf("cleanup status %q is invalid", cleanup.Status)
	}
	return nil
}

// expectedOperationalStatus validates the declared summary; it does not construct or normalize a Run.
func expectedOperationalStatus(document ExperimentRun) string {
	for _, phase := range document.PhaseOutcomes {
		if phase.Status == "failed" {
			return "failed"
		}
	}
	for _, control := range document.ControlAttempts {
		if control.Status == "rejected" || control.Status == "unsupported" || control.Status == "failed" {
			return "failed"
		}
	}
	for _, source := range document.SourceClosures {
		if source.Status == "failed" {
			return "failed"
		}
	}
	if document.Cleanup.Status == "failed" {
		return "failed"
	}
	for _, phase := range document.PhaseOutcomes {
		if phase.Status != "succeeded" {
			return "incomplete"
		}
	}
	for _, control := range document.ControlAttempts {
		if control.Status != "accepted" {
			return "incomplete"
		}
	}
	for _, source := range document.SourceClosures {
		if source.Status != "closed" {
			return "incomplete"
		}
	}
	if document.Cleanup.Status != "complete" {
		return "incomplete"
	}
	return "succeeded"
}

func ValidateExperimentRunClosure(
	document ExperimentRun,
	experiment Experiment,
	runtimeConfiguration RuntimeConfiguration,
) error {
	if err := ValidateRuntimeConfigurationExperimentClosure(runtimeConfiguration, experiment); err != nil {
		return err
	}
	experimentBinding, err := ExperimentArtifactBinding(experiment)
	if err != nil {
		return err
	}
	if document.Experiment != experimentBinding {
		return errors.New("ExperimentRun experiment binding does not match ExperimentSpec")
	}
	if document.RuntimeConfiguration != RuntimeConfigurationArtifactBinding(runtimeConfiguration) {
		return errors.New("ExperimentRun runtime configuration binding does not match RuntimeConfiguration")
	}
	if !slices.Equal(document.Limits, runtimeConfiguration.PhaseLimits) {
		return errors.New("ExperimentRun limits do not match RuntimeConfiguration phase limits")
	}
	if len(document.ControlAttempts) != len(experiment.Plan.LinearExtension) {
		return errors.New("ExperimentRun control attempts do not close over planned occurrences")
	}
	planned := make(map[string]Occurrence, len(experiment.Plan.LinearExtension))
	for _, occurrence := range experiment.Plan.LinearExtension {
		planned[occurrence.DefinitionID] = occurrence
	}
	for _, attempt := range document.ControlAttempts {
		occurrence, ok := planned[attempt.OccurrenceDefinitionID]
		if !ok || occurrence.ActionDefinitionID != attempt.ActionDefinitionID {
			return fmt.Errorf("control attempt %q does not match a planned occurrence",
				attempt.OccurrenceDefinitionID)
		}
		delete(planned, attempt.OccurrenceDefinitionID)
	}
	if len(planned) != 0 {
		return errors.New("ExperimentRun is missing a planned control attempt")
	}
	return nil
}

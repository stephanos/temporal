package qualification

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"slices"

	"go.temporal.io/server/tests/umpire3/canary"
	umpire3runtime "go.temporal.io/server/tests/umpire3/execution"
	"go.temporal.io/server/tests/umpire3/protocol"
	releaseassurance "go.temporal.io/server/tests/umpire3/release"
)

const FormatVersion = protocol.QualificationReceiptFormatVersion

type Receipt = protocol.QualificationReceipt

type Request struct {
	ReleaseBytes    []byte
	ExperimentBytes []byte
	ResultBytes     []byte
	Profile         string
	SigningKey      ed25519.PrivateKey
}

type PromotionRequest struct {
	ReleaseBytes []byte
	Receipts     [][]byte
}

func Qualify(request Request) (Receipt, error) {
	if len(request.ReleaseBytes) == 0 || len(request.ExperimentBytes) == 0 ||
		len(request.ResultBytes) == 0 || request.Profile == "" || len(request.SigningKey) == 0 {
		return Receipt{}, errors.New("release, experiment, result, profile, and signing key are required")
	}
	release, err := protocol.DecodeReleaseManifest(request.ReleaseBytes)
	if err != nil {
		return Receipt{}, fmt.Errorf("decode release: %w", err)
	}
	if err := releaseassurance.ValidateAgainstCurrent(release); err != nil {
		return Receipt{}, fmt.Errorf("validate release against current artifacts: %w", err)
	}
	gate, required := requiredQualification(release, request.Profile)
	if !required {
		return Receipt{}, fmt.Errorf("profile %q is not an external gate for release %q", request.Profile, release.Release)
	}
	if gate.Authority == nil {
		return Receipt{}, fmt.Errorf("profile %q qualification authority is not provisioned", request.Profile)
	}
	experiment, err := protocol.DecodeExperiment(bytes.NewReader(request.ExperimentBytes), protocol.DefaultDecodeLimit)
	if err != nil {
		return Receipt{}, fmt.Errorf("decode experiment: %w", err)
	}
	releasedExperiment, released := release.Experiments[experiment.ExperimentID]
	experimentDigest, err := experiment.Digest()
	if err != nil {
		return Receipt{}, err
	}
	if !released || releasedExperiment.SemanticHash != experiment.Model.SemanticHash ||
		releasedExperiment.Digest != experimentDigest {
		return Receipt{}, errors.New("experiment is not bound to the candidate release")
	}
	result, canaryEnvelope, err := decodeResultEnvelope(request.ResultBytes)
	if err != nil {
		return Receipt{}, err
	}
	if request.Profile == "production-canary" && !canaryEnvelope {
		return Receipt{}, errors.New("production canary qualification requires a canary envelope")
	}
	if request.Profile != "production-canary" && canaryEnvelope {
		return Receipt{}, fmt.Errorf("profile %q cannot use a production canary envelope", request.Profile)
	}
	if err := validateResult(request.Profile, experiment, result); err != nil {
		return Receipt{}, err
	}
	evidenceBytes, err := result.Evidence.CanonicalJSON()
	if err != nil {
		return Receipt{}, fmt.Errorf("encode evidence: %w", err)
	}
	evidenceHash := sha256.Sum256(evidenceBytes)
	releaseBytes, err := release.CanonicalJSON()
	if err != nil {
		return Receipt{}, fmt.Errorf("encode canonical release: %w", err)
	}
	releaseHash := sha256.Sum256(releaseBytes)
	resultHash := sha256.Sum256(request.ResultBytes)
	receipt := Receipt{
		FormatVersion: FormatVersion, Release: release.Release,
		ReleaseDigest: "sha256:" + hex.EncodeToString(releaseHash[:]), Profile: request.Profile,
		ExperimentID: experiment.ExperimentID, ExperimentDigest: result.ExperimentDigest,
		ResultDigest: "sha256:" + hex.EncodeToString(resultHash[:]),
		BuildID:      result.Environment.BuildID, ConfigurationIdentity: result.Environment.ConfigurationIdentity,
		EvidenceDigest: "sha256:" + hex.EncodeToString(evidenceHash[:]),
		Authority:      *gate.Authority,
	}
	return protocol.SignQualificationReceipt(receipt, request.SigningKey)
}

func DecodeReceipt(encoded []byte) (Receipt, error) {
	return protocol.DecodeQualificationReceipt(encoded)
}

func ParseSigningKey(encoded []byte) (ed25519.PrivateKey, error) {
	block, trailing := pem.Decode(encoded)
	if block == nil || block.Type != "PRIVATE KEY" || len(bytes.TrimSpace(trailing)) != 0 {
		return nil, errors.New("qualification signing key must be one PKCS#8 PRIVATE KEY PEM block")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse qualification signing key: %w", err)
	}
	privateKey, ok := parsed.(ed25519.PrivateKey)
	if !ok || len(privateKey) != ed25519.PrivateKeySize {
		return nil, errors.New("qualification signing key must be Ed25519")
	}
	return privateKey, nil
}

func Promote(request PromotionRequest) (protocol.ReleaseManifest, error) {
	if len(request.ReleaseBytes) == 0 || len(request.Receipts) == 0 {
		return protocol.ReleaseManifest{}, errors.New("candidate release and qualification receipts are required")
	}
	release, err := protocol.DecodeReleaseManifest(request.ReleaseBytes)
	if err != nil {
		return protocol.ReleaseManifest{}, fmt.Errorf("decode candidate release: %w", err)
	}
	if release.Status != "candidate" {
		return protocol.ReleaseManifest{}, errors.New("only a candidate release can be promoted")
	}
	if err := releaseassurance.ValidateAgainstCurrent(release); err != nil {
		return protocol.ReleaseManifest{}, fmt.Errorf("validate candidate release against current artifacts: %w", err)
	}
	canonicalRelease, err := release.CanonicalJSON()
	if err != nil {
		return protocol.ReleaseManifest{}, err
	}
	candidateDigest := digestBytes(canonicalRelease)
	required := make(map[string]protocol.ExternalQualification, len(release.ExternalQualifications))
	for _, gate := range release.ExternalQualifications {
		required[gate.Profile] = gate
	}
	seen := make(map[string]struct{}, len(request.Receipts))
	qualifications := make([]protocol.ReleaseQualification, 0, len(request.Receipts))
	experimentID := ""
	experimentDigest := ""
	for _, encoded := range request.Receipts {
		receipt, decodeErr := DecodeReceipt(encoded)
		if decodeErr != nil {
			return protocol.ReleaseManifest{}, decodeErr
		}
		if receipt.Release != release.Release || receipt.ReleaseDigest != candidateDigest {
			return protocol.ReleaseManifest{}, errors.New("qualification receipt is not bound to the candidate release")
		}
		gate, requiredProfile := required[receipt.Profile]
		if !requiredProfile {
			return protocol.ReleaseManifest{}, fmt.Errorf("profile %q is not a required external qualification", receipt.Profile)
		}
		if gate.Authority == nil {
			return protocol.ReleaseManifest{}, fmt.Errorf("profile %q qualification authority is not provisioned", receipt.Profile)
		}
		if err := receipt.Verify(*gate.Authority); err != nil {
			return protocol.ReleaseManifest{}, fmt.Errorf("verify profile %q qualification: %w", receipt.Profile, err)
		}
		if _, duplicate := seen[receipt.Profile]; duplicate {
			return protocol.ReleaseManifest{}, fmt.Errorf("duplicate external qualification for profile %q", receipt.Profile)
		}
		seen[receipt.Profile] = struct{}{}
		if _, released := release.Experiments[receipt.ExperimentID]; !released {
			return protocol.ReleaseManifest{}, fmt.Errorf("qualification receipt references unreleased experiment %q", receipt.ExperimentID)
		}
		if experimentID == "" {
			experimentID = receipt.ExperimentID
			experimentDigest = receipt.ExperimentDigest
		} else if receipt.ExperimentID != experimentID || receipt.ExperimentDigest != experimentDigest {
			return protocol.ReleaseManifest{}, errors.New("external qualifications must use the same experiment digest")
		}
		receiptDigest, digestErr := receipt.Digest()
		if digestErr != nil {
			return protocol.ReleaseManifest{}, digestErr
		}
		qualifications = append(qualifications, protocol.ReleaseQualification{
			QualificationReceipt: receipt,
			ReceiptDigest:        receiptDigest,
		})
	}
	for profile := range required {
		if _, present := seen[profile]; !present {
			return protocol.ReleaseManifest{}, fmt.Errorf("missing external qualification for profile %q", profile)
		}
	}
	if len(seen) != len(required) {
		return protocol.ReleaseManifest{}, errors.New("external qualification profile count does not match the candidate gates")
	}
	slices.SortFunc(qualifications, func(left, right protocol.ReleaseQualification) int {
		return stringCompare(left.Profile, right.Profile)
	})
	release.Status = "qualified"
	release.ExternalQualifications = nil
	release.Qualifications = qualifications
	release, err = releaseassurance.RebindCurrentAssurance(release)
	if err != nil {
		return protocol.ReleaseManifest{}, fmt.Errorf("bind promoted release assurance: %w", err)
	}
	if err := releaseassurance.ValidateAgainstCurrent(release); err != nil {
		return protocol.ReleaseManifest{}, fmt.Errorf("validate promoted release: %w", err)
	}
	return release, nil
}

func digestBytes(value []byte) string {
	digest := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func stringCompare(left, right string) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func requiredQualification(release protocol.ReleaseManifest, name string) (protocol.ExternalQualification, bool) {
	index := slices.IndexFunc(release.ExternalQualifications, func(value protocol.ExternalQualification) bool {
		return value.Profile == name && value.Status == "required"
	})
	if index < 0 {
		return protocol.ExternalQualification{}, false
	}
	return release.ExternalQualifications[index], true
}

func DecodeResult(encoded []byte) (umpire3runtime.Result, error) {
	result, _, err := decodeResultEnvelope(encoded)
	return result, err
}

func decodeResultEnvelope(encoded []byte) (umpire3runtime.Result, bool, error) {
	var envelope struct {
		Runtime  json.RawMessage `json:"runtime"`
		Complete bool            `json:"complete"`
	}
	if err := json.Unmarshal(encoded, &envelope); err != nil {
		return umpire3runtime.Result{}, false, fmt.Errorf("decode result envelope: %w", err)
	}
	if len(envelope.Runtime) != 0 {
		var canaryResult canary.Result
		decoder := json.NewDecoder(bytes.NewReader(encoded))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&canaryResult); err != nil {
			return umpire3runtime.Result{}, false, fmt.Errorf("decode canary result: %w", err)
		}
		if !canaryResult.Complete {
			return umpire3runtime.Result{}, false, errors.New("canary result is incomplete")
		}
		if err := canaryResult.ValidateQualification(); err != nil {
			return umpire3runtime.Result{}, false, err
		}
		if err := validateDecodedResult(canaryResult.Runtime); err != nil {
			return umpire3runtime.Result{}, false, err
		}
		return canaryResult.Runtime, true, nil
	}
	var result umpire3runtime.Result
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return umpire3runtime.Result{}, false, fmt.Errorf("decode runtime result: %w", err)
	}
	if err := validateDecodedResult(result); err != nil {
		return umpire3runtime.Result{}, false, err
	}
	return result, false, nil
}

func validateDecodedResult(result umpire3runtime.Result) error {
	if result.FormatVersion != umpire3runtime.ResultFormatVersion {
		return fmt.Errorf("unsupported runtime result format %q", result.FormatVersion)
	}
	if err := result.ValidateAssurance(); err != nil {
		return fmt.Errorf("validate result assurance: %w", err)
	}
	if err := result.ValidateEvidenceDigest(); err != nil {
		return fmt.Errorf("validate result evidence digest: %w", err)
	}
	return nil
}

func validateResult(profile string, experiment protocol.Experiment, result umpire3runtime.Result) error {
	digest, err := experiment.Digest()
	if err != nil {
		return err
	}
	if result.FormatVersion != umpire3runtime.ResultFormatVersion || result.ExperimentDigest != digest {
		return errors.New("result is not bound to the executed experiment")
	}
	if err := result.ValidateAssurance(); err != nil {
		return fmt.Errorf("validate result assurance: %w", err)
	}
	if err := result.Environment.Validate(); err != nil {
		return fmt.Errorf("validate environment identity: %w", err)
	}
	if result.Environment.Name != profile || result.Environment.BuildID == "" ||
		result.Environment.ConfigurationIdentity == "" {
		return errors.New("result lacks deployment profile attestation")
	}
	if profile == "production-canary" && !result.Environment.HardExecutionBudget {
		return errors.New("production canary qualification requires a hard execution boundary")
	}
	if profile != "production-canary" && result.Environment.HardExecutionBudget {
		return fmt.Errorf("profile %q cannot claim the production hard execution boundary", profile)
	}
	expectedEvidenceProfile := umpire3runtime.EvidenceProfilePublicGRPCHistory
	switch profile {
	case "local-in-process":
		expectedEvidenceProfile = umpire3runtime.EvidenceProfileInProcessHooks
	case "grpc-only-black-box":
		expectedEvidenceProfile = umpire3runtime.EvidenceProfilePublicGRPC
	}
	if result.Environment.EvidenceProfile != expectedEvidenceProfile {
		return fmt.Errorf("profile %q requires %q evidence", profile, expectedEvidenceProfile)
	}
	if result.Claim.Kind != umpire3runtime.ClaimConforming || result.Claim.Property != experiment.Property.Identifier {
		return errors.New("qualification requires a conforming claim for the experiment property")
	}
	if len(result.Omissions) != 0 || !result.Cleanup.Complete {
		return errors.New("qualification requires complete evidence and cleanup")
	}
	if len(result.Faults) != len(experiment.Faults) {
		return errors.New("qualification requires realization evidence for every declared fault")
	}
	for index, fault := range result.Faults {
		if fault.Identifier != experiment.Faults[index].Identifier || !fault.Realized ||
			!fault.Released || !fault.CleanupComplete || fault.Error != "" ||
			fault.SourceIdentity == "" || fault.Reference == "" || fault.EntityIdentity == "" {
			return fmt.Errorf("qualification fault %q lacks complete realization and cleanup evidence", fault.Identifier)
		}
	}
	if len(result.Evidence.Facts) == 0 || len(result.Evidence.Claims) == 0 {
		return errors.New("qualification result has no supporting evidence claim")
	}
	if err := result.Evidence.Validate(); err != nil {
		return fmt.Errorf("validate qualification evidence: %w", err)
	}
	return nil
}

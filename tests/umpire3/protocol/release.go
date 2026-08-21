package protocol

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

const ReleaseFormatVersion = "umpire3/release/v6"
const QualificationReceiptFormatVersion = "umpire3/qualification-receipt/v3"

type ReleaseMigration struct {
	FormatVersion           string `json:"formatVersion"`
	LedgerHash              string `json:"ledgerHash"`
	BehaviorCount           int    `json:"behaviorCount"`
	ExactCount              int    `json:"exactCount"`
	SemanticEquivalentCount int    `json:"semanticEquivalentCount"`
	PartialCount            int    `json:"partialCount"`
	InventoryOnlyCount      int    `json:"inventoryOnlyCount"`
}

type ExternalQualification struct {
	Profile   string                  `json:"profile"`
	Command   string                  `json:"command"`
	Status    string                  `json:"status"`
	Authority *QualificationAuthority `json:"authority,omitempty"`
}

type QualificationAuthority struct {
	Identity  string `json:"identity"`
	PublicKey string `json:"publicKey"`
}

type QualificationReceipt struct {
	FormatVersion         string                 `json:"formatVersion"`
	Release               string                 `json:"release"`
	ReleaseDigest         string                 `json:"releaseDigest"`
	Profile               string                 `json:"profile"`
	ExperimentID          string                 `json:"experimentID"`
	ExperimentDigest      string                 `json:"experimentDigest"`
	ResultDigest          string                 `json:"resultDigest"`
	BuildID               string                 `json:"buildID"`
	ConfigurationIdentity string                 `json:"configurationIdentity"`
	EvidenceDigest        string                 `json:"evidenceDigest"`
	Authority             QualificationAuthority `json:"authority"`
	Signature             string                 `json:"signature"`
}

type ReleaseQualification struct {
	QualificationReceipt
	ReceiptDigest string `json:"receiptDigest"`
}

type ReleaseProofManifest struct {
	Identifier string `json:"identifier"`
	Digest     string `json:"digest"`
}

type ReleaseExperiment struct {
	SemanticHash string `json:"semanticHash"`
	Digest       string `json:"digest"`
}

type ReleaseManifest struct {
	Release                 string                       `json:"release"`
	Status                  string                       `json:"status"`
	FormatVersion           string                       `json:"formatVersion"`
	ExperimentFormatVersion string                       `json:"experimentFormatVersion"`
	LeanVersion             string                       `json:"leanVersion"`
	CatalogHash             string                       `json:"catalogHash"`
	DescriptorHash          string                       `json:"descriptorHash"`
	MonitorSemanticHash     string                       `json:"monitorSemanticHash"`
	CompositionSemanticHash string                       `json:"compositionSemanticHash"`
	ParitySemanticHash      string                       `json:"paritySemanticHash"`
	CheckerCoverageHash     string                       `json:"checkerCoverageHash"`
	Experiments             map[string]ReleaseExperiment `json:"experiments"`
	ProofManifests          []ReleaseProofManifest       `json:"proofManifests"`
	Profiles                []string                     `json:"profiles"`
	Migration               ReleaseMigration             `json:"migration"`
	Assurance               ReleaseAssurance             `json:"assurance"`
	Documents               map[string]string            `json:"documents"`
	ExternalQualifications  []ExternalQualification      `json:"externalQualifications"`
	Qualifications          []ReleaseQualification       `json:"qualifications,omitempty"`
}

func DecodeReleaseManifest(encoded []byte) (ReleaseManifest, error) {
	var manifest ReleaseManifest
	if err := decodeStrictJSON(bytes.NewReader(encoded), DefaultDecodeLimit, "release manifest", &manifest); err != nil {
		return ReleaseManifest{}, err
	}
	if err := manifest.Validate(); err != nil {
		return ReleaseManifest{}, err
	}
	return manifest, nil
}

func BindReleaseArtifactBindings(manifest ReleaseManifest, experiments []Experiment) (ReleaseManifest, error) {
	if err := manifest.Validate(); err != nil {
		return ReleaseManifest{}, err
	}
	if len(experiments) == 0 {
		return ReleaseManifest{}, errors.New("release binding requires experiments")
	}
	catalog, err := DefaultCatalog()
	if err != nil {
		return ReleaseManifest{}, err
	}
	manifest.CatalogHash, err = catalog.Digest()
	if err != nil {
		return ReleaseManifest{}, err
	}
	manifest.LeanVersion = catalog.LeanVersion
	protobuf, err := DefaultProtobufInventory()
	if err != nil {
		return ReleaseManifest{}, err
	}
	manifest.DescriptorHash = protobuf.DescriptorDigest
	monitors, err := DefaultMonitorCatalog()
	if err != nil {
		return ReleaseManifest{}, err
	}
	manifest.MonitorSemanticHash = monitors.SemanticHash
	composition, err := DefaultComposition()
	if err != nil {
		return ReleaseManifest{}, err
	}
	manifest.CompositionSemanticHash = composition.SemanticHash
	parity, err := DefaultParityLedger()
	if err != nil {
		return ReleaseManifest{}, err
	}
	manifest.ParitySemanticHash = parity.SemanticHash
	checkerCoverage, err := DefaultCheckerCoverage()
	if err != nil {
		return ReleaseManifest{}, err
	}
	checkerCoverageJSON, err := checkerCoverage.CanonicalJSON()
	if err != nil {
		return ReleaseManifest{}, err
	}
	manifest.CheckerCoverageHash = digestBytes(checkerCoverageJSON)

	manifest.Experiments = make(map[string]ReleaseExperiment, len(experiments))
	for _, experiment := range experiments {
		if err := experiment.Validate(); err != nil {
			return ReleaseManifest{}, fmt.Errorf("bind experiment %q: %w", experiment.ExperimentID, err)
		}
		if experiment.Model.LeanVersion != catalog.LeanVersion {
			return ReleaseManifest{}, fmt.Errorf("bind experiment %q: Lean version %q does not match catalog %q",
				experiment.ExperimentID, experiment.Model.LeanVersion, catalog.LeanVersion)
		}
		if _, duplicate := manifest.Experiments[experiment.ExperimentID]; duplicate {
			return ReleaseManifest{}, fmt.Errorf("bind duplicate experiment %q", experiment.ExperimentID)
		}
		digest, digestErr := experiment.Digest()
		if digestErr != nil {
			return ReleaseManifest{}, fmt.Errorf("digest experiment %q: %w", experiment.ExperimentID, digestErr)
		}
		manifest.Experiments[experiment.ExperimentID] = ReleaseExperiment{
			SemanticHash: experiment.Model.SemanticHash,
			Digest:       digest,
		}
	}

	proofManifests, err := DefaultProofManifests()
	if err != nil {
		return ReleaseManifest{}, err
	}
	manifest.ProofManifests = make([]ReleaseProofManifest, len(proofManifests))
	for index, proofManifest := range proofManifests {
		if proofManifest.LeanVersion != catalog.LeanVersion {
			return ReleaseManifest{}, fmt.Errorf("proof manifest %q Lean version %q does not match catalog %q",
				proofManifest.Identifier, proofManifest.LeanVersion, catalog.LeanVersion)
		}
		digest, digestErr := proofManifest.Digest()
		if digestErr != nil {
			return ReleaseManifest{}, digestErr
		}
		manifest.ProofManifests[index] = ReleaseProofManifest{
			Identifier: proofManifest.Identifier,
			Digest:     digest,
		}
	}
	slices.SortFunc(manifest.ProofManifests, func(left, right ReleaseProofManifest) int {
		return stringCompare(left.Identifier, right.Identifier)
	})
	if err := manifest.Validate(); err != nil {
		return ReleaseManifest{}, err
	}
	return manifest, nil
}

func (m ReleaseManifest) CanonicalJSON() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	encoded, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode release manifest: %w", err)
	}
	return append(encoded, '\n'), nil
}

func DecodeQualificationReceipt(encoded []byte) (QualificationReceipt, error) {
	var receipt QualificationReceipt
	if err := decodeStrictJSON(bytes.NewReader(encoded), DefaultDecodeLimit, "qualification receipt", &receipt); err != nil {
		return QualificationReceipt{}, err
	}
	if err := receipt.Verify(receipt.Authority); err != nil {
		return QualificationReceipt{}, err
	}
	return receipt, nil
}

func (r QualificationReceipt) Validate() error {
	if r.FormatVersion != QualificationReceiptFormatVersion || r.Release == "" || r.Profile == "" ||
		r.ExperimentID == "" || r.BuildID == "" ||
		!validHash(r.ReleaseDigest) || !validHash(r.ExperimentDigest) ||
		!validHash(r.ResultDigest) || !validHash(r.EvidenceDigest) {
		return errors.New("qualification receipt identity and digests are incomplete")
	}
	if !validHash(r.ConfigurationIdentity) {
		return errors.New("qualification configuration identity must be a SHA-256 digest")
	}
	if err := r.Authority.Validate(); err != nil {
		return err
	}
	if _, err := decodeQualificationSignature(r.Signature); err != nil {
		return err
	}
	return nil
}

func (r QualificationReceipt) CanonicalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	encoded, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode qualification receipt: %w", err)
	}
	return append(encoded, '\n'), nil
}

func (r QualificationReceipt) Digest() (string, error) {
	encoded, err := r.CanonicalJSON()
	if err != nil {
		return "", err
	}
	return digestBytes(encoded), nil
}

func (r QualificationReceipt) Verify(authority QualificationAuthority) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if r.Authority != authority {
		return errors.New("qualification receipt authority does not match the release gate")
	}
	publicKey, err := authority.publicKey()
	if err != nil {
		return err
	}
	signature, err := decodeQualificationSignature(r.Signature)
	if err != nil {
		return err
	}
	payload, err := r.signingPayload()
	if err != nil {
		return err
	}
	if !ed25519.Verify(publicKey, payload, signature) {
		return errors.New("qualification receipt signature is invalid")
	}
	return nil
}

func SignQualificationReceipt(receipt QualificationReceipt, privateKey ed25519.PrivateKey) (QualificationReceipt, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return QualificationReceipt{}, errors.New("Ed25519 qualification signing key is required")
	}
	if err := receipt.validateUnsigned(); err != nil {
		return QualificationReceipt{}, err
	}
	publicKey := privateKey.Public().(ed25519.PublicKey)
	expectedPublicKey, err := receipt.Authority.publicKey()
	if err != nil {
		return QualificationReceipt{}, err
	}
	if !bytes.Equal(publicKey, expectedPublicKey) {
		return QualificationReceipt{}, errors.New("qualification signing key does not match the release authority")
	}
	payload, err := receipt.signingPayload()
	if err != nil {
		return QualificationReceipt{}, err
	}
	receipt.Signature = base64.RawStdEncoding.EncodeToString(ed25519.Sign(privateKey, payload))
	if err := receipt.Verify(receipt.Authority); err != nil {
		return QualificationReceipt{}, err
	}
	return receipt, nil
}

func (a QualificationAuthority) Validate() error {
	if a.Identity == "" {
		return errors.New("qualification authority identity is required")
	}
	_, err := a.publicKey()
	return err
}

func NewQualificationAuthority(identity string, publicKey ed25519.PublicKey) (QualificationAuthority, error) {
	authority := QualificationAuthority{
		Identity:  identity,
		PublicKey: base64.RawStdEncoding.EncodeToString(publicKey),
	}
	if err := authority.Validate(); err != nil {
		return QualificationAuthority{}, err
	}
	return authority, nil
}

func (r QualificationReceipt) validateUnsigned() error {
	unsigned := r
	unsigned.Signature = base64.RawStdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
	return unsigned.Validate()
}

func (r QualificationReceipt) signingPayload() ([]byte, error) {
	payload := struct {
		FormatVersion         string                 `json:"formatVersion"`
		Release               string                 `json:"release"`
		ReleaseDigest         string                 `json:"releaseDigest"`
		Profile               string                 `json:"profile"`
		ExperimentID          string                 `json:"experimentID"`
		ExperimentDigest      string                 `json:"experimentDigest"`
		ResultDigest          string                 `json:"resultDigest"`
		BuildID               string                 `json:"buildID"`
		ConfigurationIdentity string                 `json:"configurationIdentity"`
		EvidenceDigest        string                 `json:"evidenceDigest"`
		Authority             QualificationAuthority `json:"authority"`
	}{
		FormatVersion: r.FormatVersion, Release: r.Release, ReleaseDigest: r.ReleaseDigest,
		Profile: r.Profile, ExperimentID: r.ExperimentID, ExperimentDigest: r.ExperimentDigest,
		ResultDigest: r.ResultDigest, BuildID: r.BuildID,
		ConfigurationIdentity: r.ConfigurationIdentity, EvidenceDigest: r.EvidenceDigest,
		Authority: r.Authority,
	}
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode qualification signature payload: %w", err)
	}
	return append(encoded, '\n'), nil
}

func (a QualificationAuthority) publicKey() (ed25519.PublicKey, error) {
	encoded, err := base64.RawStdEncoding.DecodeString(a.PublicKey)
	if err != nil || len(encoded) != ed25519.PublicKeySize {
		return nil, errors.New("qualification authority public key is invalid")
	}
	return ed25519.PublicKey(encoded), nil
}

func decodeQualificationSignature(value string) ([]byte, error) {
	encoded, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil || len(encoded) != ed25519.SignatureSize {
		return nil, errors.New("qualification receipt signature is invalid")
	}
	return encoded, nil
}

func (m ReleaseManifest) Validate() error {
	if m.Release == "" || m.FormatVersion != ReleaseFormatVersion ||
		m.ExperimentFormatVersion != FormatVersion || m.LeanVersion == "" {
		return errors.New("complete release identity and format versions are required")
	}
	if m.Status != "candidate" && m.Status != "qualified" {
		return fmt.Errorf("unknown release status %q", m.Status)
	}
	for name, hash := range map[string]string{
		"catalog": m.CatalogHash, "descriptor": m.DescriptorHash, "monitor": m.MonitorSemanticHash,
		"composition": m.CompositionSemanticHash, "parity": m.ParitySemanticHash,
		"checker coverage": m.CheckerCoverageHash,
	} {
		if !validHash(hash) {
			return fmt.Errorf("release %s hash is invalid", name)
		}
	}
	if len(m.Experiments) == 0 || len(m.ProofManifests) == 0 {
		return errors.New("release experiments and proof manifests are required")
	}
	for identifier, experiment := range m.Experiments {
		if identifier == "" || !validHash(experiment.SemanticHash) || !validHash(experiment.Digest) {
			return errors.New("release experiment identity, semantic hash, and digest are required")
		}
	}
	proofManifestIdentifiers := make(map[string]struct{}, len(m.ProofManifests))
	for _, manifest := range m.ProofManifests {
		if manifest.Identifier == "" {
			return errors.New("release proof manifest identity is required")
		}
		if !validHash(manifest.Digest) {
			return fmt.Errorf("release proof manifest digest for %q is invalid", manifest.Identifier)
		}
		if _, duplicate := proofManifestIdentifiers[manifest.Identifier]; duplicate {
			return fmt.Errorf("release contains duplicate proof manifest %q", manifest.Identifier)
		}
		proofManifestIdentifiers[manifest.Identifier] = struct{}{}
	}
	profiles := append([]string(nil), m.Profiles...)
	slices.Sort(profiles)
	if !slices.Equal(profiles, requiredReleaseProfiles) {
		return fmt.Errorf("release profiles %v do not cover %v", profiles, requiredReleaseProfiles)
	}
	if m.Migration.FormatVersion != "umpire3/migration-ledger/v3" ||
		!validHash(m.Migration.LedgerHash) || m.Migration.BehaviorCount <= 0 {
		return errors.New("complete root-test migration evidence is required")
	}
	if m.Migration.ExactCount < 0 || m.Migration.SemanticEquivalentCount < 0 ||
		m.Migration.PartialCount < 0 || m.Migration.InventoryOnlyCount < 0 ||
		m.Migration.ExactCount+m.Migration.SemanticEquivalentCount+
			m.Migration.PartialCount+m.Migration.InventoryOnlyCount != m.Migration.BehaviorCount {
		return errors.New("release migration fidelity counts do not match the behavior count")
	}
	if err := m.Assurance.Validate(); err != nil {
		return err
	}
	if m.Status == "qualified" && !m.Assurance.Complete() {
		return errors.New("qualified release assurance has unresolved omissions")
	}
	for _, name := range []string{"support", "authoring", "modeling", "operations", "security", "incident-recovery"} {
		if m.Documents[name] == "" {
			return fmt.Errorf("release document %q is required", name)
		}
	}
	if m.Status == "qualified" && len(m.ExternalQualifications) != 0 {
		return errors.New("qualified release cannot retain external qualification gates")
	}
	externalProfiles := make(map[string]struct{}, len(m.ExternalQualifications))
	for _, qualification := range m.ExternalQualifications {
		if qualification.Profile == "" || qualification.Command == "" || qualification.Status != "required" {
			return errors.New("external qualifications require a profile, command, and required status")
		}
		if qualification.Authority != nil {
			if err := qualification.Authority.Validate(); err != nil {
				return fmt.Errorf("external qualification %q: %w", qualification.Profile, err)
			}
		}
		if _, duplicate := externalProfiles[qualification.Profile]; duplicate {
			return fmt.Errorf("release contains duplicate external qualification %q", qualification.Profile)
		}
		externalProfiles[qualification.Profile] = struct{}{}
	}
	if m.Status == "candidate" {
		profiles = profiles[:0]
		for profile := range externalProfiles {
			profiles = append(profiles, profile)
		}
		slices.Sort(profiles)
		if !slices.Equal(profiles, requiredReleaseProfiles) {
			return fmt.Errorf("candidate release qualification gates cover %v, expected %v",
				profiles, requiredReleaseProfiles)
		}
	}
	if m.Status == "qualified" && (m.Migration.PartialCount != 0 || m.Migration.InventoryOnlyCount != 0) {
		return errors.New("qualified release migration behaviors remain partial")
	}
	if m.Status == "candidate" && len(m.Qualifications) != 0 {
		return errors.New("candidate release cannot contain completed qualification evidence")
	}
	if err := m.validateQualificationEvidence(); err != nil {
		return err
	}
	return nil
}

func (m ReleaseManifest) validateQualificationEvidence() error {
	if m.Status != "qualified" {
		return nil
	}
	profiles := make([]string, 0, len(m.Qualifications))
	seenProfiles := make(map[string]struct{}, len(m.Qualifications))
	experimentID := ""
	experimentDigest := ""
	candidateDigest := ""
	for _, qualification := range m.Qualifications {
		if err := qualification.QualificationReceipt.Validate(); err != nil {
			return fmt.Errorf("qualified release profile %q: %w", qualification.Profile, err)
		}
		if !validHash(qualification.ReceiptDigest) {
			return errors.New("qualified release contains incomplete qualification evidence")
		}
		receiptDigest, err := qualification.QualificationReceipt.Digest()
		if err != nil {
			return err
		}
		if qualification.ReceiptDigest != receiptDigest {
			return fmt.Errorf("qualified release profile %q receipt digest is invalid", qualification.Profile)
		}
		if qualification.Release != m.Release {
			return fmt.Errorf("qualified release profile %q references release %q", qualification.Profile, qualification.Release)
		}
		if err := qualification.QualificationReceipt.Verify(qualification.Authority); err != nil {
			return fmt.Errorf("qualified release profile %q: %w", qualification.Profile, err)
		}
		if _, released := m.Experiments[qualification.ExperimentID]; !released {
			return fmt.Errorf("qualification evidence references unreleased experiment %q", qualification.ExperimentID)
		}
		if _, duplicate := seenProfiles[qualification.Profile]; duplicate {
			return fmt.Errorf("qualified release contains duplicate qualification evidence for %q", qualification.Profile)
		}
		seenProfiles[qualification.Profile] = struct{}{}
		profiles = append(profiles, qualification.Profile)
		if experimentID == "" {
			experimentID = qualification.ExperimentID
			experimentDigest = qualification.ExperimentDigest
			candidateDigest = qualification.ReleaseDigest
		} else if qualification.ExperimentID != experimentID || qualification.ExperimentDigest != experimentDigest {
			return errors.New("qualified release profiles do not share the same experiment digest")
		} else if qualification.ReleaseDigest != candidateDigest {
			return errors.New("qualified release evidence does not share one candidate release digest")
		}
	}
	slices.Sort(profiles)
	if !slices.Equal(profiles, requiredReleaseProfiles) {
		return fmt.Errorf("qualified release qualification evidence covers %v, expected %v",
			profiles, requiredReleaseProfiles)
	}
	return nil
}

func (m ReleaseManifest) ValidateArtifactBindingsAgainstCurrent() error {
	if err := m.Validate(); err != nil {
		return err
	}
	catalog, err := DefaultCatalog()
	if err != nil {
		return err
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return err
	}
	if m.LeanVersion != catalog.LeanVersion {
		return fmt.Errorf("release Lean version %q does not match current %q", m.LeanVersion, catalog.LeanVersion)
	}
	monitors, err := DefaultMonitorCatalog()
	if err != nil {
		return err
	}
	composition, err := DefaultComposition()
	if err != nil {
		return err
	}
	parity, err := DefaultParityLedger()
	if err != nil {
		return err
	}
	checkerCoverage, err := DefaultCheckerCoverage()
	if err != nil {
		return err
	}
	checkerCoverageJSON, err := checkerCoverage.CanonicalJSON()
	if err != nil {
		return err
	}
	protobuf, err := DefaultProtobufInventory()
	if err != nil {
		return err
	}
	proofManifests, err := DefaultProofManifests()
	if err != nil {
		return err
	}
	for name, values := range map[string][2]string{
		"catalog": {m.CatalogHash, catalogHash}, "descriptor": {m.DescriptorHash, protobuf.DescriptorDigest},
		"monitor":          {m.MonitorSemanticHash, monitors.SemanticHash},
		"composition":      {m.CompositionSemanticHash, composition.SemanticHash},
		"parity":           {m.ParitySemanticHash, parity.SemanticHash},
		"checker coverage": {m.CheckerCoverageHash, digestBytes(checkerCoverageJSON)},
	} {
		if values[0] != values[1] {
			return fmt.Errorf("release %s hash %q does not match current %q", name, values[0], values[1])
		}
	}
	currentProofManifests := make(map[string]string, len(proofManifests))
	for _, manifest := range proofManifests {
		digest, err := manifest.Digest()
		if err != nil {
			return err
		}
		currentProofManifests[manifest.Identifier] = digest
	}
	if len(m.ProofManifests) != len(currentProofManifests) {
		return errors.New("release proof manifests do not match current proof manifests")
	}
	for _, manifest := range m.ProofManifests {
		currentDigest, exists := currentProofManifests[manifest.Identifier]
		if !exists || manifest.Digest != currentDigest {
			return fmt.Errorf("release proof manifest %q does not match current artifact", manifest.Identifier)
		}
	}
	if m.Status == "qualified" {
		if err := validateQualifiedComposition(composition); err != nil {
			return err
		}
		if err := validateQualifiedParity(parity); err != nil {
			return err
		}
	}
	return nil
}

func validateQualifiedComposition(composition Composition) error {
	if composition.ResultClass != ResultClassCompositionProved {
		return errors.New("qualified release composition evidence is not proof-backed")
	}
	if len(composition.MissingMetadata()) != 0 {
		return errors.New("qualified release composition has missing metadata")
	}
	return nil
}

func validateQualifiedParity(parity ParityLedger) error {
	if parity.ResultClass != ResultClassEvidenceResolved {
		return errors.New("qualified release parity evidence is not declaration-resolved")
	}
	for _, entry := range parity.Entries {
		if entry.EvidenceStatus != MetadataPresent || entry.Disposition == ParityNotYetImplemented ||
			entry.Fidelity == FidelityPartial || entry.Fidelity == FidelityInventoryOnly ||
			(entry.EvidenceLevel != EvidenceLocalIntegration && entry.EvidenceLevel != EvidenceProfileQualified) {
			return fmt.Errorf("qualified release parity entry %q is incomplete", entry.LegacyName)
		}
	}
	return nil
}

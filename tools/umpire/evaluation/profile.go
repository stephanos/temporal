package evaluation

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
)

// The Profiles Lean declares, rendered by `make umpire-gen-evaluation-profiles`. They ship with the
// command, so they live beside the code rather than under testdata.
//
//go:embed profiles/*.json
var embeddedProfiles embed.FS

// The closed vocabularies a Profile is written in, as Umpire.Evaluation spells them.
var (
	conditions = []string{
		"verdict-violated", "verdict-inconclusive", "disposition-stopped", "disposition-incomplete",
		"cleanup-unclosed", "known-gap-blocking", "unsupported-rule",
	}
	forcedDecisions = []string{DecisionRejected, DecisionIncomplete}
	knownGapKinds   = []string{"capability", "input", "interpretation", "claim"}
)

// The decisions an assessment reaches. A reason forces rejected or incomplete; accepted is what
// remains when no reason holds.
const (
	DecisionAccepted   = "accepted"
	DecisionRejected   = "rejected"
	DecisionIncomplete = "incomplete"
)

// ErrUnknownProfile says a name is not one of the embedded Profiles.
var ErrUnknownProfile = errors.New("no such Evaluation Profile")

// profileFormatVersion is the rendered Profile format this reader knows.
const profileFormatVersion = 1

// Reason is one row of a Profile's reason table: its name, the condition it tests and the decision
// it forces.
type Reason struct {
	Name      string `json:"name"`
	Condition string `json:"condition"`
	Decision  string `json:"decision"`
}

// Profile is one loaded, validated Evaluation Profile.
type Profile struct {
	Version           int      `json:"version"`
	Name              string   `json:"name"`
	Claim             string   `json:"claim"`
	Trust             string   `json:"trust"`
	BlockingKnownGaps []string `json:"blockingKnownGaps"`
	Reasons           []Reason `json:"reasons"`
	// Identity is `sha256:` and the hex SHA-256 of the Profile's canonical bytes.
	Identity string `json:"-"`
}

// ProfileNames lists the embedded Profiles' names, sorted.
func ProfileNames() ([]string, error) {
	entries, err := fs.ReadDir(embeddedProfiles, "profiles")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		names = append(names, strings.TrimSuffix(entry.Name(), ".json"))
	}
	slices.Sort(names)
	return names, nil
}

// LoadProfile selects an embedded Profile by its exact name, never a path, and validates it.
func LoadProfile(name string) (*Profile, error) {
	names, err := ProfileNames()
	if err != nil {
		return nil, err
	}
	if !slices.Contains(names, name) {
		return nil, fmt.Errorf("%w: %q; the Profiles are %s", ErrUnknownProfile, name, strings.Join(names, ", "))
	}
	encoded, err := embeddedProfiles.ReadFile(path.Join("profiles", name+".json"))
	if err != nil {
		return nil, err
	}
	profile, err := ParseProfile(encoded)
	if err != nil {
		return nil, fmt.Errorf("evaluation Profile %q: %w", name, err)
	}
	if profile.Name != name {
		return nil, fmt.Errorf("the Evaluation Profile file %q names Profile %q", name, profile.Name)
	}
	return profile, nil
}

// ParseProfile decodes a rendered Profile strictly and validates it as Lean checks a declaration,
// so an assessment only ever receives a valid Profile. Strict means the bytes are exactly the
// canonical rendering of what they decode to: an unknown, repeated or case-folded key, other
// spacing or another field order is refused.
func ParseProfile(encoded []byte) (*Profile, error) {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var profile Profile
	if err := decoder.Decode(&profile); err != nil {
		return nil, fmt.Errorf("decode Profile: %w", err)
	}
	canonical, err := renderProfile(&profile)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(canonical, encoded) {
		return nil, errors.New("the Profile is not in its canonical form")
	}
	if err := validateProfile(&profile); err != nil {
		return nil, err
	}
	digest := sha256.Sum256(encoded)
	profile.Identity = "sha256:" + hex.EncodeToString(digest[:])
	return &profile, nil
}

// renderProfile is the canonical rendering Lean's Profile.render produces: compact, the fields in
// declaration order, no HTML escaping, one trailing newline.
func renderProfile(profile *Profile) ([]byte, error) {
	normalized := *profile
	if normalized.BlockingKnownGaps == nil {
		normalized.BlockingKnownGaps = []string{}
	}
	profile = &normalized
	var rendered bytes.Buffer
	encoder := json.NewEncoder(&rendered)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(profile); err != nil {
		return nil, err
	}
	return rendered.Bytes(), nil
}

func validProfileName(name string) bool {
	if name == "" || strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return false
	}
	return strings.Trim(name, "abcdefghijklmnopqrstuvwxyz0123456789-") == ""
}

func firstRepeated(values []string) string {
	seen := map[string]bool{}
	for _, value := range values {
		if seen[value] {
			return value
		}
		seen[value] = true
	}
	return ""
}

// validateProfile holds a Profile to what Umpire.Evaluation's Profile.declare checks, plus the
// closed vocabularies and the fixed kind order a rendering carries.
func validateProfile(profile *Profile) error {
	if profile.Version != profileFormatVersion {
		return fmt.Errorf("Profile format version %d, not %d", profile.Version, profileFormatVersion)
	}
	if !validProfileName(profile.Name) {
		return fmt.Errorf("Profile name %q is not lowercase letters, digits and inner hyphens", profile.Name)
	}
	if profile.Claim == "" {
		return errors.New("the Profile states no claim")
	}
	if profile.Trust == "" {
		return errors.New("the Profile states no trust basis")
	}
	if len(profile.Reasons) == 0 {
		return errors.New("the Profile's reason table is empty")
	}
	var names, named []string
	for _, reason := range profile.Reasons {
		if reason.Name == "" {
			return errors.New("a reason has an empty name")
		}
		if !slices.Contains(conditions, reason.Condition) {
			return fmt.Errorf("reason %q names unknown condition %q", reason.Name, reason.Condition)
		}
		if !slices.Contains(forcedDecisions, reason.Decision) {
			return fmt.Errorf("reason %q forces unknown decision %q", reason.Name, reason.Decision)
		}
		names = append(names, reason.Name)
		named = append(named, reason.Condition)
	}
	if repeated := firstRepeated(names); repeated != "" {
		return fmt.Errorf("reason %q is declared twice", repeated)
	}
	if repeated := firstRepeated(named); repeated != "" {
		return fmt.Errorf("condition %q is named by two reasons", repeated)
	}
	ranks := make([]int, 0, len(profile.BlockingKnownGaps))
	for _, kind := range profile.BlockingKnownGaps {
		rank := slices.Index(knownGapKinds, kind)
		if rank < 0 {
			return fmt.Errorf("unknown Known Gap kind %q", kind)
		}
		ranks = append(ranks, rank)
	}
	if repeated := firstRepeated(profile.BlockingKnownGaps); repeated != "" {
		return fmt.Errorf("blocking Known Gap kind %q is named twice", repeated)
	}
	if !slices.IsSorted(ranks) {
		return errors.New("the blocking Known Gap kinds are not in the kind order")
	}
	blocking := slices.Contains(named, "known-gap-blocking")
	if blocking && len(profile.BlockingKnownGaps) == 0 {
		return errors.New("a 'known-gap-blocking' reason is declared with no blocking kind")
	}
	if !blocking && len(profile.BlockingKnownGaps) > 0 {
		return errors.New("blocking Known Gap kinds are declared with no 'known-gap-blocking' reason")
	}
	return nil
}

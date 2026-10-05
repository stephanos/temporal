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

// The Evaluation Profiles: checked-in configuration data that no live target renders. They ship with the
// command, so they live beside the code rather than under testdata.
//
//go:embed profiles/*.json
var embeddedProfiles embed.FS

// The Known Gap kinds a Profile may name as blocking, in the order a rendering lists them.
var knownGapKinds = []string{"capability", "input", "interpretation", "claim"}

// The decisions an assessment reaches. A reason forces rejected or incomplete; accepted is what
// remains when no reason holds.
const (
	DecisionAccepted   = "accepted"
	DecisionRejected   = "rejected"
	DecisionIncomplete = "incomplete"
)

// ErrUnknownProfile says a name is not one of the embedded Profiles.
var ErrUnknownProfile = errors.New("no such Evaluation Profile")

// profileFormatVersion is the rendered Profile format this reader knows. Version 1 carried a reason
// table; the reasons are now Assess's own, and a Profile states only its policy.
const profileFormatVersion = 2

// Profile is one loaded, validated Evaluation Profile: the policy one deployment's claim is decided
// under. What decides is Assess's fixed precedence; a Profile names the claim and the trust it
// asserts, which Known Gap kinds keep a subject from being accepted, and what an unsupported rule
// forces.
type Profile struct {
	Version           int      `json:"version"`
	Name              string   `json:"name"`
	Claim             string   `json:"claim"`
	Trust             string   `json:"trust"`
	BlockingKnownGaps []string `json:"blockingKnownGaps"`
	// UnsupportedRule is the decision a rule concluded at a terminal state with no supporting event
	// forces: rejected or incomplete.
	UnsupportedRule string `json:"unsupportedRule"`
	// Identity is `sha256:` and the hex SHA-256 of the Profile's canonical bytes.
	Identity string `json:"-"`
}

// LoadProfile selects an embedded Profile by its exact name, never a path, and validates it.
func LoadProfile(name string) (*Profile, error) {
	return LoadProfileIn(embeddedProfiles, name)
}

// LoadProfileIn selects a Profile by its exact name from a set of rendered Profiles laid out as
// `profiles/<name>.json`, never by a path, and validates it; the canary loads its own set this way.
func LoadProfileIn(profiles fs.FS, name string) (*Profile, error) {
	entries, err := fs.ReadDir(profiles, "profiles")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		names = append(names, strings.TrimSuffix(entry.Name(), ".json"))
	}
	slices.Sort(names)
	if !slices.Contains(names, name) {
		return nil, fmt.Errorf("%w: %q; the Profiles are %s", ErrUnknownProfile, name, strings.Join(names, ", "))
	}
	encoded, err := fs.ReadFile(profiles, path.Join("profiles", name+".json"))
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

// ParseProfile decodes a rendered Profile strictly and validates it, so an assessment only ever
// receives a valid Profile. Strict means the bytes are exactly the
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

// renderProfile is a Profile's canonical rendering: compact, the fields in declaration order, no
// HTML escaping, one trailing newline.
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

// validateProfile holds a Profile to its format: this version, a name, a claim, a trust basis,
// blocking Known Gap kinds from the closed set, each once and in the kind order, and a decision for
// an unsupported rule.
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
	if profile.UnsupportedRule != DecisionRejected && profile.UnsupportedRule != DecisionIncomplete {
		return fmt.Errorf("an unsupported rule forces unknown decision %q", profile.UnsupportedRule)
	}
	return nil
}

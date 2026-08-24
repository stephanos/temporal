package deterministicio

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"strings"

	iowire "go.temporal.io/server/tools/gomad3/deterministicio/internal/wire"
)

const SemanticCoverageSchema = "gomad3.semantic-coverage/v1"

type SemanticCoverage struct {
	Schema string   `json:"schema"`
	Digest Digest   `json:"digest"`
	Probes []string `json:"probes"`
}

type MissingSemanticProbesError struct {
	Probes []string
}

func (err *MissingSemanticProbesError) Error() string {
	return "required semantic probes were not observed: " + strings.Join(err.Probes, ", ")
}

func DecodeSemanticCoverage(transcript []byte) (SemanticCoverage, error) {
	operations, err := DecodeTranscript(transcript)
	if err != nil {
		return SemanticCoverage{}, err
	}
	probeNames := make(map[[sha256.Size]byte]string, len(generatedBoundaryProbes))
	for _, probe := range generatedBoundaryProbes {
		var argument [8]byte
		binary.BigEndian.PutUint64(argument[:], probe.ID)
		probeNames[iowire.Hash(argument[:])] = probe.Name
	}
	observed := make(map[string]struct{}, len(probeNames))
	for _, operation := range operations {
		if operation.Name != "boundary.probe" {
			continue
		}
		name, found := probeNames[operation.ArgumentHash]
		if !found {
			return SemanticCoverage{}, fmt.Errorf("I/O transcript contains unknown boundary probe %x", operation.ArgumentHash)
		}
		if _, duplicate := observed[name]; duplicate {
			return SemanticCoverage{}, fmt.Errorf("I/O transcript contains duplicate boundary probe %s", name)
		}
		observed[name] = struct{}{}
	}
	probes := make([]string, 0, len(observed))
	for name := range observed {
		probes = append(probes, name)
	}
	return SummarizeSemanticProbes(probes)
}

func SummarizeSemanticProbes(probes []string) (SemanticCoverage, error) {
	known := make(map[string]struct{}, len(generatedBoundaryProbes))
	for _, probe := range generatedBoundaryProbes {
		known[probe.Name] = struct{}{}
	}
	unique := make(map[string]struct{}, len(probes))
	for _, probe := range probes {
		if _, found := known[probe]; !found {
			return SemanticCoverage{}, fmt.Errorf("unknown semantic probe %q", probe)
		}
		unique[probe] = struct{}{}
	}
	probes = make([]string, 0, len(unique))
	for probe := range unique {
		probes = append(probes, probe)
	}
	sort.Strings(probes)
	identity := SemanticCoverageSchema + "\x00"
	if len(probes) != 0 {
		identity += strings.Join(probes, "\n") + "\n"
	}
	hash := sha256.Sum256([]byte(identity))
	return SemanticCoverage{Schema: SemanticCoverageSchema, Digest: Digest(fmt.Sprintf("sha256:%x", hash)), Probes: probes}, nil
}

func MissingRequiredSemanticProbes(coverage SemanticCoverage, required []string) ([]string, error) {
	known := make(map[string]struct{}, len(generatedBoundaryProbes))
	for _, probe := range generatedBoundaryProbes {
		known[probe.Name] = struct{}{}
	}
	observed := make(map[string]struct{}, len(coverage.Probes))
	for _, probe := range coverage.Probes {
		observed[probe] = struct{}{}
	}
	seen := make(map[string]struct{}, len(required))
	missing := make([]string, 0, len(required))
	for _, probe := range required {
		if _, found := known[probe]; !found {
			return nil, fmt.Errorf("unknown required semantic probe %q", probe)
		}
		if _, duplicate := seen[probe]; duplicate {
			return nil, fmt.Errorf("required semantic probe is duplicated: %s", probe)
		}
		seen[probe] = struct{}{}
		if _, found := observed[probe]; !found {
			missing = append(missing, probe)
		}
	}
	sort.Strings(missing)
	return missing, nil
}

func RequiredSemanticProbes() []string {
	probes := make([]string, len(generatedBoundaryProbes))
	for index, probe := range generatedBoundaryProbes {
		probes[index] = probe.Name
	}
	return probes
}

func BoundaryManifestIdentity() (string, Digest) {
	return generatedBoundaryManifestVersion, Digest(generatedBoundaryManifestSHA256)
}

func SemanticInstrumentationIdentity() Digest {
	hasher := sha256.New()
	_, _ = hasher.Write([]byte(SemanticCoverageSchema))
	_, _ = hasher.Write([]byte{0})
	var identifier [8]byte
	for _, probe := range generatedBoundaryProbes {
		binary.BigEndian.PutUint64(identifier[:], probe.ID)
		_, _ = hasher.Write(identifier[:])
		_, _ = hasher.Write([]byte{0})
		_, _ = hasher.Write([]byte(probe.Name))
		_, _ = hasher.Write([]byte{0})
	}
	return Digest(fmt.Sprintf("sha256:%x", hasher.Sum(nil)))
}

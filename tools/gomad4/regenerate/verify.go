package regenerate

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"go.temporal.io/server/tools/gomad4/conformance"
	"go.temporal.io/server/tools/gomad4/trace"
)

const CorpusFile = "corpus.json"

type Generator interface {
	Generate(context.Context, string) error
}

type Config struct {
	ExpectedSemanticDigest string
	RequiredCoverage       []string
	MaxFiles               int
	MaxBytes               uint64
	Conformance            conformance.Limits
}

type Evidence struct {
	Schema                string   `json:"schema"`
	SemanticDigest        string   `json:"semantic_digest"`
	FirstCorpusDigest     string   `json:"first_corpus_digest"`
	SecondCorpusDigest    string   `json:"second_corpus_digest"`
	FirstOutputDigest     string   `json:"first_output_digest"`
	SecondOutputDigest    string   `json:"second_output_digest"`
	Coverage              []string `json:"coverage"`
	Files                 int      `json:"files"`
	Bytes                 uint64   `json:"bytes"`
	ConformanceTraces     int      `json:"conformance_traces"`
	ConformanceSteps      int      `json:"conformance_steps"`
	ConformanceRejections int      `json:"conformance_rejections"`
}

func Verify(ctx context.Context, generator Generator, config Config) (Evidence, error) {
	if generator == nil {
		return Evidence{}, errors.New("virtual time generator is required")
	}
	if config.ExpectedSemanticDigest == "" {
		return Evidence{}, errors.New("expected virtual time semantic digest is required")
	}
	if config.MaxFiles <= 0 || config.MaxBytes == 0 {
		return Evidence{}, errors.New("virtual time regeneration bounds must be positive")
	}
	if config.Conformance.MaxTraces == 0 && config.Conformance.MaxSteps == 0 && config.Conformance.MaxRejections == 0 {
		config.Conformance = conformance.Limits{MaxTraces: 100, MaxSteps: 10_000, MaxRejections: 1_000}
	}
	root, err := os.MkdirTemp("", "gomadv4-regenerate-verify-*")
	if err != nil {
		return Evidence{}, fmt.Errorf("create virtual time regeneration root: %w", err)
	}
	defer func() { _ = os.RemoveAll(root) }()

	outputs := make([]generatedOutput, 2)
	for index := range outputs {
		directory := filepath.Join(root, fmt.Sprintf("generation-%d", index+1))
		if err := os.Mkdir(directory, 0o700); err != nil {
			return Evidence{}, fmt.Errorf("create virtual time generation directory: %w", err)
		}
		if err := generator.Generate(ctx, directory); err != nil {
			return Evidence{}, fmt.Errorf("generate virtual time corpus attempt %d: %w", index+1, err)
		}
		outputs[index], err = inspectOutput(directory, config.MaxFiles, config.MaxBytes)
		if err != nil {
			return Evidence{}, fmt.Errorf("inspect virtual time corpus attempt %d: %w", index+1, err)
		}
		if outputs[index].corpus.SemanticDigest != config.ExpectedSemanticDigest {
			return Evidence{}, fmt.Errorf("virtual time semantic digest = %q, want locked %q", outputs[index].corpus.SemanticDigest, config.ExpectedSemanticDigest)
		}
		if err := requireCoverage(outputs[index].corpus.Coverage, config.RequiredCoverage); err != nil {
			return Evidence{}, err
		}
	}
	if !bytes.Equal(outputs[0].canonical, outputs[1].canonical) {
		return Evidence{}, fmt.Errorf("virtual time generation is not byte-stable: %s != %s", outputs[0].digest, outputs[1].digest)
	}
	if outputs[0].corpus.CorpusDigest != outputs[1].corpus.CorpusDigest {
		return Evidence{}, errors.New("virtual time corpus digests differ across clean generations")
	}
	report, err := conformance.Replay(outputs[0].corpus, config.Conformance)
	if err != nil {
		return Evidence{}, fmt.Errorf("replay regenerated virtual time corpus: %w", err)
	}
	return Evidence{
		Schema: "gomadv4.regeneration-evidence/v1", SemanticDigest: outputs[0].corpus.SemanticDigest,
		FirstCorpusDigest: outputs[0].corpus.CorpusDigest, SecondCorpusDigest: outputs[1].corpus.CorpusDigest,
		FirstOutputDigest: outputs[0].digest, SecondOutputDigest: outputs[1].digest,
		Coverage: append([]string(nil), outputs[0].corpus.Coverage...), Files: outputs[0].files, Bytes: outputs[0].bytes,
		ConformanceTraces: report.Traces, ConformanceSteps: report.Steps, ConformanceRejections: report.Rejections,
	}, nil
}

type CommandGenerator struct {
	Command        []string
	Directory      string
	MaxOutputBytes uint64
}

func (generator CommandGenerator) Generate(ctx context.Context, output string) error {
	if len(generator.Command) == 0 || strings.TrimSpace(generator.Command[0]) == "" {
		return errors.New("virtual time generator command is required")
	}
	if generator.MaxOutputBytes == 0 || generator.MaxOutputBytes > uint64(^uint(0)>>1) {
		return errors.New("virtual time generator output limit is invalid")
	}
	arguments := append([]string(nil), generator.Command[1:]...)
	arguments = append(arguments, "--output", output)
	command := exec.CommandContext(ctx, generator.Command[0], arguments...)
	command.Dir = generator.Directory
	outputBuffer := &boundedBuffer{limit: generator.MaxOutputBytes}
	command.Stdout = outputBuffer
	command.Stderr = outputBuffer
	if err := command.Run(); err != nil {
		if outputBuffer.exceeded {
			return fmt.Errorf("virtual time generator diagnostics exceed %d bytes", generator.MaxOutputBytes)
		}
		return fmt.Errorf("run virtual time generator: %w: %s", err, outputBuffer.buffer.Bytes())
	}
	if outputBuffer.exceeded {
		return fmt.Errorf("virtual time generator diagnostics exceed %d bytes", generator.MaxOutputBytes)
	}
	return nil
}

type generatedOutput struct {
	canonical []byte
	digest    string
	corpus    trace.Corpus
	files     int
	bytes     uint64
}

func inspectOutput(root string, maxFiles int, maxBytes uint64) (generatedOutput, error) {
	type fileRecord struct {
		path string
		data []byte
	}
	var records []fileRecord
	var total uint64
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return fmt.Errorf("generated virtual time output %q is not a regular file", path)
		}
		if len(records) == maxFiles {
			return fmt.Errorf("generated virtual time file count exceeds %d", maxFiles)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if uint64(len(data)) > maxBytes-total {
			return fmt.Errorf("generated virtual time bytes exceed %d", maxBytes)
		}
		total += uint64(len(data))
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		records = append(records, fileRecord{path: filepath.ToSlash(relative), data: data})
		return nil
	})
	if err != nil {
		return generatedOutput{}, err
	}
	slices.SortFunc(records, func(left, right fileRecord) int { return strings.Compare(left.path, right.path) })
	var canonical bytes.Buffer
	var size [8]byte
	for _, record := range records {
		binary.BigEndian.PutUint64(size[:], uint64(len(record.path)))
		canonical.Write(size[:])
		canonical.WriteString(record.path)
		binary.BigEndian.PutUint64(size[:], uint64(len(record.data)))
		canonical.Write(size[:])
		canonical.Write(record.data)
	}
	corpusBytes, err := os.ReadFile(filepath.Join(root, CorpusFile))
	if err != nil {
		return generatedOutput{}, fmt.Errorf("read generated %s: %w", CorpusFile, err)
	}
	corpus, err := trace.Decode(corpusBytes)
	if err != nil {
		return generatedOutput{}, err
	}
	return generatedOutput{
		canonical: canonical.Bytes(), digest: digest("gomadv4.generated-output/v1", canonical.Bytes()),
		corpus: corpus, files: len(records), bytes: total,
	}, nil
}

func requireCoverage(actual, required []string) error {
	for _, feature := range required {
		if _, found := slices.BinarySearch(actual, feature); !found {
			return fmt.Errorf("regenerated virtual time corpus is missing required coverage %q", feature)
		}
	}
	return nil
}

type boundedBuffer struct {
	buffer   bytes.Buffer
	limit    uint64
	exceeded bool
}

func (buffer *boundedBuffer) Write(data []byte) (int, error) {
	remaining := buffer.limit - uint64(buffer.buffer.Len())
	if uint64(len(data)) > remaining {
		_, _ = buffer.buffer.Write(data[:int(remaining)])
		buffer.exceeded = true
		return len(data), nil
	}
	return buffer.buffer.Write(data)
}

func digest(domain string, data []byte) string {
	hasher := sha256.New()
	_, _ = hasher.Write([]byte(domain))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write(data)
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil))
}

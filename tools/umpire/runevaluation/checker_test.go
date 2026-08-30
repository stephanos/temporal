package runevaluation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/await"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

var checkerFixtureExecutable string

func TestMain(m *testing.M) {
	directory, err := os.MkdirTemp("", "umpire-run-evaluation-checker-")
	if err != nil {
		panic(err)
	}
	checkerFixtureExecutable = filepath.Join(directory, "checker-fixture")
	command := exec.Command("go", "build", "-o", checkerFixtureExecutable, "./testdata/checker")
	if output, buildErr := command.CombinedOutput(); buildErr != nil {
		panic(string(output))
	}
	m.Run()
	if err := os.RemoveAll(directory); err != nil {
		panic(err)
	}
}

func TestEncodeCheckerRequestUsesCanonicalProtocolEnvelope(t *testing.T) {
	encoded, err := encodeCheckerRequest(testCheckerRequest())
	require.NoError(t, err)
	expected, err := os.ReadFile("testdata/checker/request.json")
	require.NoError(t, err)
	require.Equal(t, expected, encoded)
}

func TestCheckerRequestWriterPreservesCanonicalStringEncoding(t *testing.T) {
	type stringProbe struct {
		Value string `json:"value"`
	}
	probe := stringProbe{Value: "<>&\x00\x01\b\f\n\r\t\"\\\u2028\u2029" + string([]byte{0xff})}

	var encoded bytes.Buffer
	require.NoError(t, writeCanonicalPrettyJSON(&encoded, probe))
	const expected = "{\n  \"value\": \"<>&\\u0000\\u0001\\b\\f\\n\\r\\t\\\"\\\\\\u2028\\u2029�\"\n}\n"
	//nolint:testifylint
	require.Equal(t, expected, encoded.String())
}

func TestDecodeCheckerResponseRequiresCanonicalClosedBindings(t *testing.T) {
	encoded, err := os.ReadFile("testdata/checker/response.json")
	require.NoError(t, err)

	response, err := decodeCheckerResponse(encoded, testCheckerRequest())
	require.NoError(t, err)
	require.Equal(t, testCheckerResponse(), response)

	for _, testCase := range []struct {
		name   string
		mutate func([]byte) []byte
	}{
		{name: "malformed", mutate: func(encoded []byte) []byte { return encoded[:len(encoded)-2] }},
		{name: "noncanonical", mutate: func(encoded []byte) []byte {
			return bytes.Replace(encoded, []byte("  \"formatVersion\""), []byte(" \"formatVersion\""), 1)
		}},
		{name: "trailing", mutate: func(encoded []byte) []byte { return append(encoded, ' ') }},
		{name: "unknown field", mutate: func(encoded []byte) []byte {
			return bytes.Replace(encoded, []byte("{\n"), []byte("{\n  \"unexpected\": true,\n"), 1)
		}},
		{name: "duplicate field", mutate: func(encoded []byte) []byte {
			return bytes.Replace(encoded, []byte("{\n"),
				[]byte("{\n  \"formatVersion\": \"umpire-semantic-check-response/v2\",\n"), 1)
		}},
		{name: "wrong handshake", mutate: func(encoded []byte) []byte {
			return bytes.Replace(encoded, []byte(checkerIdentity), []byte("temporal.checker.substituted"), 1)
		}},
		{name: "wrong version", mutate: func(encoded []byte) []byte {
			return bytes.Replace(encoded, []byte("\"checkerVersion\": 2"), []byte("\"checkerVersion\": 3"), 1)
		}},
		{name: "stale response", mutate: func(encoded []byte) []byte {
			return bytes.Replace(encoded, []byte(testDigest("3")), []byte(testDigest("9")), 1)
		}},
		{name: "open observation enum", mutate: func(encoded []byte) []byte {
			return bytes.Replace(encoded, []byte("\"unknown\""), []byte("\"maybe\""), 1)
		}},
		{name: "invalid Known Gap kind", mutate: func(encoded []byte) []byte {
			return bytes.Replace(encoded, []byte("\"kind\": \"input\""), []byte("\"kind\": \"free-form\""), 1)
		}},
		{name: "stale query identity", mutate: func(encoded []byte) []byte {
			return bytes.Replace(encoded, []byte("temporal.query.caller-closure.fixture"),
				[]byte("temporal.query.caller-closure.stale"), 1)
		}},
		{name: "divergent summary verdicts", mutate: func(encoded []byte) []byte {
			return bytes.Replace(encoded,
				[]byte("\"propertyVerdicts\": [],\n    \"queryDefinitionId\""),
				[]byte("\"propertyVerdicts\": [{}],\n    \"queryDefinitionId\""), 1)
		}},
		{name: "invalid nested enum", mutate: func(encoded []byte) []byte {
			return bytes.Replace(encoded, []byte("\"unit\": \"semantic-transitions\""),
				[]byte("\"unit\": \"invalid\""), 1)
		}},
		{name: "invalid diagnostic kind", mutate: func(encoded []byte) []byte {
			return bytes.Replace(encoded, []byte("\"kind\": \"empty-evidence\""),
				[]byte("\"kind\": \"unlisted-diagnostic\""), 1)
		}},
		{name: "stale Known Gap union", mutate: func(encoded []byte) []byte {
			return replaceLast(encoded, []byte("umpire.gap.observation.fixture"),
				[]byte("umpire.gap.observation.stale"))
		}},
		{name: "divergent semantic status", mutate: func(encoded []byte) []byte {
			return bytes.Replace(encoded, []byte("\"semanticStatus\": \"incomplete\""),
				[]byte("\"semanticStatus\": \"satisfied\""), 1)
		}},
		{name: "inconsistent outcome checksum", mutate: func(encoded []byte) []byte {
			return bytes.Replace(encoded, []byte("\"evaluationOutcomeChecksum\": null"),
				[]byte("\"evaluationOutcomeChecksum\": \""+testDigest("a")+"\""), 1)
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := decodeCheckerResponse(testCase.mutate(bytes.Clone(encoded)), testCheckerRequest())
			require.Error(t, err)
		})
	}
}

func replaceLast(value []byte, old []byte, replacement []byte) []byte {
	index := bytes.LastIndex(value, old)
	if index < 0 {
		return value
	}
	result := make([]byte, 0, len(value)-len(old)+len(replacement))
	result = append(result, value[:index]...)
	result = append(result, replacement...)
	return append(result, value[index+len(old):]...)
}

func TestCheckerRequestWriterBoundsExactNAndNPlusOne(t *testing.T) {
	request := testCheckerRequest()
	empty := ""
	request.RunKnownGaps[0].Detail = &empty
	base, err := artifact.CanonicalPretty(request)
	require.NoError(t, err)
	padding := maximumCheckerProtocolBytes - len(base)
	require.Positive(t, padding)

	exactPadding := strings.Repeat("x", padding)
	request.RunKnownGaps[0].Detail = &exactPadding
	exact := newBoundedCapture(maximumCheckerProtocolBytes, nil)
	exactWrites := &measuringWriter{writer: exact}
	require.NoError(t, writeCanonicalCheckerRequest(request, exactWrites))
	require.False(t, exact.exceeded())
	require.Equal(t, maximumCheckerProtocolBytes, exact.length())
	require.LessOrEqual(t, exact.capacity(), maximumCheckerProtocolBytes)
	require.LessOrEqual(t, exactWrites.maximumWrite, 32<<10)
	encoded, err := encodeCheckerRequest(request)
	require.NoError(t, err)
	require.Len(t, encoded, maximumCheckerProtocolBytes)
	require.LessOrEqual(t, cap(encoded), maximumCheckerProtocolBytes)

	overPadding := exactPadding + "x"
	request.RunKnownGaps[0].Detail = &overPadding
	over := newBoundedCapture(maximumCheckerProtocolBytes, nil)
	overWrites := &measuringWriter{writer: over}
	require.NoError(t, writeCanonicalCheckerRequest(request, overWrites))
	require.True(t, over.exceeded())
	require.Equal(t, maximumCheckerProtocolBytes, over.length())
	require.LessOrEqual(t, over.capacity(), maximumCheckerProtocolBytes)
	require.LessOrEqual(t, overWrites.maximumWrite, 32<<10)
	_, err = encodeCheckerRequest(request)
	require.Error(t, err)
}

type measuringWriter struct {
	writer       io.Writer
	maximumWrite int
}

func (writer *measuringWriter) Write(value []byte) (int, error) {
	writer.maximumWrite = max(writer.maximumWrite, len(value))
	return writer.writer.Write(value)
}

func TestCheckerProcessRoundTripsTheExactClosedProtocol(t *testing.T) {
	process := testCheckerProcess(t, "valid")

	response, err := process.run(context.Background(), testCheckerRequest())
	require.NoError(t, err)
	require.Equal(t, testCheckerResponse(), response)
	requireProcessReaped(t, process.controllerExecutable)
}

func TestCheckerProcessFailsClosedAndReapsEveryChild(t *testing.T) {
	for _, testCase := range []struct {
		name string
		code checkerFailureCode
	}{
		{name: "wrong-handshake", code: checkerFailureInvalidResponse},
		{name: "wrong-version", code: checkerFailureInvalidResponse},
		{name: "stale-response", code: checkerFailureInvalidResponse},
		{name: "malformed", code: checkerFailureInvalidResponse},
		{name: "noncanonical", code: checkerFailureInvalidResponse},
		{name: "trailing", code: checkerFailureInvalidResponse},
		{name: "nonzero", code: checkerFailureExit},
		{name: "stderr", code: checkerFailureStderr},
		{name: "oversized", code: checkerFailureOversized},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			process := testCheckerProcess(t, testCase.name)

			_, err := process.run(context.Background(), testCheckerRequest())
			require.Error(t, err)
			require.ErrorIs(t, err, &checkerFailure{code: testCase.code})
			require.NotContains(t, err.Error(), "fixture")
			require.NotContains(t, err.Error(), process.controllerExecutable)
			requireProcessReaped(t, process.controllerExecutable)
		})
	}
}

func TestCheckerProcessRejectsAnUnexecutableSiblingBeforeSpawn(t *testing.T) {
	controller := testController(t, "unexecutable")
	checker := filepath.Join(filepath.Dir(controller), checkerExecutableName)
	executable, err := os.ReadFile(checkerFixtureExecutable)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(checker, executable, 0o600))

	_, err = (checkerProcess{controllerExecutable: controller, timeout: checkerTimeout}).run(
		context.Background(), testCheckerRequest())
	require.ErrorIs(t, err, &checkerFailure{code: checkerFailureStart})
	require.NotContains(t, err.Error(), checker)
}

func TestCheckerProcessCancellationAndTimeoutReapTheChild(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		process := testCheckerProcess(t, "sleep")
		process.timeout = time.Second

		_, err := process.run(context.Background(), testCheckerRequest())
		require.ErrorIs(t, err, &checkerFailure{code: checkerFailureTimeout})
		requireProcessReaped(t, process.controllerExecutable)
	})

	t.Run("cancellation", func(t *testing.T) {
		process := testCheckerProcess(t, "sleep")
		ctx, cancel := context.WithCancel(context.Background())
		result := make(chan error, 1)
		go func() {
			_, err := process.run(ctx, testCheckerRequest())
			result <- err
		}()
		pidPath := filepath.Join(filepath.Dir(process.controllerExecutable), "child.pid")
		await.RequireTrue(t, func() bool {
			_, err := os.Stat(pidPath)
			return err == nil
		}, time.Second, 10*time.Millisecond)
		cancel()

		err := <-result
		require.ErrorIs(t, err, &checkerFailure{code: checkerFailureCanceled})
		requireProcessReaped(t, process.controllerExecutable)
	})
}

func TestResolveCheckerSiblingRejectsUnsafeOrMissingTargets(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		controller := testController(t, "missing")
		_, err := resolveCheckerSibling(controller)
		require.ErrorIs(t, err, &checkerFailure{code: checkerFailureMissing})
	})

	t.Run("symlink", func(t *testing.T) {
		controller := testController(t, "symlink")
		require.NoError(t, os.Symlink(checkerFixtureExecutable,
			filepath.Join(filepath.Dir(controller), checkerExecutableName)))
		_, err := resolveCheckerSibling(controller)
		require.ErrorIs(t, err, &checkerFailure{code: checkerFailureUnsafe})
	})

	t.Run("non-regular", func(t *testing.T) {
		controller := testController(t, "non-regular")
		require.NoError(t, os.Mkdir(filepath.Join(filepath.Dir(controller), checkerExecutableName), 0o700))
		_, err := resolveCheckerSibling(controller)
		require.ErrorIs(t, err, &checkerFailure{code: checkerFailureNonRegular})
	})

	t.Run("misdirected controller", func(t *testing.T) {
		realController := testController(t, "real-controller")
		aliasDirectory := t.TempDir()
		aliasController := filepath.Join(aliasDirectory, "umpire-local-run-evaluation")
		require.NoError(t, os.Symlink(realController, aliasController))
		require.NoError(t, os.Link(checkerFixtureExecutable,
			filepath.Join(aliasDirectory, checkerExecutableName)))

		_, err := resolveCheckerSibling(aliasController)
		require.ErrorIs(t, err, &checkerFailure{code: checkerFailureMissing})
	})
}

func TestResolveVerifiedCheckerSiblingRejectsChangedBytes(t *testing.T) {
	controller := testController(t, "verified-sibling")
	checker := filepath.Join(filepath.Dir(controller), checkerExecutableName)
	executable, err := os.ReadFile(checkerFixtureExecutable)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(checker, executable, 0o700))
	expected := fileSHA256(t, checker)

	resolved, err := resolveVerifiedCheckerSibling(controller, expected)
	require.NoError(t, err)
	require.Equal(t, checker, resolved)

	require.NoError(t, os.WriteFile(checker, append(executable, 0), 0o700))
	_, err = resolveVerifiedCheckerSibling(controller, expected)
	require.ErrorIs(t, err, &checkerFailure{code: checkerFailureUnsafe})
}

func TestRunFixedCheckerRequiresInstalledDigest(t *testing.T) {
	previous := installedCheckerSHA256
	installedCheckerSHA256 = ""
	t.Cleanup(func() { installedCheckerSHA256 = previous })

	_, err := runFixedChecker(context.Background(), testCheckerRequest())
	require.ErrorIs(t, err, &checkerFailure{code: checkerFailureUnsafe})
}

func TestCheckerProcessExecutesVerifiedBytesWhenSiblingChangesBeforeStart(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		mutate func(*testing.T, string)
	}{
		{
			name: "replaced",
			mutate: func(t *testing.T, checker string) {
				replacement := filepath.Join(filepath.Dir(checker), "replacement-checker")
				require.NoError(t, os.WriteFile(replacement, []byte("substituted"), 0o700))
				require.NoError(t, os.Rename(replacement, checker))
			},
		},
		{
			name: "modified in place",
			mutate: func(t *testing.T, checker string) {
				require.NoError(t, os.WriteFile(checker, []byte("substituted"), 0o700))
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			controller := testController(t, "valid-substitution-"+testCase.name)
			checker := filepath.Join(filepath.Dir(controller), checkerExecutableName)
			executable, err := os.ReadFile(checkerFixtureExecutable)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(checker, executable, 0o700))
			process := checkerProcess{
				controllerExecutable: controller,
				expectedSHA256:       fileSHA256(t, checker),
				timeout:              checkerTimeout,
				beforeStart:          func(string) { testCase.mutate(t, checker) },
			}

			response, err := process.run(context.Background(), testCheckerRequest())
			require.NoError(t, err)
			require.Equal(t, testCheckerResponse(), response)
			for _, entry := range directoryEntries(t, filepath.Dir(checker)) {
				require.False(t, strings.HasPrefix(entry.Name(), ".umpire-run-evaluation-checker-"))
			}
		})
	}
}

func TestCheckerProcessExecutesVerifiedSnapshotWhenReplacementIsAttempted(t *testing.T) {
	controller := testController(t, "valid-snapshot-substitution")
	checker := filepath.Join(filepath.Dir(controller), checkerExecutableName)
	executable, err := os.ReadFile(checkerFixtureExecutable)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(checker, executable, 0o700))
	var replacementErr error
	process := checkerProcess{
		controllerExecutable: controller,
		expectedSHA256:       fileSHA256(t, checker),
		timeout:              checkerTimeout,
		beforeStart: func(snapshot string) {
			replacement := filepath.Join(filepath.Dir(snapshot), "replacement-snapshot")
			require.NoError(t, os.WriteFile(replacement, []byte("substituted"), 0o500))
			replacementErr = os.Rename(replacement, snapshot)
		},
	}

	response, err := process.run(context.Background(), testCheckerRequest())
	require.NoError(t, err)
	require.Equal(t, testCheckerResponse(), response)
	requireProcessReaped(t, process.controllerExecutable)
	if replacementErr != nil {
		require.ErrorIs(t, replacementErr, os.ErrPermission)
	}
}

func TestResolveCheckerSiblingRequiresTheFixedControllerName(t *testing.T) {
	directory := t.TempDir()
	controller := filepath.Join(directory, "renamed-controller")
	require.NoError(t, os.WriteFile(controller, []byte("fixture"), 0o700))
	require.NoError(t, os.Link(checkerFixtureExecutable,
		filepath.Join(directory, checkerExecutableName)))

	_, err := resolveCheckerSibling(controller)
	require.ErrorIs(t, err, &checkerFailure{code: checkerFailureController})
}

func fileSHA256(t *testing.T, path string) string {
	t.Helper()
	encoded, err := os.ReadFile(path)
	require.NoError(t, err)
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func directoryEntries(t *testing.T, path string) []os.DirEntry {
	t.Helper()
	entries, err := os.ReadDir(path)
	require.NoError(t, err)
	return entries
}

func TestBoundedCheckerCaptureNeverAllocatesTheLimitPlusOneByte(t *testing.T) {
	capture := newBoundedCapture(maximumCheckerProtocolBytes, nil)
	chunk := make([]byte, 1<<20)
	for range maximumCheckerProtocolBytes / len(chunk) {
		written, err := capture.Write(chunk)
		require.NoError(t, err)
		require.Equal(t, len(chunk), written)
	}
	require.Equal(t, maximumCheckerProtocolBytes, capture.length())
	require.LessOrEqual(t, capture.capacity(), maximumCheckerProtocolBytes)
	require.False(t, capture.exceeded())

	written, err := capture.Write([]byte{0})
	require.NoError(t, err)
	require.Equal(t, 1, written)
	require.Equal(t, maximumCheckerProtocolBytes, capture.length())
	require.LessOrEqual(t, capture.capacity(), maximumCheckerProtocolBytes)
	require.True(t, capture.exceeded())
	encoded := capture.take()
	require.Len(t, encoded, maximumCheckerProtocolBytes)
	require.LessOrEqual(t, cap(encoded), maximumCheckerProtocolBytes)
}

func TestCheckerProcessSupportsConcurrentIndependentInvocations(t *testing.T) {
	const invocationCount = 8
	processes := make([]checkerProcess, invocationCount)
	for index := range invocationCount {
		processes[index] = testCheckerProcess(t, "valid-"+strconv.Itoa(index))
	}
	results := make(chan error, invocationCount)
	var wait sync.WaitGroup
	for index := range invocationCount {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			response, err := processes[index].run(context.Background(), testCheckerRequest())
			if err == nil && !reflect.DeepEqual(response, testCheckerResponse()) {
				err = io.ErrUnexpectedEOF
			}
			results <- err
		}(index)
	}
	wait.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err)
	}
}

func FuzzDecodeCheckerResponse(f *testing.F) {
	encoded, err := os.ReadFile("testdata/checker/response.json")
	require.NoError(f, err)
	f.Add(encoded)
	f.Fuzz(func(t *testing.T, candidate []byte) {
		_, _ = decodeCheckerResponse(candidate, testCheckerRequest())
	})
}

func testCheckerRequest() checkerRequest {
	return checkerRequest{
		FormatVersion:              checkerRequestFormat,
		CheckerIdentity:            checkerIdentity,
		CheckerVersion:             artifactv2.NaturalFromUint64(2),
		CheckerBehaviorFingerprint: checkerBehaviorFingerprint,
		Experiment:                 testArtifactBinding(artifactv2.ExperimentFormat, "1"),
		RuntimeConfiguration:       testArtifactBinding(artifactv2.RuntimeConfigurationFormat, "2"),
		Run:                        testArtifactBinding(artifactv2.ExperimentRunFormat, "3"),
		RawEvidence:                testArtifactBinding(artifactv2.RawEvidenceFormat, "4"),
		RunIdentity:                "temporal.run.caller-closure.fixture",
		Query: definitionReference{
			DefinitionID:        "temporal.query.caller-closure.fixture",
			BehaviorFingerprint: testDigest("5"),
		},
		Properties: []propertyReference{{
			DefinitionID:             "temporal.property.caller-closure.fixture",
			BehaviorFingerprint:      testDigest("6"),
			RequirementDefinitionIDs: []string{"temporal.requirement.caller-closure.fixture"},
		}},
		ObservationProgram: definitionReference{
			DefinitionID:        "temporal.observation.caller-closure.fixture",
			BehaviorFingerprint: testDigest("7"),
		},
		Mapping: definitionReference{
			DefinitionID:        "temporal.mapping.caller-closure.fixture",
			BehaviorFingerprint: testDigest("8"),
		},
		PhaseOutcomes:   []artifactv2.PhaseOutcome{},
		ControlAttempts: []artifactv2.ControlAttempt{},
		SourceClosures:  []artifactv2.SourceClosure{},
		CaptureStatus:   "closed",
		Sources:         []artifactv2.RawEvidenceSource{},
		Facts:           []artifactv2.RawEvidenceFact{},
		RunKnownGaps: []artifactv2.KnownGap{{
			Kind: "capability-contract", Code: "umpire.gap.run.fixture",
		}},
		RawEvidenceKnownGaps: []artifactv2.KnownGap{{
			Kind: "input", Code: "umpire.gap.raw-evidence.fixture",
		}},
	}
}

func testCheckerResponse() checkerResponse {
	return checkerResponse{
		FormatVersion:                           checkerResponseFormat,
		CheckerIdentity:                         checkerIdentity,
		CheckerVersion:                          artifactv2.NaturalFromUint64(2),
		CheckerBehaviorFingerprint:              checkerBehaviorFingerprint,
		ExperimentArtifactChecksum:              testDigest("1"),
		RuntimeConfigurationArtifactChecksum:    testDigest("2"),
		RunArtifactChecksum:                     testDigest("3"),
		RawEvidenceArtifactChecksum:             testDigest("4"),
		ExperimentBehaviorFingerprint:           testDigest("1"),
		RuntimeConfigurationBehaviorFingerprint: testDigest("2"),
		RunIdentity:                             "temporal.run.caller-closure.fixture",
		ObservationEvaluationStatus:             "unknown",
		EvidenceLinks:                           []artifactv2.EvidenceLink{},
		Dispositions:                            []artifactv2.FieldDispositionRecord{},
		Diagnostics: []artifactv2.ObservationDiagnostic{{
			Kind:                        "empty-evidence",
			ObservationPlanDefinitionID: "temporal.mapping.caller-closure.fixture",
			RelatedDefinitionIDs:        []string{},
			Alternatives:                []string{},
		}},
		ObservationKnownGaps: []artifactv2.KnownGap{{
			Kind: "interpretation", Code: "umpire.gap.observation.fixture",
		}},
		PropertyVerdicts: []artifactv2.PropertyVerdict{},
		QuerySummary: artifactv2.QuerySummary{
			QueryDefinitionID:               "temporal.query.caller-closure.fixture",
			Status:                          "incomplete",
			QueryLimits:                     testLimits(),
			RequiredPropertyDefinitionIDs:   []string{"temporal.property.caller-closure.fixture"},
			PropertyVerdicts:                []artifactv2.PropertyVerdict{},
			MissingPropertyDefinitionIDs:    []string{"temporal.property.caller-closure.fixture"},
			DuplicatePropertyDefinitionIDs:  []string{},
			UnexpectedPropertyDefinitionIDs: []string{},
			DivergentPropertyDefinitionIDs:  []string{},
			WrongQueryResultDefinitionIDs:   []string{},
			TraceIDs:                        []string{},
		},
		SemanticStatus: "incomplete",
		ResultKnownGaps: []artifactv2.KnownGap{
			{Kind: "capability-contract", Code: "umpire.gap.run.fixture"},
			{Kind: "input", Code: "umpire.gap.raw-evidence.fixture"},
			{Kind: "interpretation", Code: "umpire.gap.observation.fixture"},
		},
	}
}

func testLimits() artifactv2.Limits {
	return artifactv2.Limits{
		Behavior: artifactv2.BehaviorLimits{
			Transitions: artifactv2.Limit{
				Value: artifactv2.NaturalFromUint64(1), Unit: "semantic-transitions",
			},
			SelectedActions: artifactv2.Limit{
				Value: artifactv2.NaturalFromUint64(1), Unit: "selected-actions",
			},
		},
		Search: artifactv2.Limit{
			Value: artifactv2.NaturalFromUint64(1), Unit: "candidate-evaluations",
		},
	}
}

func testCheckerProcess(t *testing.T, mode string) checkerProcess {
	t.Helper()
	controller := testController(t, mode)
	require.NoError(t, os.Link(checkerFixtureExecutable,
		filepath.Join(filepath.Dir(controller), checkerExecutableName)))
	return checkerProcess{controllerExecutable: controller, timeout: checkerTimeout}
}

func testController(t *testing.T, mode string) string {
	t.Helper()
	directory := filepath.Join(t.TempDir(), mode)
	require.NoError(t, os.Mkdir(directory, 0o700))
	controller := filepath.Join(directory, "umpire-local-run-evaluation")
	require.NoError(t, os.WriteFile(controller, []byte("fixture"), 0o700))
	return controller
}

func requireProcessReaped(t *testing.T, controller string) {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join(filepath.Dir(controller), "child.pid"))
	require.NoError(t, err)
	pid, err := strconv.Atoi(string(encoded))
	require.NoError(t, err)
	process, err := os.FindProcess(pid)
	require.NoError(t, err)
	require.Error(t, process.Signal(syscall.Signal(0)))
}

func testArtifactBinding(format string, digit string) artifactv2.ArtifactBinding {
	return artifactv2.ArtifactBinding{
		FormatVersion:       format,
		ArtifactChecksum:    testDigest(digit),
		BehaviorFingerprint: testDigest(digit),
		ProvenanceChecksum:  testDigest(digit),
	}
}

func testDigest(digit string) string {
	return "sha256:" + strings.Repeat(digit, 64)
}

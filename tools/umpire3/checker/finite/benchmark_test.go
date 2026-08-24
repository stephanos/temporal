package finite

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tools/umpire3/protocol/checker"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

func TestBenchmarkRecordsTenXDeterminismAndRecovery(t *testing.T) {
	t.Parallel()

	view := soundView(t)
	checker := filepath.Join(t.TempDir(), "checker")
	require.NoError(t, os.WriteFile(checker, []byte("checked executable identity"), 0o700))
	report, certificate, receipt, err := runBenchmark(
		context.Background(), view,
		BenchmarkOptions{ParallelWorkers: 8, Limits: testOptions(1).Limits, CheckerCommand: []string{checker}},
		func(_ context.Context, _ []string, _ protocolchecker.FirstOrderView, certificate Certificate) (
			Receipt, CertificateCheckMeasurement, error,
		) {
			return checkedReceipt(certificate), CertificateCheckMeasurement{
				DurationNanos: 2 * time.Millisecond.Nanoseconds(), PeakMemoryBytes: 4096,
			}, nil
		},
	)
	require.NoError(t, err)
	require.Equal(t, 10, report.Replicas)
	require.Equal(t, 1, report.SearchRuns[0].Workers)
	require.Equal(t, 8, report.SearchRuns[1].Workers)
	require.Equal(t, report.SearchRuns[0].CertificateDigest, report.SearchRuns[1].CertificateDigest)
	require.Equal(t, report.Certificate.RepresentativeStates*10, report.Certificate.ExpandedStates)
	require.True(t, report.Recovery.MatchesUninterrupted)
	require.True(t, report.Recovery.PartialPublicationRecovered)
	require.NoError(t, report.Validate(view, certificate, receipt))

	encoded, err := report.CanonicalJSON(view, certificate, receipt)
	require.NoError(t, err)
	decoded, err := DecodeBenchmarkReport(bytes.NewReader(encoded), protocolexperiment.DefaultDecodeLimit,
		view, certificate, receipt)
	require.NoError(t, err)
	require.Equal(t, report, decoded)
}

func TestBenchmarkReportFailsClosedOnMissingDeterminismAndRecovery(t *testing.T) {
	t.Parallel()

	view := soundView(t)
	checker := filepath.Join(t.TempDir(), "checker")
	require.NoError(t, os.WriteFile(checker, []byte("checked executable identity"), 0o700))
	report, certificate, receipt, err := runBenchmark(
		context.Background(), view,
		BenchmarkOptions{ParallelWorkers: 4, Limits: testOptions(1).Limits, CheckerCommand: []string{checker}},
		func(_ context.Context, _ []string, _ protocolchecker.FirstOrderView, certificate Certificate) (
			Receipt, CertificateCheckMeasurement, error,
		) {
			return checkedReceipt(certificate), CertificateCheckMeasurement{
				DurationNanos: 1, PeakMemoryBytes: 1,
			}, nil
		},
	)
	require.NoError(t, err)

	for name, mutate := range map[string]func(*BenchmarkReport){
		"worker digest": func(value *BenchmarkReport) {
			value.SearchRuns[1].CertificateDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		},
		"resume": func(value *BenchmarkReport) {
			value.Recovery.MatchesUninterrupted = false
		},
		"partial publication": func(value *BenchmarkReport) {
			value.Recovery.PartialPublicationRecovered = false
		},
	} {
		t.Run(name, func(t *testing.T) {
			mutated := report
			mutated.SearchRuns = append([]SearchMeasurement(nil), report.SearchRuns...)
			mutate(&mutated)
			require.NoError(t, mutated.seal())
			require.Error(t, mutated.Validate(view, certificate, receipt))
		})
	}
}

func checkedReceipt(certificate Certificate) Receipt {
	return Receipt{
		FormatVersion: ReceiptFormatVersion, CertificateDigest: certificate.Digest,
		ViewDigest: certificate.ViewDigest, Target: certificate.Target, Property: certificate.Property,
		World: certificate.World, Variant: certificate.Variant, SemanticHash: certificate.SemanticHash,
		ResultClass:          protocolcatalog.ResultClassFiniteExhaustive,
		TrustBadge:           protocolcatalog.TrustBadgeCheckedCertificate,
		ExpandedStates:       certificate.Statistics.ExpandedStates,
		RepresentativeStates: certificate.Statistics.RepresentativeStates,
		Replicas:             certificate.Symmetry.Replicas, Nodes: certificate.Nodes, Axioms: []string{},
	}
}

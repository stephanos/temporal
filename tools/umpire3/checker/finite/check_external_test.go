package finite

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire3/internal/subprocess"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

func TestCanonicalLeanCheckerAcceptsScaleCertificateAndRejectsCorruption(t *testing.T) {
	command := os.Getenv("UMPIRE3_NATIVE_CERTIFICATE_CHECK")
	if command == "" {
		t.Skip("canonical native certificate checker executable is not configured")
	}
	view := soundView(t)
	certificate, err := Produce(context.Background(), view, testOptions(8), nil)
	require.NoError(t, err)
	receipt, err := CheckCertificate(context.Background(), []string{command}, view, certificate)
	require.NoError(t, err)
	require.Equal(t, protocolcatalog.ResultClassFiniteExhaustive, receipt.ResultClass)
	require.Equal(t, protocolcatalog.TrustBadgeCheckedCertificate, receipt.TrustBadge)
	require.Equal(t, 260, receipt.ExpandedStates)

	arguments, err := certificateArguments(certificate)
	require.NoError(t, err)
	corrupted := append([]string(nil), arguments...)
	corrupted[19] = string(protocolcatalog.ActionKindPersistSuccess)
	_, err = subprocess.Run(context.Background(), subprocess.Request{
		Command: append([]string{command}, corrupted...), Timeout: 30 * time.Second,
		MaxOutputBytes: protocolexperiment.DefaultDecodeLimit,
		Limits:         subprocess.Limits{CPUSeconds: 30, MemoryBytes: 1 << 30},
	})
	require.Error(t, err)
	_, err = subprocess.Run(context.Background(), subprocess.Request{
		Command: append([]string{command}, arguments[:len(arguments)-7]...), Timeout: 30 * time.Second,
		MaxOutputBytes: protocolexperiment.DefaultDecodeLimit,
		Limits:         subprocess.Limits{CPUSeconds: 30, MemoryBytes: 1 << 30},
	})
	require.Error(t, err)
}

package native

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/process"
	"go.temporal.io/server/tests/umpire3/protocol"
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
	require.Equal(t, protocol.ResultClassFiniteExhaustive, receipt.ResultClass)
	require.Equal(t, protocol.TrustBadgeCheckedCertificate, receipt.TrustBadge)
	require.Equal(t, 260, receipt.ExpandedStates)

	arguments, err := certificateArguments(certificate)
	require.NoError(t, err)
	corrupted := append([]string(nil), arguments...)
	corrupted[19] = string(protocol.ActionKindPersistSuccess)
	_, err = process.Run(context.Background(), process.Request{
		Command: append([]string{command}, corrupted...), Timeout: 30 * time.Second,
		MaxOutputBytes: protocol.DefaultDecodeLimit,
		Limits:         process.Limits{CPUSeconds: 30, MemoryBytes: 1 << 30},
	})
	require.Error(t, err)
	_, err = process.Run(context.Background(), process.Request{
		Command: append([]string{command}, arguments[:len(arguments)-7]...), Timeout: 30 * time.Second,
		MaxOutputBytes: protocol.DefaultDecodeLimit,
		Limits:         process.Limits{CPUSeconds: 30, MemoryBytes: 1 << 30},
	})
	require.Error(t, err)
}

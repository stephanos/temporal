package authority

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

const testAPIKey = "canary-api-key-4f6d0a1c9e"

var testCoordinates = map[string]string{
	VariableGRPC:         "canary-frontend.example.internal:7233",
	VariableNamespace:    "canary-namespace-7c1",
	VariableTaskQueue:    "canary-queue-2b9",
	VariableHandlerQueue: "canary-handler-queue-5e3",
	VariableEndpoint:     "canary-endpoint-8d4",
}

// testPair is a fresh self-signed certificate and its key, PEM-encoded as the environment holds them.
func testPair(t *testing.T) (certificatePEM string, keyPEM string) {
	t.Helper()
	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "canary-client"},
		NotBefore: time.Unix(0, 0), NotAfter: time.Unix(1<<32, 0),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &private.PublicKey, private)
	require.NoError(t, err)
	keyDER, err := x509.MarshalECPrivateKey(private)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))
}

func environment(extra map[string]string) Lookup {
	values := map[string]string{}
	for key, value := range testCoordinates {
		values[key] = value
	}
	for key, value := range extra {
		values[key] = value
	}
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func TestLoadCoordinatesRequiresEach(t *testing.T) {
	coordinates, err := LoadCoordinates(environment(nil))
	require.NoError(t, err)
	require.Equal(t, testCoordinates[VariableGRPC], coordinates.GRPC)
	require.Equal(t, testCoordinates[VariableEndpoint], coordinates.Driver().NexusEndpoint)
	require.Equal(t, testCoordinates[VariableHandlerQueue], coordinates.Driver().HandlerTaskQueue)
	for variable := range testCoordinates {
		t.Run(variable, func(t *testing.T) {
			_, err := LoadCoordinates(environment(map[string]string{variable: ""}))
			var refusal *Error
			require.ErrorAs(t, err, &refusal)
			require.Equal(t, variable, refusal.Variable)
		})
	}
}

// Every credential yields a TLS transport for both the Driver's endpoint and the SDK client; none
// is plaintext.
func TestLoadBuildsATLSTransportFromEachCredential(t *testing.T) {
	certificate, key := testPair(t)
	for name, credential := range map[string]map[string]string{
		"a TLS pair": {VariableTLSCert: certificate, VariableTLSKey: key},
		"an API key": {VariableAPIKey: testAPIKey},
		"both":       {VariableTLSCert: certificate, VariableTLSKey: key, VariableAPIKey: testAPIKey},
	} {
		t.Run(name, func(t *testing.T) {
			loaded, err := Load(environment(credential))
			require.NoError(t, err)
			transport := loaded.Transport
			require.NoError(t, transport.Validate())
			require.Equal(t, "tls", transport.Credentials.Info().SecurityProtocol)
			require.NotNil(t, transport.ClientTLS)
			require.GreaterOrEqual(t, transport.ClientTLS.MinVersion, uint16(0x0303))
			_, withPair := credential[VariableTLSCert]
			require.Equal(t, withPair, len(transport.ClientTLS.Certificates) == 1)

			_, withKey := credential[VariableAPIKey]
			require.Equal(t, withKey, transport.PerRPCCredentials != nil)
			require.Equal(t, withKey, transport.ClientCredentials != nil)
			if withKey {
				require.True(t, transport.PerRPCCredentials.RequireTransportSecurity())
				header, err := transport.PerRPCCredentials.GetRequestMetadata(t.Context())
				require.NoError(t, err)
				require.Equal(t, map[string]string{"authorization": "Bearer " + testAPIKey}, header)
				require.NotContains(t, fmt.Sprintf("%v %+v", transport.PerRPCCredentials, transport.PerRPCCredentials), testAPIKey)
			}

			endpoint := transport.Endpoint(loaded.Coordinates.Namespace)
			require.Equal(t, testCoordinates[VariableGRPC], endpoint.Target)
			require.Equal(t, metadata.Pairs("temporal-namespace", testCoordinates[VariableNamespace]), endpoint.Metadata)
			options := transport.ClientOptions(loaded.Coordinates.Namespace, nil)
			require.Equal(t, testCoordinates[VariableGRPC], options.HostPort)
			require.Equal(t, testCoordinates[VariableNamespace], options.Namespace)
			require.NotNil(t, options.ConnectionOptions.TLS)
			require.NotSame(t, transport.ClientTLS, options.ConnectionOptions.TLS, "each client gets its own TLS config")
		})
	}
}

// No credential, half a pair or an unreadable pair refuses, naming the variable and never a value.
func TestLoadRefusesWithoutAUsableCredential(t *testing.T) {
	certificate, key := testPair(t)
	otherCertificate, _ := testPair(t)
	for name, test := range map[string]struct {
		credential map[string]string
		variable   string
	}{
		"no credential":          {nil, ""},
		"a certificate alone":    {map[string]string{VariableTLSCert: certificate}, VariableTLSKey},
		"a key alone":            {map[string]string{VariableTLSKey: key}, VariableTLSCert},
		"a mismatched pair":      {map[string]string{VariableTLSCert: otherCertificate, VariableTLSKey: key}, VariableTLSCert + " and " + VariableTLSKey},
		"a pair that is not PEM": {map[string]string{VariableTLSCert: "not-a-certificate", VariableTLSKey: "not-a-key"}, VariableTLSCert + " and " + VariableTLSKey},
	} {
		t.Run(name, func(t *testing.T) {
			loaded, err := Load(environment(test.credential))
			require.Error(t, err)
			require.Nil(t, loaded)
			if test.variable == "" {
				require.ErrorIs(t, err, ErrNoCredential)
			} else {
				var refusal *Error
				require.ErrorAs(t, err, &refusal)
				require.Equal(t, test.variable, refusal.Variable)
			}
			for _, value := range test.credential {
				requireNoneOf(t, err.Error(), value)
			}
			for _, value := range testCoordinates {
				require.NotContains(t, err.Error(), value)
			}
		})
	}
	_, err := Load(nil)
	require.Error(t, err)
}

// planted is every credential and coordinate, and the forms they are written in: whole, as a JSON
// string with escaped newlines, split across lines, and the gRPC coordinate's host alone.
func planted(t *testing.T, certificate, key string) []string {
	t.Helper()
	values := []string{certificate, key, testAPIKey}
	for _, value := range testCoordinates {
		values = append(values, value)
	}
	values = append(values, "canary-frontend.example.internal")
	for _, value := range []string{certificate, key} {
		encoded, err := json.Marshal(value)
		require.NoError(t, err)
		values = append(values, string(encoded))
	}
	return values
}

// requireNoneOf fails if text holds value, or for a PEM value any of its body lines.
func requireNoneOf(t *testing.T, text, value string) {
	t.Helper()
	require.NotContains(t, text, value)
	for line := range strings.SplitSeq(value, "\n") {
		if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "-----") {
			require.NotContains(t, text, line)
		}
	}
}

// Every credential and coordinate planted in any written text is removed, whether it is redacted
// whole, written in pieces through the Writer, or logged by an SDK logger.
func TestTheRedactorRemovesEveryPlantedValue(t *testing.T) {
	certificate, key := testPair(t)
	loaded, err := Load(environment(map[string]string{VariableTLSCert: certificate, VariableTLSKey: key, VariableAPIKey: testAPIKey}))
	require.NoError(t, err)
	values := planted(t, certificate, key)
	text := "before " + strings.Join(values, " | ") + " after\n"

	redacted := loaded.Redactor.Redact(text)
	require.Contains(t, redacted, Redacted)
	require.True(t, strings.HasPrefix(redacted, "before ") && strings.HasSuffix(redacted, " after\n"))
	for _, value := range values {
		requireNoneOf(t, redacted, value)
	}

	var written bytes.Buffer
	writer := loaded.Redactor.Writer(&written)
	for chunk := range slicesOf(text, 7) {
		_, err := writer.Write([]byte(chunk))
		require.NoError(t, err)
	}
	_, err = writer.Write([]byte("tail " + testAPIKey))
	require.NoError(t, err)
	require.NotContains(t, written.String(), "tail", "an unterminated line waits for Close")
	require.NoError(t, writer.Close())
	for _, value := range values {
		requireNoneOf(t, written.String(), value)
	}
	require.Contains(t, written.String(), "tail "+Redacted)

	var logged bytes.Buffer
	logger := loaded.Redactor.Logger(&logged)
	logger.Info("an info line is not written", "namespace", testCoordinates[VariableNamespace])
	logger.Warn("dial failed", "target", testCoordinates[VariableGRPC], "error", errors.New("bad key "+testAPIKey))
	require.NotContains(t, logged.String(), "an info line")
	require.Contains(t, logged.String(), "dial failed")
	for _, value := range values {
		requireNoneOf(t, logged.String(), value)
	}
}

// The longest value is removed first, so a coordinate that contains another is removed whole.
func TestTheRedactorRemovesTheLongestValueFirst(t *testing.T) {
	redactor := NewRedactor("canary", "canary-namespace", "")
	require.Equal(t, Redacted+" and "+Redacted, redactor.Redact("canary-namespace and canary"))
	require.Equal(t, "unchanged", NewRedactor().Redact("unchanged"))
}

func slicesOf(text string, size int) func(func(string) bool) {
	return func(yield func(string) bool) {
		for start := 0; start < len(text); start += size {
			if !yield(text[start:min(start+size, len(text))]) {
				return
			}
		}
	}
}

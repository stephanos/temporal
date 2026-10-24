// Package authority turns the protected workflow's environment into the canary's transport and its
// Redactor. Credentials and coordinates arrive only as environment variables, read through a
// Lookup the caller supplies (the process environment, or a test's); nothing reads a file or a flag
// for them, and no transport this package builds is plaintext.
package authority

import (
	"context"
	"crypto/tls"
	"errors"

	"go.temporal.io/sdk/client"
	sdklog "go.temporal.io/sdk/log"
	"go.temporal.io/server/common/testing/testpilot/temporal/act"
	"go.temporal.io/server/tools/canary/policy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// The environment variables the protected workflow sets from the production-canary environment.
const (
	VariableTLSCert   = "UMPIRE_CANARY_TLS_CERT"
	VariableTLSKey    = "UMPIRE_CANARY_TLS_KEY"
	VariableAPIKey    = "UMPIRE_CANARY_API_KEY"
	VariableGRPC      = "UMPIRE_CANARY_GRPC"
	VariableNamespace = "UMPIRE_CANARY_NAMESPACE"
	VariableTaskQueue = "UMPIRE_CANARY_TASK_QUEUE"
)

// Lookup reads one environment variable, as os.LookupEnv does.
type Lookup func(key string) (string, bool)

// ErrNoCredential is a load with neither a TLS certificate and key nor an API key.
var ErrNoCredential = errors.New("no canary credential: set " + VariableTLSCert + " and " + VariableTLSKey + ", or " + VariableAPIKey)

// Error names the variable a load refused and why, never its value.
type Error struct {
	Variable string
	Reason   string
}

func (e *Error) Error() string { return e.Variable + ": " + e.Reason }

// Coordinates are the target's raw names. Preflight compares their digests with the policy's, and
// the Redactor removes them from progress, summaries and logs. A published Run's World holds the
// namespace and task queue, which the Model needs to replay it; nothing written holds the gRPC
// target.
type Coordinates struct {
	GRPC      string
	Namespace string
	TaskQueue string
}

// Digests are the coordinates as the policy records them.
func (c Coordinates) Digests() policy.Coordinates {
	return policy.DigestsOf(c.GRPC, c.Namespace, c.TaskQueue)
}

// Redactor removes every coordinate and each given secret. It is the one list of coordinates a
// canary redacts, so neither build can leave a coordinate unredacted that the other redacts.
func (c Coordinates) Redactor(secrets ...string) *Redactor {
	return NewRedactor(append([]string{c.GRPC, c.Namespace, c.TaskQueue}, secrets...)...)
}

// LoadCoordinates reads every coordinate; each is required.
func LoadCoordinates(lookup Lookup) (Coordinates, error) {
	if lookup == nil {
		return Coordinates{}, errors.New("an environment lookup is required")
	}
	var coordinates Coordinates
	for _, field := range []struct {
		variable string
		value    *string
	}{
		{VariableGRPC, &coordinates.GRPC}, {VariableNamespace, &coordinates.Namespace},
		{VariableTaskQueue, &coordinates.TaskQueue},
	} {
		value, ok := lookup(field.variable)
		if !ok || value == "" {
			return Coordinates{}, &Error{Variable: field.variable, Reason: "is not set"}
		}
		*field.value = value
	}
	return coordinates, nil
}

// Transport is how the canary reaches its target: the Temporal Driver and the lease's SDK client
// share one target and one credential. The controller takes it as a value; Load is the untagged
// binary's only source of one.
type Transport struct {
	Target            string
	Credentials       credentials.TransportCredentials
	PerRPCCredentials credentials.PerRPCCredentials
	// ClientTLS and ClientCredentials are the SDK client's side of the same credential.
	ClientTLS         *tls.Config
	ClientCredentials client.Credentials
}

// Driver is the Temporal Driver's transport to the canary namespace. The namespace header is the
// routing key a namespace-scoped API key is checked against, which the SDK client sends itself.
func (t Transport) Driver(namespace string) act.Transport {
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(t.Credentials),
		grpc.WithChainUnaryInterceptor(func(ctx context.Context, method string, request, reply any, conn *grpc.ClientConn, invoke grpc.UnaryInvoker, options ...grpc.CallOption) error {
			return invoke(metadata.AppendToOutgoingContext(ctx, "temporal-namespace", namespace), method, request, reply, conn, options...)
		}),
	}
	if t.PerRPCCredentials != nil {
		options = append(options, grpc.WithPerRPCCredentials(t.PerRPCCredentials))
	}
	var clientTLS *tls.Config
	if t.ClientTLS != nil {
		clientTLS = t.ClientTLS.Clone()
	}
	return act.Transport{DialOptions: options, ClientTLS: clientTLS, ClientCredentials: t.ClientCredentials}
}

// ClientOptions are the SDK client's options for the canary namespace, logging through logger.
func (t Transport) ClientOptions(namespace string, logger sdklog.Logger) client.Options {
	var clientTLS *tls.Config
	if t.ClientTLS != nil {
		clientTLS = t.ClientTLS.Clone()
	}
	return client.Options{
		HostPort: t.Target, Namespace: namespace, Credentials: t.ClientCredentials, Logger: logger,
		ConnectionOptions: client.ConnectionOptions{TLS: clientTLS},
	}
}

// Authority is one loaded environment: the coordinates, the transport built from the credential,
// and the Redactor that knows every credential and coordinate.
type Authority struct {
	Coordinates Coordinates
	Transport   Transport
	Redactor    *Redactor
}

// Load reads the coordinates and the credential -- a TLS certificate and key, an API key, or both
// -- and builds a TLS transport from them. A half pair, an unreadable pair or no credential at all
// refuses, naming the variable and never its value.
func Load(lookup Lookup) (*Authority, error) {
	coordinates, err := LoadCoordinates(lookup)
	if err != nil {
		return nil, err
	}
	certificate, _ := lookup(VariableTLSCert)
	key, _ := lookup(VariableTLSKey)
	apiKey, _ := lookup(VariableAPIKey)
	if (certificate == "") != (key == "") {
		missing := VariableTLSKey
		if certificate == "" {
			missing = VariableTLSCert
		}
		return nil, &Error{Variable: missing, Reason: "is not set, but the other half of the TLS pair is"}
	}
	if certificate == "" && apiKey == "" {
		return nil, ErrNoCredential
	}
	// The server name is the target's host, which gRPC and the SDK both take from the dial target.
	config := &tls.Config{MinVersion: tls.VersionTLS12}
	transport := Transport{Target: coordinates.GRPC}
	if certificate != "" {
		pair, err := tls.X509KeyPair([]byte(certificate), []byte(key))
		if err != nil {
			return nil, &Error{Variable: VariableTLSCert + " and " + VariableTLSKey, Reason: "are not a PEM certificate and its private key"}
		}
		config.Certificates = []tls.Certificate{pair}
	}
	if apiKey != "" {
		transport.PerRPCCredentials = bearer{token: apiKey}
		transport.ClientCredentials = client.NewAPIKeyStaticCredentials(apiKey)
	}
	transport.Credentials = credentials.NewTLS(config.Clone())
	transport.ClientTLS = config
	return &Authority{Coordinates: coordinates, Transport: transport, Redactor: coordinates.Redactor(certificate, key, apiKey)}, nil
}

// bearer sends the API key on every RPC, and only over TLS.
type bearer struct{ token string }

func (b bearer) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer " + b.token}, nil
}

func (bearer) RequireTransportSecurity() bool { return true }

// String keeps the token out of any formatted value.
func (bearer) String() string { return "bearer(redacted)" }

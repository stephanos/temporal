package temporal

import (
	"crypto/tls"
	"errors"
	"net"
	"net/url"
	"strings"

	sdkclient "go.temporal.io/sdk/client"
)

func ClientOptions(address, namespace, identity, apiKey string) (sdkclient.Options, error) {
	if address == "" || namespace == "" || identity == "" {
		return sdkclient.Options{}, errors.New("temporal address, namespace, and client identity are required")
	}
	hostPort, tlsConfig, err := parseAddress(address)
	if err != nil {
		return sdkclient.Options{}, err
	}
	result := sdkclient.Options{
		HostPort: hostPort, Namespace: namespace, Identity: identity,
		ConnectionOptions: sdkclient.ConnectionOptions{TLS: tlsConfig},
	}
	if apiKey != "" {
		result.Credentials = sdkclient.NewAPIKeyStaticCredentials(apiKey)
	}
	return result, nil
}

func parseAddress(address string) (string, *tls.Config, error) {
	if !strings.Contains(address, "://") {
		if _, _, err := net.SplitHostPort(address); err != nil {
			return "", nil, errors.New("temporal address must be host:port or an HTTPS origin")
		}
		return address, nil, nil
	}
	parsed, err := url.Parse(address)
	if err != nil {
		return "", nil, errors.New("temporal address must be host:port or an HTTPS origin")
	}
	if parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" {
		return "", nil, errors.New("remote Temporal address must be an HTTPS origin")
	}
	return parsed.Host, &tls.Config{MinVersion: tls.VersionTLS12, ServerName: parsed.Hostname()}, nil
}

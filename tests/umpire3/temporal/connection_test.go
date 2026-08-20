package temporal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientOptionsAcceptLocalAndStrictRemoteAddresses(t *testing.T) {
	local, err := ClientOptions("127.0.0.1:7233", "namespace", "identity", "")
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:7233", local.HostPort)
	require.Nil(t, local.ConnectionOptions.TLS)

	remote, err := ClientOptions("https://temporal.example:7233", "namespace", "identity", "redacted")
	require.NoError(t, err)
	require.Equal(t, "temporal.example:7233", remote.HostPort)
	require.Equal(t, "temporal.example", remote.ConnectionOptions.TLS.ServerName)
	require.NotNil(t, remote.Credentials)

	_, err = ClientOptions("https://user@temporal.example/path?secret=value", "namespace", "identity", "")
	require.EqualError(t, err, "remote Temporal address must be an HTTPS origin")
}

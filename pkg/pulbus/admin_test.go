package pulbus

import (
	"path/filepath"
	"testing"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/apache/pulsar-client-go/pulsaradmin"
	"github.com/stretchr/testify/require"
)

func TestAdminValidatesConfiguration(t *testing.T) {
	for _, endpoint := range []string{"", "localhost:8080", "pulsar://localhost:6650", "http://localhost:8080?token=hidden"} {
		admin, err := NewAdmin(endpoint)
		require.Error(t, err)
		require.Nil(t, admin)
		require.NotContains(t, err.Error(), "hidden")
	}

	for _, option := range []AdminOption{
		func(config *pulsaradmin.Config) { config.AuthPlugin = "unsupported" },
		func(config *pulsaradmin.Config) { config.TLSCertFile = "certificate.pem" },
	} {
		admin, err := NewAdmin("http://localhost:8080", option)
		require.Error(t, err)
		require.Nil(t, admin)
	}

	bus, err := New("pulsar://localhost:6650", WithAdmin("http://localhost:8080", func(config *pulsaradmin.Config) {
		config.TLSTrustCertsFilePath = filepath.Join(t.TempDir(), "missing-ca.pem")
	}))
	require.Error(t, err)
	require.Nil(t, bus)

	bus = &Pulbus{}
	WithClientOptions(pulsar.ClientOptions{Authentication: pulsar.NewAuthenticationToken("example-token")})(bus)
	require.NotNil(t, bus.clientOptions.Authentication)
}

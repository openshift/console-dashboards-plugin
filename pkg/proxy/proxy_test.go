package proxy

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/require"

	datasources "github.com/openshift/console-dashboards-plugin/pkg/datasources"
)

func TestCreateProxyHandler_TLSConfiguration(t *testing.T) {
	// Test that the proxy handler correctly accepts TLS configuration
	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS13,
		CipherSuites: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
	}

	datasourceManager := datasources.NewDatasourceManager()

	// Create the proxy handler with TLS config
	handler := CreateProxyHandler(datasourceManager, tlsConfig)

	// Verify handler is created successfully
	require.NotNil(t, handler)
}

func TestCreateProxyHandler_NilTLSConfiguration(t *testing.T) {
	// Test that the proxy handler works with nil TLS configuration
	datasourceManager := datasources.NewDatasourceManager()

	// Create the proxy handler with nil TLS config
	handler := CreateProxyHandler(datasourceManager, nil)

	// Verify handler is created successfully
	require.NotNil(t, handler)
}

func TestCreateProxyHandler_SystemCADefaults(t *testing.T) {
	// Test that proxy works without any CA configuration
	datasourceManager := datasources.NewDatasourceManager()

	handler := CreateProxyHandler(datasourceManager, nil)
	require.NotNil(t, handler)

	// Verify it would use system defaults (integration test would be needed
	// to fully verify, but this confirms handler creation succeeds)
}

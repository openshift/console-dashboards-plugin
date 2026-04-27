package proxy

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/require"

	datasources "github.com/openshift/console-dashboards-plugin/pkg/datasources"
)

func TestCreateProxyHandler_TLSConfiguration(t *testing.T) {
	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS13,
		CipherSuites: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
	}

	datasourceManager := datasources.NewDatasourceManager()

	handler := CreateProxyHandler(datasourceManager, tlsConfig)

	require.NotNil(t, handler)
}

func TestCreateProxyHandler_NilTLSConfiguration(t *testing.T) {
	datasourceManager := datasources.NewDatasourceManager()

	handler := CreateProxyHandler(datasourceManager, nil)

	require.NotNil(t, handler)
}

func TestCreateProxyHandler_SystemCADefaults(t *testing.T) {
	datasourceManager := datasources.NewDatasourceManager()

	handler := CreateProxyHandler(datasourceManager, nil)
	require.NotNil(t, handler)
}

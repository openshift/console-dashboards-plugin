package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	datasources "github.com/openshift/console-dashboards-plugin/pkg/datasources"
)

func TestCreateProxyHandler_TLSConfiguration(t *testing.T) {
	datasourceManager := datasources.NewDatasourceManager()
	tlsMinVersion := uint16(tls.VersionTLS13)
	tlsCipherSuites := []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256}

	handler := CreateProxyHandler(datasourceManager, tlsMinVersion, tlsCipherSuites)

	require.NotNil(t, handler)
}

func TestCreateProxyHandler_NilTLSConfiguration(t *testing.T) {
	datasourceManager := datasources.NewDatasourceManager()

	handler := CreateProxyHandler(datasourceManager, 0, nil)

	require.NotNil(t, handler)
}

func TestCreateProxyHandler_SystemCADefaults(t *testing.T) {
	datasourceManager := datasources.NewDatasourceManager()

	handler := CreateProxyHandler(datasourceManager, 0, nil)
	require.NotNil(t, handler)
}

func startTLSServer(t *testing.T, tlsConfig *tls.Config) (*httptest.Server, string) {
	t.Helper()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	server := httptest.NewUnstartedServer(handler)
	server.TLS = tlsConfig
	server.StartTLS()

	caFile := t.TempDir() + "/ca.cert"
	certOut, err := os.Create(caFile)
	require.NoError(t, err)

	for _, cert := range server.TLS.Certificates {
		for _, certBytes := range cert.Certificate {
			_ = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
		}
	}
	certOut.Close()

	return server, caFile
}

func buildTLSConfigFromParams(t *testing.T, caFile string, tlsMinVersion uint16, tlsCipherSuites []uint16) *tls.Config {
	t.Helper()

	proxyTLSBaseConfig := &tls.Config{}

	if caFile != "" {
		serviceCertPEM, err := os.ReadFile(caFile)
		require.NoError(t, err)

		serviceProxyRootCAs := x509.NewCertPool()
		require.True(t, serviceProxyRootCAs.AppendCertsFromPEM(serviceCertPEM), "Failed to parse CA certificate")

		proxyTLSBaseConfig.RootCAs = serviceProxyRootCAs
	}

	if tlsMinVersion != 0 {
		proxyTLSBaseConfig.MinVersion = tlsMinVersion
	}
	if len(tlsCipherSuites) > 0 {
		proxyTLSBaseConfig.CipherSuites = tlsCipherSuites
	}

	return proxyTLSBaseConfig
}

func TestProxyHandler_TLSMinVersionEnforcement(t *testing.T) {
	server, caFile := startTLSServer(t, &tls.Config{
		MinVersion: tls.VersionTLS13,
	})
	defer server.Close()

	tlsConfig13 := buildTLSConfigFromParams(t, caFile, uint16(tls.VersionTLS13), nil)
	client13 := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig13,
		},
	}

	resp, err := client13.Get(server.URL + "/health")
	require.NoError(t, err, "TLS 1.3 proxy should be able to connect to TLS 1.3 server")
	resp.Body.Close()

	tlsConfig12 := buildTLSConfigFromParams(t, caFile, 0, nil)
	tlsConfig12.MaxVersion = tls.VersionTLS12

	client12 := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig12,
		},
	}

	_, err = client12.Get(server.URL + "/health")
	require.Error(t, err, "TLS 1.2 client should be rejected by TLS 1.3 server")
}

func TestProxyHandler_TLSCipherSuiteEnforcement(t *testing.T) {
	serverCipherSuite := tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256

	server, caFile := startTLSServer(t, &tls.Config{
		MaxVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{serverCipherSuite},
	})
	defer server.Close()

	tlsConfigMatch := buildTLSConfigFromParams(t, caFile, 0, []uint16{serverCipherSuite})
	tlsConfigMatch.MaxVersion = tls.VersionTLS12

	clientMatch := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfigMatch,
		},
	}

	resp, err := clientMatch.Get(server.URL + "/health")
	require.NoError(t, err, "Client with matching cipher suite should connect")
	resp.Body.Close()

	tlsConfigMismatch := buildTLSConfigFromParams(t, caFile, 0, []uint16{tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384})
	tlsConfigMismatch.MaxVersion = tls.VersionTLS12

	clientMismatch := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfigMismatch,
		},
	}

	_, err = clientMismatch.Get(server.URL + "/health")
	require.Error(t, err, "Client with non-matching cipher suite should be rejected")
}

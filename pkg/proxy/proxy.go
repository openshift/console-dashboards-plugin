package proxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	validator "github.com/asaskevich/govalidator"
	"github.com/gorilla/mux"
	datasources "github.com/openshift/console-dashboards-plugin/pkg/datasources"
	oscrypto "github.com/openshift/library-go/pkg/crypto"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("module", "proxy")

// These headers aren't things that proxies should pass along. Some are forbidden by http2.
// This fixes the bug where Chrome users saw a ERR_SPDY_PROTOCOL_ERROR for all proxied requests.
func FilterHeaders(r *http.Response) error {
	badHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Connection",
		"Transfer-Encoding",
		"Upgrade",
		"Access-Control-Allow-Headers",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Origin",
		"Access-Control-Expose-Headers",
	}
	for _, h := range badHeaders {
		r.Header.Del(h)
	}
	return nil
}

func getProxy(datasourceName string, serviceCAfile string, datasourceManager *datasources.DatasourceManager) *httputil.ReverseProxy {
	existingProxy := datasourceManager.GetProxy(datasourceName)

	if existingProxy != nil {
		return existingProxy
	}

	datasource := datasourceManager.GetDatasource(datasourceName)

	if datasource == nil {
		return nil
	}

	ca := datasourceManager.GetCA(datasourceName)
	var serviceCertPEM []byte
	var err error

	if ca != nil && len(*ca) > 0 {
		serviceCertPEM = []byte(*ca)
	} else {
		serviceCertPEM, err = os.ReadFile(serviceCAfile)
		if err != nil {
			log.Errorf("failed to read certificate file: tried '%s' and got %v", serviceCAfile, err)
			return nil
		}
	}

	if len(serviceCertPEM) == 0 {
		log.Errorf("no certificate provided. Proxy to datasource '%s' will fail", datasourceName)
		return nil
	}

	serviceProxyRootCAs := x509.NewCertPool()
	if !serviceProxyRootCAs.AppendCertsFromPEM(serviceCertPEM) {
		log.Errorf("no CA found or is invalid. Proxy to datasource '%s' will fail", datasourceName)
		return nil
	}
	serviceProxyTLSConfig := oscrypto.SecureTLSConfig(&tls.Config{
		RootCAs: serviceProxyRootCAs,
	})

	const (
		dialerKeepalive       = 30 * time.Second
		dialerTimeout         = 5 * time.Minute // Maximum request timeout for most browsers.
		tlsHandshakeTimeout   = 10 * time.Second
		websocketPingInterval = 30 * time.Second
		websocketTimeout      = 30 * time.Second
	)

	dialer := &net.Dialer{
		Timeout:   dialerTimeout,
		KeepAlive: dialerKeepalive,
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, addr)
		},
		TLSClientConfig:     serviceProxyTLSConfig,
		TLSHandshakeTimeout: tlsHandshakeTimeout,
	}

	targetURL := datasource.Spec.Plugin.Spec.DirectURL
	proxyURL, err := url.Parse(targetURL)

	if err != nil {
		log.WithError(err).Error("cannot parse direct URL", targetURL)
		return nil
	} else {
		reverseProxy := httputil.NewSingleHostReverseProxy(proxyURL)
		reverseProxy.FlushInterval = time.Millisecond * 100
		reverseProxy.Transport = transport
		reverseProxy.ModifyResponse = FilterHeaders
		datasourceManager.SetProxy(datasourceName, reverseProxy)
		return reverseProxy
	}
}

func CreateProxyHandler(serviceCAfile string, datasourceManager *datasources.DatasourceManager) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		datasourceName := vars["datasourceName"]

		if !validator.IsDNSName(datasourceName) {
			log.Error("invalid datasource name")
			http.Error(w, "invalid datasource name", http.StatusBadRequest)
			return
		}

		if len(datasourceName) == 0 {
			log.Errorf("cannot proxy request, datasource name was not provided")
			http.Error(w, "cannot proxy request, datasource name was not provided", http.StatusBadRequest)
			return
		}

		datasourceProxy := getProxy(datasourceName, serviceCAfile, datasourceManager)

		if datasourceProxy == nil {
			log.Errorf("cannot proxy request, invalid datasource proxy: %s", datasourceName)
			http.Error(w, "cannot proxy request, invalid datasource proxy", http.StatusNotFound)
			return
		}

		http.StripPrefix(fmt.Sprintf("/proxy/%s", datasourceName), http.HandlerFunc(datasourceProxy.ServeHTTP)).ServeHTTP(w, r)
	}
}

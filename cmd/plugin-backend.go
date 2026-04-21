package main

import (
	"context"
	"flag"
	"os"
	"strconv"
	"strings"

	server "github.com/openshift/console-dashboards-plugin/pkg/server"
	"github.com/sirupsen/logrus"
	k8sapiflag "k8s.io/component-base/cli/flag"
)

var (
	portArg                = flag.Int("port", 0, "server port to listen on (default: 9004)")
	certArg                = flag.String("cert", "", "cert file path to enable TLS (disabled by default)")
	keyArg                 = flag.String("key", "", "private key file path to enable TLS (disabled by default)")
	staticPathArg          = flag.String("static-path", "", "static files path to serve frontend (default: './web/dist')")
	dashboardsNamespaceArg = flag.String("dashboards-namespace", "", "namespace to watch for custom datasources for dashboards (default: 'openshift-config-managed')")
	logLevelArg            = flag.String("log-level", "error", "verbosity of logs\noptions: ['panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace']\n'trace' level will log all incoming requests\n(default 'error')")
	tlsMinVersionArg       = flag.String("tls-min-version", "", "minimum TLS version supported. Values are from tls package constants (default: VersionTLS12)")
	tlsCipherSuitesArg     = flag.String("tls-cipher-suites", "", "comma-separated list of cipher suites for the server")
)

func main() {
	flag.Parse()

	port := mergeEnvValueInt("PORT", *portArg, 9004)
	cert := mergeEnvValue("CERT_FILE_PATH", *certArg, "")
	key := mergeEnvValue("PRIVATE_KEY_FILE_PATH", *keyArg, "")
	staticPath := mergeEnvValue("CONSOLE_DASHBOARDS_PLUGIN_STATIC_PATH", *staticPathArg, "./web/dist")
	logLevel := mergeEnvValue("CONSOLE_DASHBOARDS_PLUGIN_LOG_LEVEL", *logLevelArg, "error")
	dashboardsNamespace := mergeEnvValue("DASHBOARDS_NAMESPACE", *dashboardsNamespaceArg, "openshift-config-managed")

	tlsMinVersion := mergeEnvValue("TLS_MIN_VERSION", *tlsMinVersionArg, "VersionTLS12")
	tlsCipherSuites := mergeEnvValue("TLS_CIPHER_SUITES", *tlsCipherSuitesArg, "")

	tlsMinVer, err := k8sapiflag.TLSVersion(tlsMinVersion)
	if err != nil {
		logrus.Fatalf("Invalid TLS version %q: %v", tlsMinVersion, err)
	}

	var tlsCiphers []uint16
	if tlsCipherSuites != "" {
		cipherNames := strings.Split(strings.ReplaceAll(tlsCipherSuites, " ", ""), ",")
		tlsCiphers, err = k8sapiflag.TLSCipherSuites(cipherNames)
		if err != nil {
			logrus.Fatalf("Invalid TLS cipher suites %q: %v", tlsCipherSuites, err)
		}
	}

	srv, err := server.CreateServer(context.Background(), &server.Config{
		Port:                port,
		CertFile:            cert,
		PrivateKeyFile:      key,
		StaticPath:          staticPath,
		LogLevel:            logLevel,
		DashboardsNamespace: dashboardsNamespace,
		TLSMinVersion:       tlsMinVer,
		TLSCipherSuites:     tlsCiphers,
	})
	if err != nil {
		logrus.Fatalf("Failed to create server: %v", err)
	}

	if err = srv.StartHTTPServer(); err != nil {
		logrus.Fatalf("Failed to start HTTP server: %v", err)
	}
}

func mergeEnvValue(key string, arg string, defaultValue string) string {
	if arg != "" {
		return arg
	}

	envValue := os.Getenv(key)

	if envValue != "" {
		return envValue
	}

	return defaultValue
}

func mergeEnvValueInt(key string, arg int, defaultValue int) int {
	if arg != 0 {
		return arg
	}

	envValue := os.Getenv(key)

	num, err := strconv.Atoi(envValue)
	if err != nil && num != 0 {
		return num
	}

	return defaultValue
}

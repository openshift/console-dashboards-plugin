package main

import (
	"flag"
	"os"
	"strconv"

	server "github.com/openshift/console-dashboards-plugin/pkg/server"
	"github.com/sirupsen/logrus"
)

var (
	portArg                = flag.Int("port", 0, "server port to listen on (default: 9004)")
	certArg                = flag.String("cert", "", "cert file path to enable TLS (disabled by default)")
	keyArg                 = flag.String("key", "", "private key file path to enable TLS (disabled by default)")
	staticPathArg          = flag.String("static-path", "", "static files path to serve frontend (default: './web/dist')")
	dashboardsNamespaceArg = flag.String("dashboards-namespace", "", "namespace to watch for custom datasources for dashboards (default: 'console-dashboards')")
	logLevel               = flag.String("loglevel", "info", "log level (default: info)")
	log                    = logrus.WithField("module", "main")
)

func main() {
	flag.Parse()

	port := mergeEnvValueInt("PORT", *portArg, 9004)
	cert := mergeEnvValue("CERT_FILE_PATH", *certArg, "")
	key := mergeEnvValue("PRIVATE_KEY_FILE_PATH", *keyArg, "")
	staticPath := mergeEnvValue("STATIC_PATH", *staticPathArg, "./web/dist")
	dashboardsNamespace := mergeEnvValue("DASHBOARDS_NAMESPACE", *dashboardsNamespaceArg, "console-dashboards")

	lvl, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		log.Errorf("Log level %s not recognized, using info", *logLevel)
		*logLevel = "info"
		lvl = logrus.InfoLevel
	}
	logrus.SetLevel(lvl)

	server.Start(&server.Config{
		Port:                port,
		CertFile:            cert,
		PrivateKeyFile:      key,
		StaticPath:          staticPath,
		DashboardsNamespace: dashboardsNamespace,
	})
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

package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	apiv1 "github.com/openshift/console-dashboards-plugin/pkg/api/v1"
	datasources "github.com/openshift/console-dashboards-plugin/pkg/datasources"
	proxy "github.com/openshift/console-dashboards-plugin/pkg/proxy"
	"github.com/sirupsen/logrus"
	"k8s.io/apiserver/pkg/server/dynamiccertificates"
)

var log = logrus.WithField("module", "server")

type Config struct {
	Port                int
	CertFile            string
	PrivateKeyFile      string
	StaticPath          string
	LogLevel            string
	DashboardsNamespace string
}

func Start(cfg *Config) error {
	datasourceManager := datasources.NewDatasourceManager()

	go datasourceManager.WatchDatasources(cfg.DashboardsNamespace)

	muxRouter := mux.NewRouter()

	muxRouter.PathPrefix("/health").HandlerFunc(healthHandler())
	muxRouter.PathPrefix("/proxy/{datasourceName}/").HandlerFunc(proxy.CreateProxyHandler(cfg.CertFile, datasourceManager))
	muxRouter.HandleFunc("/api/v1/datasources/{name}", apiv1.CreateDashboardsHandler(datasourceManager))
	muxRouter.PathPrefix("/").Handler(filesHandler(http.Dir(cfg.StaticPath)))

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	tlsEnabled := cfg.CertFile != "" && cfg.PrivateKeyFile != ""
	if tlsEnabled {
		// Build and run the controller which reloads the certificate and key
		// files whenever they change.
		certKeyPair, err := dynamiccertificates.NewDynamicServingContentFromFiles("serving-cert", cfg.CertFile, cfg.PrivateKeyFile)
		if err != nil {
			logrus.WithError(err).Fatal("unable to create TLS controller")
		}
		ctrl := dynamiccertificates.NewDynamicServingCertificateController(
			tlsConfig,
			nil,
			certKeyPair,
			nil,
			nil,
		)

		// Check that the cert and key files are valid.
		if err := ctrl.RunOnce(); err != nil {
			logrus.WithError(err).Fatal("invalid certificate/key files")
		}

		ctx := context.Background()
		go ctrl.Run(1, ctx.Done())
	}

	logrusLevel, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		logrus.WithError(err).Fatal("unable to set the log level")
		logrusLevel = logrus.ErrorLevel
	}

	server := http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      muxRouter,
		TLSConfig:    tlsConfig,
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 60 * time.Second,
	}

	if logrusLevel == logrus.TraceLevel {
		loggedRouter := handlers.LoggingHandler(log.Logger.Out, muxRouter)
		server.Handler = loggedRouter
	}

	if tlsEnabled {
		log.Infof("listening on https://:%d", cfg.Port)
		logrus.SetLevel(logrusLevel)
		panic(server.ListenAndServeTLS(cfg.CertFile, cfg.PrivateKeyFile))
	} else {
		log.Warn("not using TLS")
		log.Infof("listening on http://:%d", cfg.Port)
		logrus.SetLevel(logrusLevel)
		panic(server.ListenAndServe())
	}
}

func filesHandler(root http.FileSystem) http.Handler {
	fileServer := http.FileServer(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filePath := r.URL.Path

		// disable caching for plugin entry point
		if strings.HasPrefix(filePath, "/plugin-entry.js") {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Expires", "0")
		}

		fileServer.ServeHTTP(w, r)
	})
}

func healthHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
}

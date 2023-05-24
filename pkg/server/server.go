package server

import (
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
)

var log = logrus.WithField("module", "server")

type Config struct {
	Port                int
	CertFile            string
	PrivateKeyFile      string
	StaticPath          string
	DashboardsNamespace string
}

func Start(cfg *Config) error {
	datasourceManager := datasources.NewDatasourceManager()

	go datasourceManager.WatchDatasources(cfg.DashboardsNamespace)

	muxRouter := mux.NewRouter()
	muxRouter.Use(corsHeaderMiddleware(cfg))

	loggedRouter := handlers.LoggingHandler(log.Logger.Out, muxRouter)

	muxRouter.PathPrefix("/health").HandlerFunc(healthHandler())
	muxRouter.PathPrefix("/proxy/{datasourceName}/").HandlerFunc(proxy.CreateProxyHandler(cfg.CertFile, datasourceManager))
	muxRouter.HandleFunc("/api/v1/datasources/{name}", apiv1.CreateDashboardsHandler(datasourceManager))
	muxRouter.PathPrefix("/").Handler(filesHandler(http.Dir(cfg.StaticPath)))

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	server := http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      loggedRouter,
		TLSConfig:    tlsConfig,
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 60 * time.Second,
	}

	log.Infof("server listening on port: %d", cfg.Port)

	if cfg.CertFile != "" && cfg.PrivateKeyFile != "" {
		panic(server.ListenAndServeTLS(cfg.CertFile, cfg.PrivateKeyFile))
	} else {
		log.Warn("not using TLS")
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

func corsHeaderMiddleware(cfg *Config) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			headers := w.Header()
			headers.Set("Access-Control-Allow-Origin", "*")
			next.ServeHTTP(w, r)
		})
	}
}

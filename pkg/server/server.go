package server

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	apiv1 "github.com/jgbernalp/dashboards-datasource-plugin/pkg/api/v1"
	datasources "github.com/jgbernalp/dashboards-datasource-plugin/pkg/datasources"
	proxy "github.com/jgbernalp/dashboards-datasource-plugin/pkg/proxy"
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

	muxRouter.PathPrefix("/proxy/{datasourceName}/").HandlerFunc(proxy.CreateProxyHandler(cfg.CertFile, datasourceManager))
	muxRouter.HandleFunc("/api/v1/datasources/{name}", apiv1.CreateDashboardsHandler(datasourceManager))
	muxRouter.PathPrefix("/").Handler(http.FileServer(http.Dir(cfg.StaticPath)))

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	server := http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      muxRouter,
		TLSConfig:    tlsConfig,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Infof("server listening on port: %d", cfg.Port)

	if cfg.CertFile != "" && cfg.PrivateKeyFile != "" {
		panic(server.ListenAndServeTLS(cfg.CertFile, cfg.PrivateKeyFile))
	} else {
		log.Warn("not using TLS")
		panic(server.ListenAndServe())
	}
}

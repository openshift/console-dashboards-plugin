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
	"github.com/sirupsen/logrus"
	"k8s.io/apiserver/pkg/server/dynamiccertificates"

	apiv1 "github.com/openshift/console-dashboards-plugin/pkg/api/v1"
	datasources "github.com/openshift/console-dashboards-plugin/pkg/datasources"
	proxy "github.com/openshift/console-dashboards-plugin/pkg/proxy"
)

var log = logrus.WithField("module", "server")

type Config struct {
	Port                int
	CertFile            string
	PrivateKeyFile      string
	StaticPath          string
	LogLevel            string
	DashboardsNamespace string
	TLSMinVersion       uint16
	TLSCipherSuites     []uint16
}

func (c *Config) IsTLSEnabled() bool {
	return c.CertFile != "" && c.PrivateKeyFile != ""
}

func (c *Config) ValidateTLSConfig() error {
	if (c.CertFile == "") != (c.PrivateKeyFile == "") {
		return fmt.Errorf("both cert file and private key file must be set together")
	}
	return nil
}

type PluginServer struct {
	*http.Server
	Config *Config
	cancel context.CancelFunc
}

func CreateServer(ctx context.Context, cfg *Config) (*PluginServer, error) {
	if err := cfg.ValidateTLSConfig(); err != nil {
		return nil, err
	}

	serverCtx, cancel := context.WithCancel(ctx)
	httpServer, err := createHTTPServer(serverCtx, cfg)
	if err != nil {
		cancel()
		return nil, err
	}

	return &PluginServer{
		Config: cfg,
		Server: httpServer,
		cancel: cancel,
	}, nil
}

func (s *PluginServer) StartHTTPServer() error {
	if s.Config.IsTLSEnabled() {
		log.Infof("listening for https on %s", s.Server.Addr)
		log.Infof("TLS config - MinVersion: %d, CipherSuites: %d configured", s.Server.TLSConfig.MinVersion, len(s.Server.TLSConfig.CipherSuites))
		log.Info("Using dynamic certificate controller for TLS")
		return s.Server.ListenAndServeTLS(s.Config.CertFile, s.Config.PrivateKeyFile)
	}
	log.Warn("not using TLS")
	log.Infof("listening for http on %s", s.Server.Addr)
	return s.Server.ListenAndServe()
}

func (s *PluginServer) Shutdown(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.Server != nil {
		return s.Server.Shutdown(ctx)
	}
	return nil
}

func createHTTPServer(ctx context.Context, cfg *Config) (*http.Server, error) {
	datasourceManager := datasources.NewDatasourceManager()

	go datasourceManager.WatchDatasources(cfg.DashboardsNamespace)

	// Configure TLS settings first
	tlsConfig := &tls.Config{}

	tlsEnabled := cfg.IsTLSEnabled()
	if tlsEnabled {
		// Set MinVersion - default to TLS 1.2 if not specified
		if cfg.TLSMinVersion != 0 {
			tlsConfig.MinVersion = cfg.TLSMinVersion
		} else {
			tlsConfig.MinVersion = tls.VersionTLS12
		}

		if len(cfg.TLSCipherSuites) > 0 {
			tlsConfig.CipherSuites = cfg.TLSCipherSuites
		}
	} else {
		// Even for non-TLS servers, set reasonable defaults for proxy outbound connections
		tlsConfig.MinVersion = tls.VersionTLS12
	}

	// Set up router with TLS-configured proxy handler
	muxRouter := mux.NewRouter()

	muxRouter.PathPrefix("/health").HandlerFunc(healthHandler())
	muxRouter.PathPrefix("/proxy/{datasourceName}/").HandlerFunc(proxy.CreateProxyHandler(datasourceManager, tlsConfig))
	muxRouter.HandleFunc("/api/v1/datasources/{name}", apiv1.CreateDashboardsHandler(datasourceManager))
	muxRouter.PathPrefix("/").Handler(filesHandler(http.Dir(cfg.StaticPath)))

	// Set up dynamic certificate reloading for server TLS if enabled (monitoring-plugin approach)
	if tlsEnabled {
		// Build and run the controller which reloads the certificate and key
		// files whenever they change.
		certKeyPair, err := dynamiccertificates.NewDynamicServingContentFromFiles("serving-cert", cfg.CertFile, cfg.PrivateKeyFile)
		if err != nil {
			logrus.WithError(err).Fatal("unable to create TLS controller")
		}

		// Initialize cert/key content once to validate files
		if err := certKeyPair.RunOnce(ctx); err != nil {
			logrus.WithError(err).Fatal("invalid certificate/key files")
		}

		ctrl := dynamiccertificates.NewDynamicServingCertificateController(
			tlsConfig,
			nil,
			certKeyPair,
			nil,
			nil,
		)

		// Configure the server to use the cert/key pair for all client connections.
		// This is the monitoring-plugin approach: use GetConfigForClient instead of GetCertificate
		tlsConfig.GetConfigForClient = ctrl.GetConfigForClient

		// Notify cert/key file changes to the controller.
		certKeyPair.AddListener(ctrl)

		// Start certificate controllers in background
		go ctrl.Run(1, ctx.Done())
		go certKeyPair.Run(ctx, 1)
	}

	logrusLevel, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		logrus.WithError(err).Error("unable to set the log level, using default error level")
		logrusLevel = logrus.ErrorLevel
	}
	logrus.SetLevel(logrusLevel)

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

	return &server, nil
}

type headerPreservingWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (w *headerPreservingWriter) WriteHeader(statusCode int) {
	if !w.wroteHeader {
		if w.ResponseWriter.Header().Get("Cache-Control") == "" {
			w.ResponseWriter.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		}
		if w.ResponseWriter.Header().Get("Expires") == "" {
			w.ResponseWriter.Header().Set("Expires", "0")
		}
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func filesHandler(root http.FileSystem) http.Handler {
	fileServer := http.FileServer(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filePath := r.URL.Path

		// disable caching for plugin entry point
		if strings.HasPrefix(filePath, "/plugin-entry.js") {
			fileServer.ServeHTTP(&headerPreservingWriter{ResponseWriter: w}, r)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

func healthHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
}

func Start(cfg *Config) error {
	ctx := context.Background()
	server, err := CreateServer(ctx, cfg)
	if err != nil {
		return err
	}
	return server.StartHTTPServer()
}

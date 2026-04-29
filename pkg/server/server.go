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

func tlsVersionToString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("Unknown(0x%04x)", version)
	}
}

func cipherSuitesToStrings(cipherSuites []uint16) []string {
	if len(cipherSuites) == 0 {
		return nil
	}

	lookup := make(map[uint16]string)

	for _, suite := range tls.CipherSuites() {
		lookup[suite.ID] = suite.Name
	}

	for _, suite := range tls.InsecureCipherSuites() {
		lookup[suite.ID] = suite.Name
	}

	names := make([]string, len(cipherSuites))
	for i, id := range cipherSuites {
		if name, exists := lookup[id]; exists {
			names[i] = name
		} else {
			names[i] = fmt.Sprintf("Unknown(0x%04x)", id)
		}
	}

	return names
}

func validateAndGetTLSMinVersion(cfg *Config) (uint16, error) {
	if cfg.TLSMinVersion == 0 {
		return tls.VersionTLS12, nil
	}

	if cfg.TLSMinVersion < tls.VersionTLS10 || cfg.TLSMinVersion > tls.VersionTLS13 {
		return 0, fmt.Errorf("invalid TLS min version %d: must be between TLS 1.0 (%d) and TLS 1.3 (%d)",
			cfg.TLSMinVersion, tls.VersionTLS10, tls.VersionTLS13)
	}

	return cfg.TLSMinVersion, nil
}

func validateAndGetTLSCipherSuites(cfg *Config) ([]uint16, error) {
	if len(cfg.TLSCipherSuites) == 0 {
		return nil, nil
	}

	supportedSuites := tls.CipherSuites()
	supportedMap := make(map[uint16]bool)
	for _, suite := range supportedSuites {
		supportedMap[suite.ID] = true
	}

	insecureSuites := tls.InsecureCipherSuites()
	for _, suite := range insecureSuites {
		supportedMap[suite.ID] = true
	}

	for _, suite := range cfg.TLSCipherSuites {
		if !supportedMap[suite] {
			return nil, fmt.Errorf("unsupported cipher suite: 0x%04x", suite)
		}
	}

	return cfg.TLSCipherSuites, nil
}

func extractValidatedTLSParams(cfg *Config) (serverMinVersion uint16, serverCipherSuites []uint16, proxyMinVersion uint16, proxyCipherSuites []uint16, err error) {
	serverMinVersion, err = validateAndGetTLSMinVersion(cfg)
	if err != nil {
		return 0, nil, 0, nil, fmt.Errorf("server TLS min version validation failed: %w", err)
	}

	serverCipherSuites, err = validateAndGetTLSCipherSuites(cfg)
	if err != nil {
		return 0, nil, 0, nil, fmt.Errorf("server TLS cipher suites validation failed: %w", err)
	}

	proxyMinVersion = serverMinVersion
	if serverCipherSuites != nil {
		proxyCipherSuites = make([]uint16, len(serverCipherSuites))
		copy(proxyCipherSuites, serverCipherSuites)
	} else {
		proxyCipherSuites = nil
	}

	return serverMinVersion, serverCipherSuites, proxyMinVersion, proxyCipherSuites, nil
}

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

	serverMinVersion, serverCipherSuites, proxyMinVersion, proxyCipherSuites, err := extractValidatedTLSParams(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("invalid TLS configuration")
	}

	tlsConfig := &tls.Config{
		MinVersion:   serverMinVersion,
		CipherSuites: serverCipherSuites,
	}

	tlsEnabled := cfg.IsTLSEnabled()
	if tlsEnabled {
		cipherNames := cipherSuitesToStrings(serverCipherSuites)
		if len(cipherNames) > 0 {
			log.Infof("TLS enabled with MinVersion: %s, CipherSuites: %v", tlsVersionToString(serverMinVersion), cipherNames)
		} else {
			log.Infof("TLS enabled with MinVersion: %s, CipherSuites: [default]", tlsVersionToString(serverMinVersion))
		}
	} else {
		log.Infof("TLS disabled for server, but using %s for proxy outbound connections", tlsVersionToString(proxyMinVersion))
	}

	muxRouter := mux.NewRouter()

	muxRouter.PathPrefix("/health").HandlerFunc(healthHandler())
	muxRouter.PathPrefix("/proxy/{datasourceName}/").HandlerFunc(proxy.CreateProxyHandler(datasourceManager, proxyMinVersion, proxyCipherSuites))
	muxRouter.HandleFunc("/api/v1/datasources/{name}", apiv1.CreateDashboardsHandler(datasourceManager))
	muxRouter.PathPrefix("/").Handler(filesHandler(http.Dir(cfg.StaticPath)))

	if tlsEnabled {
		// Build and run the controller which reloads the certificate and key
		// files whenever they change.
		certKeyPair, err := dynamiccertificates.NewDynamicServingContentFromFiles("serving-cert", cfg.CertFile, cfg.PrivateKeyFile)
		if err != nil {
			logrus.WithError(err).Fatal("unable to create TLS controller")
		}

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

		tlsConfig.GetConfigForClient = ctrl.GetConfigForClient

		certKeyPair.AddListener(ctrl)

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

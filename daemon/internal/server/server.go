package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"plugmyai/internal/config"
	"plugmyai/internal/provider"
	"plugmyai/internal/store"
)

const Version = "0.1.0"

type Server struct {
	cfg       *config.Config
	store     *store.Store
	registry  *provider.Registry
	startTime time.Time
	httpSrv   *http.Server
}

func New(cfg *config.Config, st *store.Store, reg *provider.Registry) *Server {
	return &Server{
		cfg:       cfg,
		store:     st,
		registry:  reg,
		startTime: time.Now(),
	}
}

func (s *Server) Start(dashboardFS fs.FS) error {
	auth := &authMiddleware{
		adminToken: s.cfg.AdminToken,
		lookupApp: func(token string) (string, string, string, []string, bool) {
			app, err := s.store.GetAppByToken(token)
			if err != nil || app == nil {
				return "", "", "", nil, false
			}
			return app.ID, app.Name, app.Scope, app.Providers, true
		},
	}

	mux := http.NewServeMux()

	// Public endpoints (no auth)
	mux.HandleFunc("GET /v1/status", s.handleStatus)
	mux.HandleFunc("POST /v1/connect", s.handleConnect)
	mux.HandleFunc("GET /v1/connect/{id}", s.handleConnectPoll)

	// Pairing approval — no auth required (request ID is the capability token,
	// only visible to the local user via the browser popup)
	mux.HandleFunc("POST /v1/connect/{id}/approve", s.handleApprove)
	mux.HandleFunc("POST /v1/connect/{id}/deny", s.handleDeny)
	mux.HandleFunc("GET /v1/connect/{id}/providers", s.handleConnectProviders)
	mux.HandleFunc("GET /v1/connect/pending", s.handleListPending)

	// App endpoints (require app or admin token)
	mux.HandleFunc("GET /v1/models", auth.requireApp(s.handleModels))
	mux.HandleFunc("POST /v1/chat/completions", auth.requireApp(s.handleChatCompletions))

	// Onboarding endpoints (require admin token)
	mux.HandleFunc("GET /v1/onboarding/status", auth.requireAdmin(s.handleOnboardingStatus))
	mux.HandleFunc("POST /v1/onboarding/test-provider", auth.requireAdmin(s.handleOnboardingTestProvider))
	mux.HandleFunc("POST /v1/onboarding/complete", auth.requireAdmin(s.handleOnboardingComplete))

	// Admin endpoints (require admin token)
	mux.HandleFunc("GET /v1/history", auth.requireAdmin(s.handleListHistory))
	mux.HandleFunc("DELETE /v1/history", auth.requireAdmin(s.handleDeleteHistory))
	mux.HandleFunc("GET /v1/history/{id}", auth.requireAdmin(s.handleGetHistory))
	mux.HandleFunc("GET /v1/providers", auth.requireAdmin(s.handleProviders))
	mux.HandleFunc("GET /v1/apps", auth.requireAdmin(s.handleListApps))
	mux.HandleFunc("DELETE /v1/apps/{id}", auth.requireAdmin(s.handleRevokeApp))

	// Dashboard SPA — serve static files, fallback to index.html
	if dashboardFS != nil {
		fileServer := http.FileServerFS(dashboardFS)

		// Read index.html once at startup, inject admin token
		indexHTML, _ := fs.ReadFile(dashboardFS, "index.html")
		injectedIndex := injectAdminToken(indexHTML, s.cfg.AdminToken)

		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Try serving the file directly (non-root paths)
			if r.URL.Path != "/" {
				_, err := fs.Stat(dashboardFS, r.URL.Path[1:]) // strip leading /
				if err == nil {
					fileServer.ServeHTTP(w, r)
					return
				}
			}
			// Serve index.html with injected admin token
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(injectedIndex)
		})
	}

	handler := corsMiddleware(mux)

	addr := fmt.Sprintf(":%d", s.cfg.Port)
	s.httpSrv = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 5 * time.Minute, // long for streaming responses
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("plug-my-ai v%s listening on http://localhost%s", Version, addr)
	log.Printf("Dashboard: http://localhost%s", addr)
	log.Printf("Admin token: %s", s.cfg.AdminToken)

	return s.httpSrv.ListenAndServe()
}

func (s *Server) Shutdown() error {
	if s.httpSrv != nil {
		return s.httpSrv.Close()
	}
	return nil
}

// --- JSON helpers ---

func jsonOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func injectAdminToken(html []byte, token string) []byte {
	script := fmt.Sprintf(`<script>localStorage.setItem("pma_admin_token",%q)</script>`, token)
	// Insert right before </head>
	return []byte(
		strings.Replace(string(html), "</head>", script+"</head>", 1),
	)
}

func jsonError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"message": msg,
			"type":    "invalid_request_error",
		},
	})
}

package server

import (
	"context"
	"net/http"
	"strings"
)

// corsMiddleware adds CORS headers for localhost web apps.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Allow any origin — the daemon runs locally and auth is via bearer tokens.
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type contextKey string

const (
	ctxAppID            contextKey = "app_id"
	ctxAppName          contextKey = "app_name"
	ctxIsAdmin          contextKey = "is_admin"
	ctxAllowedProviders contextKey = "allowed_providers"
	ctxScope            contextKey = "scope"
)

// authMiddleware validates bearer tokens for API requests.
type authMiddleware struct {
	adminToken string
	lookupApp  func(token string) (appID, appName, scope string, providers []string, ok bool)
}

// requireApp validates that the request has a valid app token.
func (a *authMiddleware) requireApp(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			jsonError(w, http.StatusUnauthorized, "missing authorization token")
			return
		}

		// Admin token also works for app endpoints — always unrestricted
		if token == a.adminToken {
			ctx := context.WithValue(r.Context(), ctxAppID, "admin")
			ctx = context.WithValue(ctx, ctxAppName, "admin")
			ctx = context.WithValue(ctx, ctxIsAdmin, true)
			ctx = context.WithValue(ctx, ctxAllowedProviders, []string(nil))
			ctx = context.WithValue(ctx, ctxScope, "full")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		appID, appName, scope, providers, ok := a.lookupApp(token)
		if !ok {
			jsonError(w, http.StatusUnauthorized, "invalid or revoked token")
			return
		}

		ctx := context.WithValue(r.Context(), ctxAppID, appID)
		ctx = context.WithValue(ctx, ctxAppName, appName)
		ctx = context.WithValue(ctx, ctxScope, scope)
		ctx = context.WithValue(ctx, ctxAllowedProviders, providers)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// requireAdmin validates that the request has the admin token.
func (a *authMiddleware) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" || token != a.adminToken {
			jsonError(w, http.StatusUnauthorized, "admin access required")
			return
		}

		ctx := context.WithValue(r.Context(), ctxIsAdmin, true)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func extractToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	// Also check query param for SSE convenience
	if t := r.URL.Query().Get("token"); t != "" {
		return t
	}
	return ""
}

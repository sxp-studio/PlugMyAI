package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"plugmyai/internal/config"
	"plugmyai/internal/provider"
	"plugmyai/internal/store"
)

// --- Status ---

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	providers := s.registry.Available()
	providerNames := make([]string, len(providers))
	for i, p := range providers {
		providerNames[i] = p.Name()
	}

	jsonOK(w, map[string]any{
		"status":    "ok",
		"version":   Version,
		"uptime_s":  int(time.Since(s.startTime).Seconds()),
		"port":      s.cfg.Port,
		"providers": providerNames,
	})
}

// --- Models ---

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	models := s.registry.AllModels()
	allowed := r.Context().Value(ctxAllowedProviders).([]string)

	// Filter models to allowed providers (empty = unrestricted)
	var data []map[string]any
	for _, m := range models {
		if !providerAllowed(allowed, m.Provider) {
			continue
		}
		data = append(data, map[string]any{
			"id":       m.ID,
			"object":   "model",
			"created":  s.startTime.Unix(),
			"owned_by": m.Provider,
		})
	}

	if data == nil {
		data = []map[string]any{}
	}

	jsonOK(w, map[string]any{
		"object": "list",
		"data":   data,
	})
}

// --- Chat Completions ---

func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	var req provider.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if len(req.Messages) == 0 {
		jsonError(w, http.StatusBadRequest, "messages array is required")
		return
	}

	// Find provider for the requested model
	p := s.registry.FindProvider(req.Model)
	if p == nil {
		jsonError(w, http.StatusBadRequest, "no available provider for model: "+req.Model)
		return
	}

	// Enforce provider scoping
	allowed := r.Context().Value(ctxAllowedProviders).([]string)
	if !providerAllowed(allowed, p.ID()) {
		jsonError(w, http.StatusForbidden, "app is not allowed to use provider: "+p.ID())
		return
	}

	appID := r.Context().Value(ctxAppID).(string)
	appName := r.Context().Value(ctxAppName).(string)
	req.Scope = r.Context().Value(ctxScope).(string)
	startTime := time.Now()

	// Start completion
	stream, err := p.Complete(r.Context(), &req)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "provider error: "+err.Error())
		return
	}

	messagesJSON, _ := json.Marshal(req.Messages)

	if req.Stream {
		s.handleStreamingResponse(w, r, stream, p, appID, appName, req.Model, messagesJSON, startTime)
	} else {
		s.handleNonStreamingResponse(w, stream, p, appID, appName, req.Model, messagesJSON, startTime)
	}
}

func (s *Server) handleStreamingResponse(w http.ResponseWriter, r *http.Request, stream <-chan provider.ChatCompletionChunk, p provider.Provider, appID, appName, model string, messagesJSON []byte, startTime time.Time) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		jsonError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	completionID := "chatcmpl-" + generateShortID()
	var fullContent string
	var usage *provider.Usage

	for chunk := range stream {
		if chunk.Error != nil {
			// Send error as SSE event
			errData, _ := json.Marshal(map[string]any{
				"error": map[string]any{"message": chunk.Error.Error()},
			})
			fmt.Fprintf(w, "data: %s\n\n", errData)
			flusher.Flush()
			break
		}

		fullContent += chunk.Content
		if chunk.Usage != nil {
			usage = chunk.Usage
		}

		// OpenAI SSE format
		sseData := map[string]any{
			"id":      completionID,
			"object":  "chat.completion.chunk",
			"created": startTime.Unix(),
			"model":   model,
			"choices": []map[string]any{
				{
					"index": 0,
					"delta": map[string]any{},
				},
			},
		}

		if chunk.Content != "" {
			sseData["choices"].([]map[string]any)[0]["delta"] = map[string]any{
				"content": chunk.Content,
			}
		}

		if chunk.Done {
			sseData["choices"].([]map[string]any)[0]["finish_reason"] = chunk.FinishReason
			if usage != nil {
				sseData["usage"] = usage
			}
		}

		data, _ := json.Marshal(sseData)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	// Send [DONE] marker
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()

	// Log the request
	s.logRequest(appID, appName, model, p.ID(), messagesJSON, fullContent, usage, startTime, nil)
}

func (s *Server) handleNonStreamingResponse(w http.ResponseWriter, stream <-chan provider.ChatCompletionChunk, p provider.Provider, appID, appName, model string, messagesJSON []byte, startTime time.Time) {
	var fullContent string
	var usage *provider.Usage
	var lastErr error

	for chunk := range stream {
		if chunk.Error != nil {
			lastErr = chunk.Error
			break
		}
		fullContent += chunk.Content
		if chunk.Usage != nil {
			usage = chunk.Usage
		}
	}

	if lastErr != nil {
		jsonError(w, http.StatusInternalServerError, lastErr.Error())
		s.logRequest(appID, appName, model, p.ID(), messagesJSON, "", nil, startTime, lastErr)
		return
	}

	completionID := "chatcmpl-" + generateShortID()
	resp := map[string]any{
		"id":      completionID,
		"object":  "chat.completion",
		"created": startTime.Unix(),
		"model":   model,
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": fullContent,
				},
				"finish_reason": "stop",
			},
		},
	}

	if usage != nil {
		resp["usage"] = usage
	}

	jsonOK(w, resp)
	s.logRequest(appID, appName, model, p.ID(), messagesJSON, fullContent, usage, startTime, nil)
}

func (s *Server) logRequest(appID, appName, model, providerID string, messagesJSON []byte, content string, usage *provider.Usage, startTime time.Time, reqErr error) {
	respJSON, _ := json.Marshal(map[string]string{"content": content})

	entry := &store.HistoryEntry{
		ID:         generateShortID(),
		AppID:      appID,
		AppName:    appName,
		Model:      model,
		Provider:   providerID,
		Messages:   messagesJSON,
		Response:   respJSON,
		DurationMS: time.Since(startTime).Milliseconds(),
		Status:     "success",
	}

	if usage != nil {
		entry.TokensIn = usage.PromptTokens
		entry.TokensOut = usage.CompletionTokens
	}

	if reqErr != nil {
		entry.Status = "error"
		entry.ErrorMessage = reqErr.Error()
	}

	if err := s.store.LogRequest(entry); err != nil {
		log.Printf("failed to log request: %v", err)
	}
}

// --- Pairing Flow ---

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AppName        string `json:"app_name"`
		AppURL         string `json:"app_url"`
		AppIcon        string `json:"app_icon"`         // optional favicon URL
		RequestedScope string `json:"requested_scope"`   // "chat" (default) or "full"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.AppName == "" {
		jsonError(w, http.StatusBadRequest, "app_name is required")
		return
	}
	// Validate and default scope
	if req.RequestedScope == "" {
		req.RequestedScope = "chat"
	}
	if req.RequestedScope != "chat" && req.RequestedScope != "full" {
		jsonError(w, http.StatusBadRequest, "requested_scope must be 'chat' or 'full'")
		return
	}

	reqID := generateShortID()
	expiresAt := time.Now().Add(5 * time.Minute)

	if err := s.store.CreateConnectRequest(reqID, req.AppName, req.AppURL, req.AppIcon, req.RequestedScope, expiresAt); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to create connect request")
		return
	}

	// Show native approval dialog (macOS) or open browser (other OS)
	approveURL := fmt.Sprintf("http://localhost:%d/#/approve?req=%s", s.cfg.Port, reqID)
	showConnectDialog(req.AppName, approveURL,
		func() {
			// Approve callback — generate token and create app with "chat" scope (safe default)
			token, err := config.GenerateAppToken()
			if err != nil {
				log.Printf("dialog approve: failed to generate token: %v", err)
				return
			}
			appID := generateShortID()
			if err := s.store.CreateApp(appID, req.AppName, req.AppURL, "chat", token); err != nil {
				log.Printf("dialog approve: failed to create app: %v", err)
				return
			}
			// No provider restriction — unrestricted access
			if err := s.store.ApproveConnectRequest(reqID, token); err != nil {
				log.Printf("dialog approve: failed to approve request: %v", err)
			}
		},
		func() {
			// Deny callback
			if err := s.store.DenyConnectRequest(reqID); err != nil {
				log.Printf("dialog deny: failed to deny request: %v", err)
			}
		},
	)

	jsonOK(w, map[string]any{
		"request_id":  reqID,
		"approve_url": approveURL,
		"expires_at":  expiresAt.Format(time.RFC3339),
		"poll_url":    fmt.Sprintf("http://localhost:%d/v1/connect/%s", s.cfg.Port, reqID),
	})
}

func (s *Server) handleConnectPoll(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cr, err := s.store.GetConnectRequest(id)
	if err != nil || cr == nil {
		jsonError(w, http.StatusNotFound, "connect request not found")
		return
	}

	if time.Now().After(cr.ExpiresAt) && cr.Status == "pending" {
		jsonOK(w, map[string]any{"status": "expired"})
		return
	}

	resp := map[string]any{
		"status":          cr.Status,
		"app_name":        cr.AppName,
		"app_url":         cr.AppURL,
		"app_icon":        cr.AppIcon,
		"requested_scope": cr.RequestedScope,
	}
	if cr.Status == "approved" {
		resp["token"] = cr.Token
	}
	jsonOK(w, resp)
}

func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cr, err := s.store.GetConnectRequest(id)
	if err != nil || cr == nil {
		jsonError(w, http.StatusNotFound, "connect request not found")
		return
	}
	if cr.Status != "pending" {
		jsonError(w, http.StatusBadRequest, "request already "+cr.Status)
		return
	}
	if time.Now().After(cr.ExpiresAt) {
		jsonError(w, http.StatusBadRequest, "request expired")
		return
	}

	// Parse optional providers and scope from body
	var body struct {
		Providers []string `json:"providers"`
		Scope     string   `json:"scope"`
	}
	// Body is optional — empty body means unrestricted providers + chat scope
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&body) // ignore decode errors (empty body is fine)
	}
	if body.Scope == "" {
		body.Scope = "chat"
	}
	if body.Scope != "chat" && body.Scope != "full" {
		jsonError(w, http.StatusBadRequest, "scope must be 'chat' or 'full'")
		return
	}

	// Generate app token and create app
	token, err := config.GenerateAppToken()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	appID := generateShortID()
	if err := s.store.CreateApp(appID, cr.AppName, cr.AppURL, body.Scope, token); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to create app")
		return
	}

	// Set provider restrictions (empty = unrestricted)
	if len(body.Providers) > 0 {
		if err := s.store.SetAppProviders(appID, body.Providers); err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to set app providers")
			return
		}
	}

	if err := s.store.ApproveConnectRequest(id, token); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to approve request")
		return
	}

	jsonOK(w, map[string]any{"status": "approved", "app_id": appID})
}

func (s *Server) handleDeny(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.DenyConnectRequest(id); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to deny request")
		return
	}
	jsonOK(w, map[string]any{"status": "denied"})
}

// handleConnectProviders returns available providers for a pending connect request.
// No auth required — the request ID acts as a capability token.
func (s *Server) handleConnectProviders(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cr, err := s.store.GetConnectRequest(id)
	if err != nil || cr == nil {
		jsonError(w, http.StatusNotFound, "connect request not found")
		return
	}
	if cr.Status != "pending" {
		jsonError(w, http.StatusBadRequest, "request already "+cr.Status)
		return
	}

	all := s.registry.Available()
	providers := make([]map[string]any, len(all))
	for i, p := range all {
		providers[i] = map[string]any{
			"id":   p.ID(),
			"name": p.Name(),
		}
	}
	jsonOK(w, map[string]any{
		"providers":       providers,
		"requested_scope": cr.RequestedScope,
	})
}

func (s *Server) handleListPending(w http.ResponseWriter, r *http.Request) {
	reqs, err := s.store.ListPendingConnectRequests()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if reqs == nil {
		reqs = []store.ConnectRequest{}
	}
	jsonOK(w, reqs)
}

// --- Dashboard API ---

func (s *Server) handleListHistory(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	f := store.HistoryFilter{
		Limit:   limit,
		Offset:  offset,
		AppName: r.URL.Query().Get("app_name"),
		SortBy:  r.URL.Query().Get("sort"),
	}

	entries, total, err := s.store.ListHistoryFiltered(f)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if entries == nil {
		entries = []store.HistoryEntry{}
	}
	jsonOK(w, map[string]any{
		"entries": entries,
		"total":   total,
	})
}

func (s *Server) handleDeleteHistory(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteAllHistory(); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonOK(w, map[string]any{"status": "ok"})
}

func (s *Server) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	entry, err := s.store.GetHistoryEntry(id)
	if err != nil || entry == nil {
		jsonError(w, http.StatusNotFound, "history entry not found")
		return
	}
	jsonOK(w, entry)
}

func (s *Server) handleProviders(w http.ResponseWriter, r *http.Request) {
	all := s.registry.All()
	data := make([]map[string]any, len(all))
	for i, p := range all {
		data[i] = map[string]any{
			"id":        p.ID(),
			"name":      p.Name(),
			"available": p.Available(),
			"models":    p.Models(),
		}
	}
	jsonOK(w, data)
}

func (s *Server) handleListApps(w http.ResponseWriter, r *http.Request) {
	apps, err := s.store.ListApps()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if apps == nil {
		apps = []store.App{}
	}
	// Strip tokens from response and ensure providers is never null
	for i := range apps {
		apps[i].Token = apps[i].Token[:8] + "..."
		if apps[i].Providers == nil {
			apps[i].Providers = []string{}
		}
	}
	jsonOK(w, apps)
}

func (s *Server) handleRevokeApp(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.RevokeApp(id); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonOK(w, map[string]any{"status": "revoked"})
}

// --- Onboarding ---

func (s *Server) handleOnboardingStatus(w http.ResponseWriter, r *http.Request) {
	all := s.registry.All()
	providers := make([]map[string]any, len(all))
	for i, p := range all {
		providers[i] = map[string]any{
			"id":        p.ID(),
			"name":      p.Name(),
			"available": p.Available(),
		}
	}

	jsonOK(w, map[string]any{
		"setup_complete": s.cfg.SetupComplete,
		"providers":      providers,
	})
}

func (s *Server) handleOnboardingTestProvider(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProviderID string `json:"provider_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	p := s.registry.FindByID(req.ProviderID)
	if p == nil {
		jsonOK(w, map[string]any{
			"success": false,
			"error":   "unknown provider: " + req.ProviderID,
		})
		return
	}

	if !p.Available() {
		jsonOK(w, map[string]any{
			"success": false,
			"error":   p.Name() + " CLI not found in PATH. Please install it first.",
		})
		return
	}

	// Run a real test with a 30-second timeout
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	stream, err := p.Complete(ctx, &provider.ChatCompletionRequest{
		Messages: []provider.Message{
			{Role: "user", Content: "Respond with just the word 'ok'"},
		},
		Stream: true,
	})
	if err != nil {
		jsonOK(w, map[string]any{
			"success": false,
			"error":   "Failed to start test: " + err.Error(),
		})
		return
	}

	// Drain the stream and collect the response
	var content strings.Builder
	var lastErr error
	for chunk := range stream {
		if chunk.Error != nil {
			lastErr = chunk.Error
			break
		}
		content.WriteString(chunk.Content)
	}

	if lastErr != nil {
		msg := lastErr.Error()
		if strings.Contains(msg, "auth") || strings.Contains(msg, "credential") || strings.Contains(msg, "login") {
			msg += " — try running 'claude' in your terminal to authenticate"
		}
		jsonOK(w, map[string]any{
			"success": false,
			"error":   msg,
		})
		return
	}

	jsonOK(w, map[string]any{
		"success": true,
		"message": strings.TrimSpace(content.String()),
	})
}

func (s *Server) handleOnboardingComplete(w http.ResponseWriter, r *http.Request) {
	s.cfg.SetupComplete = true
	if err := s.cfg.Save(); err != nil {
		s.cfg.SetupComplete = false
		jsonError(w, http.StatusInternalServerError, "failed to save config: "+err.Error())
		return
	}
	jsonOK(w, map[string]any{"status": "ok"})
}

// --- Helpers ---

// providerAllowed checks if a provider ID is permitted by the allowed list.
// An empty/nil allowed list means unrestricted (all providers allowed).
func providerAllowed(allowed []string, providerID string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, a := range allowed {
		if a == providerID {
			return true
		}
	}
	return false
}

func generateShortID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

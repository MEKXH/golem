package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/version"
	"github.com/google/uuid"
)

type ChatProcessor interface {
	ProcessForChannel(ctx context.Context, channel, chatID, senderID, content string) (string, error)
}

type Server struct {
	cfg        config.GatewayConfig
	processor  ChatProcessor
	httpServer *http.Server
}

func New(cfg config.GatewayConfig, processor ChatProcessor) *Server {
	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		host = "0.0.0.0"
	}
	port := cfg.Port
	if port <= 0 {
		port = 18790
	}

	cfg.Host = host
	cfg.Port = port
	return &Server{
		cfg:       cfg,
		processor: processor,
	}
}

func (s *Server) Addr() string {
	return fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
}

func (s *Server) Start() error {
	mux := NewHandler(s.cfg.Token, s.processor)
	s.httpServer = &http.Server{
		Addr:              s.Addr(),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	slog.Info("gateway listening", "addr", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

func NewHandler(token string, processor ChatProcessor) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		requestID := getRequestID(r)
		if r.Method != http.MethodGet {
			writeError(w, requestID, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":     "ok",
			"request_id": requestID,
		})
	})
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		requestID := getRequestID(r)
		if r.Method != http.MethodGet {
			writeError(w, requestID, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"version":    version.Version,
			"request_id": requestID,
		})
	})
	mux.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		requestID := getRequestID(r)
		if r.Method != http.MethodPost {
			writeError(w, requestID, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		if strings.TrimSpace(token) != "" && !isAuthorized(r, token) {
			writeError(w, requestID, http.StatusUnauthorized, "unauthorized", "missing or invalid bearer token")
			return
		}

		var req struct {
			Message   string `json:"message"`
			SessionID string `json:"session_id"`
			SenderID  string `json:"sender_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, requestID, http.StatusBadRequest, "bad_request", "invalid json request")
			return
		}
		msg := strings.TrimSpace(req.Message)
		if msg == "" {
			writeError(w, requestID, http.StatusBadRequest, "bad_request", "message is required")
			return
		}
		sessionID := strings.TrimSpace(req.SessionID)
		if sessionID == "" {
			sessionID = "default"
		}
		senderID := strings.TrimSpace(req.SenderID)
		if senderID == "" {
			senderID = "api"
		}

		if processor == nil {
			writeError(w, requestID, http.StatusInternalServerError, "internal_error", "chat processor is not configured")
			return
		}

		resp, err := processor.ProcessForChannel(r.Context(), "gateway", sessionID, senderID, msg)
		if err != nil {
			slog.Error("gateway chat failed", "request_id", requestID, "session_id", sessionID, "error", err)
			writeError(w, requestID, http.StatusInternalServerError, "internal_error", "failed to process chat request")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"response":   resp,
			"session_id": sessionID,
			"request_id": requestID,
		})
	})
	return mux
}

func isAuthorized(r *http.Request, expected string) bool {
	got := strings.TrimSpace(r.Header.Get("Authorization"))
	if got == "" {
		return false
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(got, prefix) {
		return false
	}
	token := strings.TrimSpace(strings.TrimPrefix(got, prefix))
	return token == expected
}

func getRequestID(r *http.Request) string {
	rid := strings.TrimSpace(r.Header.Get("X-Request-ID"))
	if rid != "" {
		return rid
	}
	return uuid.NewString()
}

func writeError(w http.ResponseWriter, requestID string, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"code":       code,
		"message":    message,
		"request_id": requestID,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

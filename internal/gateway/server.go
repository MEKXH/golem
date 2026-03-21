// Package gateway 实现 Golem 的 API 网关，允许通过 HTTP 协议与 Agent 进行交互。
package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/version"
	"github.com/google/uuid"
)

// ChatProcessor 定义了网关处理聊天请求所需的接口。
type ChatProcessor interface {
	// ProcessForChannel 处理来自指定通道和聊天 ID 的消息。
	ProcessForChannel(ctx context.Context, channel, chatID, senderID, content string) (string, error)
}

// Server 表示网关服务器实例。
type Server struct {
	cfg        config.GatewayConfig // 网关配置
	processor  ChatProcessor        // 聊天处理器
	httpServer *http.Server         // 底层 HTTP 服务器
}

// New 创建并返回一个新的网关服务器实例。
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

// Addr 返回服务器监听的完整地址。
func (s *Server) Addr() string {
	return fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
}

// Start 启动网关服务器并开始监听请求。
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

// Shutdown 优雅地关闭网关服务器。
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

// NewHandler 创建并配置网关的路由处理器。
func NewHandler(token string, processor ChatProcessor) http.Handler {
	mux := http.NewServeMux()
	webUI, webUIErr := webUIFS()

	// 健康检查接口
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

	// 版本查询接口
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

	// 聊天交互接口
	mux.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		requestID := getRequestID(r)
		start := time.Now()
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
		slog.Info("gateway chat request",
			"request_id", requestID,
			"channel", "gateway",
			"session_id", sessionID,
			"sender_id", senderID,
		)

		if processor == nil {
			writeError(w, requestID, http.StatusInternalServerError, "internal_error", "chat processor is not configured")
			return
		}

		procCtx := bus.WithRequestID(r.Context(), requestID)
		resp, err := processor.ProcessForChannel(procCtx, "gateway", sessionID, senderID, msg)
		if err != nil {
			slog.Error("gateway chat failed", "request_id", requestID, "channel", "gateway", "session_id", sessionID, "error", err)
			writeError(w, requestID, http.StatusInternalServerError, "internal_error", "failed to process chat request")
			return
		}
		slog.Info("gateway chat completed",
			"request_id", requestID,
			"channel", "gateway",
			"session_id", sessionID,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		writeJSON(w, http.StatusOK, map[string]any{
			"response":   resp,
			"session_id": sessionID,
			"request_id": requestID,
		})
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveWebUI(w, r, webUI, webUIErr)
	})

	return mux
}

func serveWebUI(w http.ResponseWriter, r *http.Request, webUI fs.FS, webUIErr error) {
	if webUIErr != nil || webUI == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.NotFound(w, r)
		return
	}

	cleanPath := path.Clean(r.URL.Path)
	if cleanPath == "." {
		cleanPath = "/"
	}
	if cleanPath == "/" || !strings.Contains(path.Base(cleanPath), ".") {
		serveEmbeddedIndexHTML(w, webUI)
		return
	}
	http.FileServer(http.FS(webUI)).ServeHTTP(w, r)
}

func serveEmbeddedIndexHTML(w http.ResponseWriter, webUI fs.FS) {
	body, err := fs.ReadFile(webUI, "index.html")
	if err != nil {
		http.Error(w, "web ui index not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(body)
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

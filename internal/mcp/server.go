package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

type Server struct {
	MCP     *server.MCPServer
	http    *http.Server
	stream  *server.StreamableHTTPServer
	cfg     *Config
	limiter *RateLimiter
}

func NewServer(cfg *Config) *Server {
	mcpServer := server.NewMCPServer(
		"nombasub-mcp",
		"0.1.0",
		server.WithToolCapabilities(false),
		server.WithResourceCapabilities(false, false),
		server.WithPromptCapabilities(false),
		server.WithLogging(),
	)

	streamable := server.NewStreamableHTTPServer(
		mcpServer,
		server.WithHTTPContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			return WithAPIKeyFromHTTP(ctx, r)
		}),
	)

	limiter := NewRateLimiter(cfg.RequestsPerMinute)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.Handle("/mcp", AuthAndRateLimit(streamable, limiter))

	return &Server{
		MCP:     mcpServer,
		stream:  streamable,
		cfg:     cfg,
		limiter: limiter,
		http: &http.Server{
			Addr:              ":" + cfg.Port,
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "nombasub-mcp"})
}

func (s *Server) Start() error {
	if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

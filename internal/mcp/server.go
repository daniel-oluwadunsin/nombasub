package mcp

import (
	"context"
	"net/http"

	"github.com/mark3labs/mcp-go/server"
)

type Server struct {
	MCP  *server.MCPServer
	HTTP *server.StreamableHTTPServer
	cfg  *Config
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

	httpServer := server.NewStreamableHTTPServer(
		mcpServer,
		server.WithHTTPContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			return WithAPIKeyFromHTTP(ctx, r)
		}),
	)

	return &Server{MCP: mcpServer, HTTP: httpServer, cfg: cfg}
}

func (s *Server) Start() error {
	return s.HTTP.Start(":" + s.cfg.Port)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.HTTP.Shutdown(ctx)
}

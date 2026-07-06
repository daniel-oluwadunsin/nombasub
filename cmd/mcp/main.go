package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/mcp"
	"github.com/daniel-oluwadunsin/nombasub/internal/mcp/tools"
)

func main() {
	cfg := mcp.LoadConfig()
	engine := mcp.NewEngineClient(cfg.EngineURL)

	srv := mcp.NewServer(cfg)

	(&tools.Query{Engine: engine}).Register(srv.MCP)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("nombasub-mcp listening on :%s (engine %s)", cfg.Port, cfg.EngineURL)
		if err := srv.Start(); err != nil {
			log.Fatalf("mcp server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down mcp...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("mcp shutdown error: %v", err)
	}
}

package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/rafaelmgr12/streamgate/internal/config"
	"github.com/rafaelmgr12/streamgate/internal/handler"
	"github.com/rafaelmgr12/streamgate/internal/transport"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default configuration")
	}

	cfg := config.New()
	h := handler.NewHTTPHandler()

	httpTransport := transport.NewHTTPTransport(cfg.ListenAddr, h)
	// Start listener in goroutine for graceful shutdown later
	go func() {
		log.Printf("Listening on %s", cfg.ListenAddr)
		if err := httpTransport.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP Listen error: %v", err)
		}
	}()

	// Wait for system interrupt (Ctrl+C, Docker stop, etc)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gracefully...")
	if err := httpTransport.Close(); err != nil {
		log.Fatalf("Error during shutdown: %v", err)
	}
	log.Println("Shutdown complete.")

}

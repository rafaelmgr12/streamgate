package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/rafaelmgr12/streamgate/internal/config"
	"github.com/rafaelmgr12/streamgate/internal/handler"
	"github.com/rafaelmgr12/streamgate/internal/transport"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	var wg sync.WaitGroup
	transports := []transport.Transport{}

	for _, t := range cfg.Transports {
		log.Printf("Initializing transport of type '%s' on address '%s'", t.Type, t.Addr)
		var trans transport.Transport
		switch t.Type {
		case "http":
			h := handler.NewHTTPHandler()
			trans = transport.NewHTTPTransport(t.Addr, h)
		default:
			log.Printf("Unknown transport type: %s", t.Type)
			continue
		}
		transports = append(transports, trans)

		wg.Add(1)
		go func(tr transport.Transport) {
			defer wg.Done()
			log.Printf("Starting transport on %s", tr.Addr())
			if err := tr.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("Transport error: %v", err)
			}
		}(trans)
	}

	// Wait for system interrupt (Ctrl+C, Docker stop, etc)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gracefully...")
	for _, t := range transports {
		if err := t.Close(); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
	}

	wg.Wait()
	log.Println("Shutdown complete.")
}

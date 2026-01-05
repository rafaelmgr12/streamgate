package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/rafaelmgr12/streamgate/internal/balancer"
	"github.com/rafaelmgr12/streamgate/internal/config"
	"github.com/rafaelmgr12/streamgate/internal/handler"
	"github.com/rafaelmgr12/streamgate/internal/transport"
)

func main() {
	defaultConfigPath := os.Getenv("STREAMGATE_CONFIG")
	if defaultConfigPath == "" {
		defaultConfigPath = "config.yaml"
	}

	configPath := flag.String("config", defaultConfigPath, "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	tracker := balancer.NewHealthTracker(0, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startActiveChecks(ctx, cfg, tracker)

	var wg sync.WaitGroup
	transports := []transport.Transport{}

	for _, t := range cfg.Transports {
		log.Printf("Initializing transport of type '%s' on address '%s'", t.Type, t.Addr)
		var trans transport.Transport
		switch t.Type {
		case "http":
			h := handler.NewHTTPHandlerWithTracker(cfg, tracker)
			trans = transport.NewHTTPTransport(t.Addr, h)
		case "grpc":
			h := handler.NewGRPCHandler()
			trans = transport.NewGRPCTransport(t.Addr, h)
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
	cancel()

	log.Println("Shutting down gracefully...")
	for _, t := range transports {
		if err := t.Close(); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
	}

	wg.Wait()
	log.Println("Shutdown complete.")
}

func startActiveChecks(ctx context.Context, cfg *config.Config, tracker *balancer.HealthTracker) {
	if cfg == nil || tracker == nil {
		return
	}

	for _, svc := range cfg.Services {
		targets := make([]*url.URL, 0, len(svc.Backends))
		for _, raw := range svc.Backends {
			u, err := parseBackendURL(raw)
			if err != nil {
				log.Printf("invalid backend for service %q: %v", svc.Name, err)
				continue
			}
			targets = append(targets, u)
		}
		if len(targets) == 0 {
			continue
		}

		hc := mergeHealthCheck(cfg.HealthCheck, svc.HealthCheck)
		checker := balancer.NewActiveHealthChecker(tracker, targets, hc.Interval, hc.Timeout, hc.Path)
		checker.Start(ctx)

	}
}

func mergeHealthCheck(global, override config.HealthCheckConfig) config.HealthCheckConfig {
	merged := global
	if strings.TrimSpace(override.Path) != "" {
		merged.Path = override.Path
	}
	if override.Interval > 0 {
		merged.Interval = override.Interval
	}
	if override.Timeout > 0 {
		merged.Timeout = override.Timeout
	}
	return merged
}

func parseBackendURL(raw string) (*url.URL, error) {
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	return url.Parse(raw)
}

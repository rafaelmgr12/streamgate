package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rafaelmgr12/streamgate/internal/config"
)

func TestLoad_Success_NoEnvOverrides(t *testing.T) {

	yaml := `
transports:
  - type: http
    addr: ":8080"
services:
  - name: api
    path_prefix: "/api"
    backends:
      - "http://localhost:9001"
      - "localhost:9002"
`

	path := writeTempYAML(t, yaml)

	unsetEnv(t, "STREAMGATE_HTTP_ADDR")
	unsetEnv(t, "STREAMGATE_GRPC_ADDR")

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load() err = %v", err)
	}
	if cfg == nil {
		t.Fatalf("Load() cfg is nil")
	}

	if len(cfg.Transports) != 1 {
		t.Fatalf("expected 1 transport, got %d", len(cfg.Transports))
	}
	if cfg.Transports[0].Type != "http" {
		t.Fatalf("expected transport type http, got %q", cfg.Transports[0].Type)
	}
	if cfg.Transports[0].Addr != ":8080" {
		t.Fatalf("expected addr :8080, got %q", cfg.Transports[0].Addr)
	}

	if len(cfg.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(cfg.Services))
	}
	if cfg.Services[0].Name != "api" {
		t.Fatalf("expected service name api, got %q", cfg.Services[0].Name)
	}
	if cfg.Services[0].PathPrefix != "/api" {
		t.Fatalf("expected path_prefix /api, got %q", cfg.Services[0].PathPrefix)
	}
	if len(cfg.Services[0].Backends) != 2 {
		t.Fatalf("expected 2 backends, got %d", len(cfg.Services[0].Backends))
	}
}

func TestLoad_EnvOverride_ExistingTransport(t *testing.T) {

	yaml := `
transports:
  - type: http
    addr: ":8080"
services:
  - name: api
    path_prefix: "/api"
    backends: ["localhost:9001"]
`
	path := writeTempYAML(t, yaml)

	setEnv(t, "STREAMGATE_HTTP_ADDR", ":9999")
	unsetEnv(t, "STREAMGATE_GRPC_ADDR")

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load() err = %v", err)
	}

	if got := findTransportAddr(cfg, "http"); got != ":9999" {
		t.Fatalf("expected http addr overridden to :9999, got %q", got)
	}
}

func TestLoad_EnvOverride_AddsTransportIfMissing(t *testing.T) {

	yaml := `
transports:
  - type: grpc
    addr: ":50051"
services:
  - name: api
    path_prefix: "/api"
    backends: ["localhost:9001"]
`
	path := writeTempYAML(t, yaml)

	setEnv(t, "STREAMGATE_HTTP_ADDR", ":8081")
	unsetEnv(t, "STREAMGATE_GRPC_ADDR")

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load() err = %v", err)
	}

	if got := findTransportAddr(cfg, "http"); got != ":8081" {
		t.Fatalf("expected http transport added with addr :8081, got %q", got)
	}
	if got := findTransportAddr(cfg, "grpc"); got != ":50051" {
		t.Fatalf("expected grpc addr preserved as :50051, got %q", got)
	}
}

func TestLoad_FileNotFound(t *testing.T) {

	unsetEnv(t, "STREAMGATE_HTTP_ADDR")
	unsetEnv(t, "STREAMGATE_GRPC_ADDR")

	_, err := config.Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {

	unsetEnv(t, "STREAMGATE_HTTP_ADDR")
	unsetEnv(t, "STREAMGATE_GRPC_ADDR")

	path := writeTempYAML(t, `transports: [this is not: valid`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestLoad_HealthCheckConfig(t *testing.T) {
	yaml := `
transports:
  - type: http
    addr: ":8080"
health_check:
  path: "/healthz"
  interval: "10s"
  timeout: "2s"
services:
  - name: api
    path_prefix: "/api"
    backends:
      - "localhost:9001"
    health_check:
      path: "/status"
      interval: "5s"
`

	path := writeTempYAML(t, yaml)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load() err = %v", err)
	}

	if cfg.HealthCheck.Path != "/healthz" {
		t.Fatalf("expected global health_check path /healthz, got %q", cfg.HealthCheck.Path)
	}
	if cfg.HealthCheck.Interval != 10*time.Second {
		t.Fatalf("expected global interval 10s, got %s", cfg.HealthCheck.Interval)
	}
	if cfg.HealthCheck.Timeout != 2*time.Second {
		t.Fatalf("expected global timeout 2s, got %s", cfg.HealthCheck.Timeout)
	}

	if cfg.Services[0].HealthCheck.Path != "/status" {
		t.Fatalf("expected service health_check path /status, got %q", cfg.Services[0].HealthCheck.Path)
	}
	if cfg.Services[0].HealthCheck.Interval != 5*time.Second {
		t.Fatalf("expected service interval 5s, got %s", cfg.Services[0].HealthCheck.Interval)
	}
	if cfg.Services[0].HealthCheck.Timeout != 0 {
		t.Fatalf("expected service timeout default 0, got %s", cfg.Services[0].HealthCheck.Timeout)
	}
}

func TestValidate_NilConfig(t *testing.T) {

	var cfg *config.Config
	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "config is nil") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_RequiresAtLeastOneTransport(t *testing.T) {

	cfg := &config.Config{}
	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "at least one transport is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_TransportTypeRequired(t *testing.T) {

	cfg := &config.Config{
		Transports: []config.TransportConfig{
			{Type: "  ", Addr: ":8080"},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "transport type is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_TransportAddrRequired(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Transports: []config.TransportConfig{
			{Type: "http", Addr: "   "},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), `transport "http" has empty addr`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_ServiceNameRequired(t *testing.T) {

	cfg := &config.Config{
		Transports: []config.TransportConfig{{Type: "http", Addr: ":8080"}},
		Services: []config.ServiceConfig{
			{Name: " ", PathPrefix: "/api", Backends: []string{"localhost:9001"}},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "service name is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_ServicePathPrefixMustStartWithSlash(t *testing.T) {

	cfg := &config.Config{
		Transports: []config.TransportConfig{{Type: "http", Addr: ":8080"}},
		Services: []config.ServiceConfig{
			{Name: "api", PathPrefix: "api", Backends: []string{"localhost:9001"}},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "path_prefix must start with") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_ServicePathPrefixWhitespace_Fails(t *testing.T) {

	cfg := &config.Config{
		Transports: []config.TransportConfig{{Type: "http", Addr: ":8080"}},
		Services: []config.ServiceConfig{
			{Name: "api", PathPrefix: "gr", Backends: []string{"localhost:9001"}},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	// Current behavior: treated as "does not start with '/'".
	if !strings.Contains(err.Error(), "path_prefix must start with") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_ServiceMustHaveAtLeastOneBackend(t *testing.T) {

	cfg := &config.Config{
		Transports: []config.TransportConfig{{Type: "http", Addr: ":8080"}},
		Services: []config.ServiceConfig{
			{Name: "api", PathPrefix: "/api", Backends: nil},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "must have at least one backend") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_HealthCheckPathMustStartWithSlash_Global(t *testing.T) {
	cfg := &config.Config{
		Transports: []config.TransportConfig{{Type: "http", Addr: ":8080"}},
		HealthCheck: config.HealthCheckConfig{
			Path: "healthz",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "path must start with") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_HealthCheckPathMustStartWithSlash_Service(t *testing.T) {
	cfg := &config.Config{
		Transports: []config.TransportConfig{{Type: "http", Addr: ":8080"}},
		Services: []config.ServiceConfig{
			{
				Name:       "api",
				PathPrefix: "/api",
				Backends:   []string{"localhost:9001"},
				HealthCheck: config.HealthCheckConfig{
					Path: "healthz",
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "path must start with") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_InvalidBackendURL(t *testing.T) {

	cfg := &config.Config{
		Transports: []config.TransportConfig{{Type: "http", Addr: ":8080"}},
		Services: []config.ServiceConfig{
			{Name: "api", PathPrefix: "/api", Backends: []string{"http://"}},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "has invalid backend") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_BackendWithoutScheme_IsAccepted(t *testing.T) {

	cfg := &config.Config{
		Transports: []config.TransportConfig{{Type: "http", Addr: ":8080"}},
		Services: []config.ServiceConfig{
			{Name: "api", PathPrefix: "/api", Backends: []string{"localhost:9001"}},
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidate_BackendWithScheme_IsAccepted(t *testing.T) {

	cfg := &config.Config{
		Transports: []config.TransportConfig{{Type: "http", Addr: ":8080"}},
		Services: []config.ServiceConfig{
			{Name: "api", PathPrefix: "/api", Backends: []string{"http://localhost:9001"}},
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp yaml: %v", err)
	}
	return path
}

func setEnv(t *testing.T, k, v string) {
	t.Helper()
	t.Setenv(k, v)
}

func unsetEnv(t *testing.T, k string) {
	t.Helper()
	t.Setenv(k, "")
}

func findTransportAddr(cfg *config.Config, transportType string) string {
	for _, tr := range cfg.Transports {
		if tr.Type == transportType {
			return tr.Addr
		}
	}
	return ""
}

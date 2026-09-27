package otlp

import "time"

// Protocol selects the OTLP transport.
type Protocol string

const (
	ProtocolGRPC Protocol = "grpc"
	ProtocolHTTP Protocol = "http"
)

// Config describes an OTLP exporter setup for traces, metrics, and logs.
//
// Only ServiceName and Endpoint are required when Enabled is true.
// Zero values for enum/numeric fields fall back to sensible defaults
// (see applyDefaults).
type Config struct {
	// Enabled toggles Setup. When false, Setup returns a no-op shutdown.
	Enabled bool

	// ServiceName populates the `service.name` resource attribute.
	ServiceName string

	// ServiceVersion populates `service.version` (optional).
	ServiceVersion string

	// Environment populates `deployment.environment.name` (optional).
	Environment string

	// Endpoint of the OTLP receiver.
	//   gRPC: "host:4317" (no scheme)
	//   HTTP: "host:4318" (no scheme)
	Endpoint string

	// Protocol: ProtocolGRPC (default) or ProtocolHTTP.
	Protocol Protocol

	// Insecure disables TLS. Use for local dev or in-cluster collectors on plaintext.
	Insecure bool

	// Headers are sent with every OTLP request (auth tokens, tenant IDs, etc.).
	Headers map[string]string

	// SampleRatio for traces: 0.0 (drop all) → 1.0 (keep all). Default 1.0.
	SampleRatio float64

	// ExportTimeout bounds each batch export operation. Default 10s.
	ExportTimeout time.Duration

	// Signal toggles. At least one should be true for Setup to do anything useful.
	Traces  bool
	Metrics bool
	Logs    bool

	// ResourceAttrs adds custom resource attributes (merged with defaults).
	ResourceAttrs map[string]string
}

func (c *Config) applyDefaults() {
	if c.Protocol == "" {
		c.Protocol = ProtocolGRPC
	}
	if c.SampleRatio <= 0 {
		c.SampleRatio = 1.0
	}
	if c.ExportTimeout <= 0 {
		c.ExportTimeout = 10 * time.Second
	}
}

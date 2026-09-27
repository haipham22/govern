package otlp

import (
	"strings"
)

// FlatConfig reads the OpenTelemetry environment variables by their specified
// names. These are not a local convention: OTEL_EXPORTER_OTLP_ENDPOINT and
// OTEL_SERVICE_NAME come from the OpenTelemetry specification, so collectors,
// sidecars and other SDKs already expect exactly these.
//
//	OTEL_EXPORTER_OTLP_ENDPOINT · OTEL_SERVICE_NAME
//	OTEL_EXPORTER_OTLP_PROTOCOL · OTEL_EXPORTER_OTLP_INSECURE
//	OTEL_SERVICE_VERSION · OTEL_TRACES_SAMPLER_ARG
//
// Embed with `mapstructure:",squash"` and call Apply:
//
//	type ServiceConfig struct {
//	    FlatOTel otlp.FlatConfig `mapstructure:",squash"`
//	    OTel     otlp.Config     `mapstructure:"otel"` // signal toggles, headers
//	}
//
//	cfg.OTel = cfg.FlatOTel.Apply(cfg.OTel)
//
// Config itself carries no mapstructure tags and never has: before this it
// could only be built in code, so a service could not be pointed at a
// collector without a rebuild.
type FlatConfig struct {
	// Endpoint is host:port with no scheme — gRPC 4317, HTTP 4318. Setting it
	// is what turns exporting on: a collector address is the one thing that
	// cannot be guessed, and requiring a second flag alongside it only creates
	// a way to set one and forget the other.
	Endpoint string `yaml:"otel_exporter_otlp_endpoint" mapstructure:"otel_exporter_otlp_endpoint"`

	ServiceName    string `yaml:"otel_service_name" mapstructure:"otel_service_name"`
	ServiceVersion string `yaml:"otel_service_version" mapstructure:"otel_service_version"`

	// Protocol is "grpc" (default) or "http".
	Protocol string `yaml:"otel_exporter_otlp_protocol" mapstructure:"otel_exporter_otlp_protocol"`

	// Insecure disables TLS — an in-cluster collector on plaintext.
	Insecure bool `yaml:"otel_exporter_otlp_insecure" mapstructure:"otel_exporter_otlp_insecure"`

	// SampleRatio is the OTEL_TRACES_SAMPLER_ARG value: 0.0 drops every trace,
	// 1.0 keeps them all.
	SampleRatio float64 `yaml:"otel_traces_sampler_arg" mapstructure:"otel_traces_sampler_arg"`
}

// Apply copies the flat fields onto base and returns it. Signal toggles,
// headers and resource attributes are left as the caller set them: which
// signals a service exports is a code decision, not a deployment one.
//
// Enabled is turned on when an endpoint arrived, and left alone otherwise, so
// a service that wires telemetry up itself is not switched off by an absent
// variable.
func (c FlatConfig) Apply(base Config) Config {
	if c.Endpoint == "" {
		return base
	}

	base.Enabled = true
	base.Endpoint = c.Endpoint
	if c.ServiceName != "" {
		base.ServiceName = c.ServiceName
	}
	if c.ServiceVersion != "" {
		base.ServiceVersion = c.ServiceVersion
	}
	if protocol := strings.ToLower(strings.TrimSpace(c.Protocol)); protocol != "" {
		base.Protocol = Protocol(protocol)
	}
	if c.Insecure {
		base.Insecure = true
	}
	if c.SampleRatio > 0 {
		base.SampleRatio = c.SampleRatio
	}
	return base
}

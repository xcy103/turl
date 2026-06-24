package configs

// Default values for the observability admin server.
const (
	// DefaultAdminListen is the default bind address of the admin server.
	DefaultAdminListen = "0.0.0.0"
	// DefaultAdminPort is the default port of the admin server.
	DefaultAdminPort = 9090
)

// ObservabilityConfig configures the admin server that exposes health checks,
// metrics, and profiling endpoints on a port separate from the public API.
//
// Tracing/metrics tunables (OTLP endpoint, sampling) are added in later phases.
type ObservabilityConfig struct {
	// Enable toggles the admin/observability server on or off.
	Enable bool `json:"enable" yaml:"enable" mapstructure:"enable"`
	// AdminListen is the bind address of the admin server.
	AdminListen string `validate:"omitempty,ip_addr|hostname" json:"admin_listen" yaml:"admin_listen" mapstructure:"admin_listen"`
	// AdminPort is the port the admin server listens on. It must differ from the
	// public API port so internal endpoints are not exposed to end users.
	AdminPort int `validate:"omitempty,min=1,max=65535" json:"admin_port" yaml:"admin_port" mapstructure:"admin_port"`
}

// WithDefaults fills unset fields with their defaults and returns the config,
// so callers can rely on a usable address even from a sparse YAML block.
func (c *ObservabilityConfig) WithDefaults() *ObservabilityConfig {
	if c.AdminListen == "" {
		c.AdminListen = DefaultAdminListen
	}

	if c.AdminPort == 0 {
		c.AdminPort = DefaultAdminPort
	}

	return c
}

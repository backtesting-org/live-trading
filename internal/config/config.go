package config

import (
	"fmt"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Plugin   PluginConfig   `mapstructure:"plugin"`
	Logging  LoggingConfig  `mapstructure:"logging"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	ReadTimeout     int    `mapstructure:"read_timeout"`
	WriteTimeout    int    `mapstructure:"write_timeout"`
	MaxUploadSize   int64  `mapstructure:"max_upload_size"` // in bytes
	CORSAllowOrigin string `mapstructure:"cors_allow_origin"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	ConnectionString string `mapstructure:"connection_string"`
	MaxOpenConns     int    `mapstructure:"max_open_conns"`
	MaxIdleConns     int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime  int    `mapstructure:"conn_max_lifetime"` // in minutes
}

// PluginConfig represents plugin system configuration
type PluginConfig struct {
	Directory       string `mapstructure:"directory"`
	MaxPluginSize   int64  `mapstructure:"max_plugin_size"` // in bytes
	AllowedVersions string `mapstructure:"allowed_versions"` // Go version compatibility
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level      string `mapstructure:"level"` // debug, info, warn, error
	Format     string `mapstructure:"format"` // json, console
	OutputPath string `mapstructure:"output_path"`
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Load .env file if it exists (ignore errors if file doesn't exist)
	_ = godotenv.Load()

	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Read from environment variables
	v.SetEnvPrefix("LIVE_TRADING")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8081)
	v.SetDefault("server.read_timeout", 30)
	v.SetDefault("server.write_timeout", 30)
	v.SetDefault("server.max_upload_size", 100*1024*1024) // 100MB
	v.SetDefault("server.cors_allow_origin", "*")

	// Database defaults
	v.SetDefault("database.connection_string", "")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", 5)

	// Plugin defaults
	v.SetDefault("plugin.directory", "./plugins")
	v.SetDefault("plugin.max_plugin_size", 50*1024*1024) // 50MB
	v.SetDefault("plugin.allowed_versions", "1.24")

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.output_path", "stdout")
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Validate server config
	if config.Server.Port < 1 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	if config.Server.MaxUploadSize < 1024*1024 { // Minimum 1MB
		return fmt.Errorf("max_upload_size too small: %d", config.Server.MaxUploadSize)
	}

	// Validate database config
	if config.Database.ConnectionString == "" {
		return fmt.Errorf("database connection string is required")
	}

	// Validate plugin config
	if config.Plugin.Directory == "" {
		return fmt.Errorf("plugin directory is required")
	}

	// Validate logging config
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[config.Logging.Level] {
		return fmt.Errorf("invalid logging level: %s", config.Logging.Level)
	}

	return nil
}

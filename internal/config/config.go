package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	v *viper.Viper
}

// New creates a new configuration instance
func New() (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("/etc/llm-spam-filter/")
	v.AddConfigPath("$HOME/.llm-spam-filter")
	v.AddConfigPath("./configs")
	v.AddConfigPath(".")

	// Set defaults
	setDefaults(v)

	// Environment variables
	v.AutomaticEnv()
	v.SetEnvPrefix("SPAM_FILTER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, using defaults
	}

	return &Config{v: v}, nil
}

// NewFromViper creates a new configuration instance from an existing Viper instance
func NewFromViper(v *viper.Viper) *Config {
	return &Config{v: v}
}

// NewEmptyViper creates a new Viper instance with defaults
func NewEmptyViper() *viper.Viper {
	v := viper.New()
	setDefaults(v)
	return v
}

// setDefaults sets the default configuration values
func setDefaults(v *viper.Viper) {
	// LLM provider defaults
	v.SetDefault("llm.provider", "bedrock")
	
	// Server defaults
	v.SetDefault("server.filter_type", "postfix")
	v.SetDefault("server.listen_address", "0.0.0.0:10025")
	v.SetDefault("server.block_spam", false)
	v.SetDefault("server.headers.spam", "X-Spam-Status")
	v.SetDefault("server.headers.score", "X-Spam-Score")
	v.SetDefault("server.headers.reason", "X-Spam-Reason")
	
	// Bedrock defaults
	v.SetDefault("bedrock.region", "us-east-1")
	v.SetDefault("bedrock.model_id", "anthropic.claude-v2")
	v.SetDefault("bedrock.max_tokens", 1000)
	v.SetDefault("bedrock.temperature", 0.1)
	v.SetDefault("bedrock.top_p", 0.9)
	v.SetDefault("bedrock.max_body_size", 4096)
	
	// Gemini defaults
	v.SetDefault("gemini.api_key", "")
	v.SetDefault("gemini.model_name", "gemini-pro")
	v.SetDefault("gemini.max_tokens", 1000)
	v.SetDefault("gemini.temperature", 0.1)
	v.SetDefault("gemini.top_p", 0.9)
	v.SetDefault("gemini.max_body_size", 4096)
	
	// OpenAI defaults
	v.SetDefault("openai.api_key", "")
	v.SetDefault("openai.model_name", "gpt-4")
	v.SetDefault("openai.max_tokens", 1000)
	v.SetDefault("openai.temperature", 0.1)
	v.SetDefault("openai.top_p", 0.9)
	v.SetDefault("openai.max_body_size", 4096)
	
	// Spam defaults
	v.SetDefault("spam.threshold", 0.7)
	v.SetDefault("spam.whitelisted_domains", []string{})
	
	// Cache defaults
	v.SetDefault("cache.type", "memory")
	v.SetDefault("cache.enabled", true)
	v.SetDefault("cache.ttl", "24h")
	v.SetDefault("cache.cleanup_frequency", "1h")
	v.SetDefault("cache.sqlite_path", "/data/spam_cache.db")
	v.SetDefault("cache.mysql_dsn", "user:password@tcp(localhost:3306)/spam_filter")
	
	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
}

// GetString gets a string value from the configuration
func (c *Config) GetString(key string) string {
	return c.v.GetString(key)
}

// GetInt gets an integer value from the configuration
func (c *Config) GetInt(key string) int {
	return c.v.GetInt(key)
}

// GetFloat64 gets a float64 value from the configuration
func (c *Config) GetFloat64(key string) float64 {
	return c.v.GetFloat64(key)
}

// GetBool gets a boolean value from the configuration
func (c *Config) GetBool(key string) bool {
	return c.v.GetBool(key)
}

// GetStringSlice gets a string slice value from the configuration
func (c *Config) GetStringSlice(key string) []string {
	return c.v.GetStringSlice(key)
}

// GetDuration gets a duration value from the configuration
func (c *Config) GetDuration(key string) (time.Duration, error) {
	return time.ParseDuration(c.GetString(key))
}

// GetViper returns the underlying Viper instance
func (c *Config) GetViper() *viper.Viper {
	return c.v
}

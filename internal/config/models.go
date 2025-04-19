package config

// LLMConfig represents the configuration for the LLM provider
type LLMConfig struct {
	Provider string
}

// BedrockConfig represents the configuration for Amazon Bedrock
type BedrockConfig struct {
	Region      string
	ModelID     string
	MaxTokens   int
	Temperature float32
	TopP        float32
	MaxBodySize int
}

// GeminiConfig represents the configuration for Google Gemini
type GeminiConfig struct {
	APIKey      string
	ModelName   string
	MaxTokens   int
	Temperature float32
	TopP        float32
	MaxBodySize int
}

// OpenAIConfig represents the configuration for OpenAI
type OpenAIConfig struct {
	APIKey      string
	ModelName   string
	MaxTokens   int
	Temperature float32
	TopP        float32
	MaxBodySize int
}

// GetLLM returns the LLM configuration
func (c *Config) GetLLM() LLMConfig {
	return LLMConfig{
		Provider: c.GetString("llm.provider"),
	}
}

// GetBedrock returns the Bedrock configuration
func (c *Config) GetBedrock() BedrockConfig {
	return BedrockConfig{
		Region:      c.GetString("bedrock.region"),
		ModelID:     c.GetString("bedrock.model_id"),
		MaxTokens:   c.GetInt("bedrock.max_tokens"),
		Temperature: float32(c.GetFloat64("bedrock.temperature")),
		TopP:        float32(c.GetFloat64("bedrock.top_p")),
		MaxBodySize: c.GetInt("bedrock.max_body_size"),
	}
}

// GetGemini returns the Gemini configuration
func (c *Config) GetGemini() GeminiConfig {
	return GeminiConfig{
		APIKey:      c.GetString("gemini.api_key"),
		ModelName:   c.GetString("gemini.model_name"),
		MaxTokens:   c.GetInt("gemini.max_tokens"),
		Temperature: float32(c.GetFloat64("gemini.temperature")),
		TopP:        float32(c.GetFloat64("gemini.top_p")),
		MaxBodySize: c.GetInt("gemini.max_body_size"),
	}
}

// GetOpenAI returns the OpenAI configuration
func (c *Config) GetOpenAI() OpenAIConfig {
	return OpenAIConfig{
		APIKey:      c.GetString("openai.api_key"),
		ModelName:   c.GetString("openai.model_name"),
		MaxTokens:   c.GetInt("openai.max_tokens"),
		Temperature: float32(c.GetFloat64("openai.temperature")),
		TopP:        float32(c.GetFloat64("openai.top_p")),
		MaxBodySize: c.GetInt("openai.max_body_size"),
	}
}

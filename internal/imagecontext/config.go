package imagecontext

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrNotConfigured = errors.New("visual context provider is not configured")

type Config struct {
	APIKey  string  `json:"apiKey"`
	Model   string  `json:"model"`
	BaseURL string  `json:"baseUrl"`
	OpenAI  *Config `json:"openai,omitempty"`
}

func LoadConfig(modelOverride string) (Config, error) {
	return LoadProviderConfig("openrouter", modelOverride)
}

func LoadProviderConfig(provider, modelOverride string) (Config, error) {
	path := os.Getenv("PI_SPECTACLES_CONFIG")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Config{}, err
		}
		path = filepath.Join(home, ".pi", "agent", "pi-spectacles.json")
	}
	config := Config{}
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("read pi-spectacles config: %w", err)
	}
	if err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return Config{}, fmt.Errorf("invalid pi-spectacles config at %s: %w", path, err)
		}
	}
	if provider == "openai" {
		if config.OpenAI != nil {
			config = *config.OpenAI
		}
		if value := os.Getenv("OPENAI_API_KEY"); value != "" {
			config.APIKey = value
		}
		if value := os.Getenv("OPENAI_VISION_MODEL"); value != "" {
			config.Model = value
		}
		if value := os.Getenv("OPENAI_BASE_URL"); value != "" {
			config.BaseURL = value
		}
		if modelOverride != "" {
			config.Model = modelOverride
		}
		if config.Model == "" {
			config.Model = DefaultOpenAIModel
		}
		if config.BaseURL == "" {
			config.BaseURL = DefaultOpenAIBaseURL
		}
		if config.APIKey == "" {
			return Config{}, fmt.Errorf(
				"%w: configure openai in %s or OPENAI_API_KEY",
				ErrNotConfigured,
				path,
			)
		}
		return config, nil
	}
	if provider != "openrouter" {
		return Config{}, fmt.Errorf("unsupported visual context provider %q", provider)
	}
	if value := os.Getenv("OPENROUTER_API_KEY"); value != "" {
		config.APIKey = value
	}
	if value := os.Getenv("OPENROUTER_MEDIA_MODEL"); value != "" {
		config.Model = value
	}
	if value := os.Getenv("OPENROUTER_BASE_URL"); value != "" {
		config.BaseURL = value
	}
	if modelOverride != "" {
		config.Model = modelOverride
	}
	if config.Model == "" {
		config.Model = DefaultModel
	}
	if config.BaseURL == "" {
		config.BaseURL = DefaultBaseURL
	}
	if config.APIKey == "" {
		return Config{}, fmt.Errorf(
			"%w: configure %s or OPENROUTER_API_KEY",
			ErrNotConfigured,
			path,
		)
	}
	return config, nil
}

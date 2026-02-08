package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	DefaultPort    = 21110
	DefaultDataDir = ".plug-my-ai"
	ConfigFileName = "config.json"
)

type Config struct {
	Port          int              `json:"port"`
	AdminToken    string           `json:"admin_token"`
	DataDir       string           `json:"data_dir"`
	Providers     []ProviderConfig `json:"providers"`
	SetupComplete bool             `json:"setup_complete"`
}

type ProviderConfig struct {
	Type    string          `json:"type"` // "claude-code", "openai", "anthropic", "ollama"
	Name    string          `json:"name"`
	Enabled bool            `json:"enabled"`
	Config  json.RawMessage `json:"config,omitempty"`
}

func DefaultConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return DefaultDataDir
	}
	return filepath.Join(home, DefaultDataDir)
}

func Load(configDir string) (*Config, error) {
	path := filepath.Join(configDir, ConfigFileName)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return createDefault(configDir)
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	cfg.DataDir = configDir
	return &cfg, nil
}

func (c *Config) Save() error {
	if err := os.MkdirAll(c.DataDir, 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	path := filepath.Join(c.DataDir, ConfigFileName)
	return os.WriteFile(path, data, 0600)
}

func createDefault(configDir string) (*Config, error) {
	token, err := generateToken("pma_admin_")
	if err != nil {
		return nil, fmt.Errorf("generating admin token: %w", err)
	}

	cfg := &Config{
		Port:       DefaultPort,
		AdminToken: token,
		DataDir:    configDir,
		Providers: []ProviderConfig{
			{
				Type:    "claude-code",
				Name:    "Claude Code",
				Enabled: true,
			},
			{
				Type:    "codex",
				Name:    "Codex",
				Enabled: false,
			},
			{
				Type:    "openai-compat",
				Name:    "OpenAI Compatible",
				Enabled: false,
			},
		},
	}

	if err := cfg.Save(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func generateToken(prefix string) (string, error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(bytes), nil
}

// GenerateAppToken creates a new token for a paired app.
func GenerateAppToken() (string, error) {
	return generateToken("pma_")
}

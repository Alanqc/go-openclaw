package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration (simplified from OpenClaw TypeScript).
type Config struct {
	Agents   AgentsConfig   `yaml:"agents"`
	Bindings []AgentBinding `yaml:"bindings"`
	Session  SessionConfig  `yaml:"session"`
}

// AgentsConfig holds agent defaults.
type AgentsConfig struct {
	Defaults AgentsDefaults `yaml:"defaults"`
	List     []AgentEntry   `yaml:"list"`
}

// AgentsDefaults holds default agent settings.
type AgentsDefaults struct {
	DefaultModel string `yaml:"default_model"`
	// LLMProvider 指定使用的 LLM 插件 id（如 "kimi"），为空则不调用大模型。
	LLMProvider string `yaml:"llm_provider"`
}

// AgentEntry represents a single agent in the list.
type AgentEntry struct {
	ID string `yaml:"id"`
}

// AgentBinding binds a channel/peer/guild to an agent.
type AgentBinding struct {
	AgentID string        `yaml:"agent_id"`
	Match   BindingMatch  `yaml:"match"`
}

// BindingMatch defines matching criteria.
type BindingMatch struct {
	Channel   string `yaml:"channel"`
	AccountID string `yaml:"account_id"`
	Peer      *struct {
		Kind string `yaml:"kind"`
		ID   string `yaml:"id"`
	} `yaml:"peer,omitempty"`
	GuildID string `yaml:"guild_id,omitempty"`
	TeamID  string `yaml:"team_id,omitempty"`
}

// SessionConfig holds session settings.
type SessionConfig struct {
	DMScope       string            `yaml:"dm_scope"`
	IdentityLinks map[string][]string `yaml:"identity_links,omitempty"`
}

// Load reads config from path (YAML). Falls back to empty config on error.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Default(), nil
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Default returns a minimal default config.
func Default() *Config {
	return &Config{
		Agents: AgentsConfig{
			Defaults: AgentsDefaults{
				DefaultModel: "kimi-k2-turbo-preview",
				LLMProvider:  "kimi",
			},
			List: []AgentEntry{{ID: "main"}},
		},
		Bindings: nil,
		Session: SessionConfig{
			DMScope: "main",
		},
	}
}

// ResolveConfigPath returns path to config file (openclaw.yaml in home or cwd).
func ResolveConfigPath() string {
	if p := os.Getenv("OPENCLAW_CONFIG"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		return filepath.Join(home, ".openclaw", "openclaw.yaml")
	}
	return "openclaw.yaml"
}

// ResolveSecretsPath returns path to secrets file (goopenclaw.secrets).
// Order: OPENCLAW_SECRETS env > ./goopenclaw.secrets (cwd) > ~/.openclaw/goopenclaw.secrets.
func ResolveSecretsPath() string {
	if p := os.Getenv("OPENCLAW_SECRETS"); p != "" {
		return p
	}
	if cwd, _ := os.Getwd(); cwd != "" {
		return filepath.Join(cwd, "goopenclaw.secrets")
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		return filepath.Join(home, ".openclaw", "goopenclaw.secrets")
	}
	return "goopenclaw.secrets"
}

// LoadSecrets reads path as KEY=VALUE lines (optional # comments) and sets each as os environment.
// Used so DISCORD_TOKEN, MOONSHOT_API_KEY etc. can be read from file without committing secrets.
func LoadSecrets(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if key == "" {
			continue
		}
		os.Setenv(key, val)
	}
	return sc.Err()
}

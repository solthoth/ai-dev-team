package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ListenAddr string
	OllamaURL  string
	PromptsDir string
	TimeoutSec int
	AgentToken string

	ModelPlanner  string
	ModelPlatform string
	ModelReviewer string
	ModelDocs     string
	ModelRouter   string
}

func Load() Config {
	return Config{
		ListenAddr:    env("AGENT_LISTEN", "0.0.0.0:4000"),
		OllamaURL:     env("OLLAMA_URL", "http://127.0.0.1:11434"),
		PromptsDir:    env("PROMPTS_DIR", "./prompts"),
		TimeoutSec:    envInt("AGENT_TIMEOUT_SEC", 180),
		AgentToken:    env("AGENT_TOKEN", ""),
		ModelPlanner:  env("MODEL_PLANNER", "llama3.1:8b"),
		ModelPlatform: env("MODEL_PLATFORM", "qwen2.5-coder:7b"),
		ModelReviewer: env("MODEL_REVIEWER", "llama3.1:8b"),
		ModelDocs:     env("MODEL_DOCS", "llama3.1:8b"),
		ModelRouter:   env("MODEL_ROUTER", "llama3.1:8b"),
	}
}

func (c Config) ModelForEnvKey(key string) string {
	switch key {
	case "MODEL_PLANNER":
		return c.ModelPlanner
	case "MODEL_PLATFORM":
		return c.ModelPlatform
	case "MODEL_REVIEWER":
		return c.ModelReviewer
	case "MODEL_DOCS":
		return c.ModelDocs
	case "MODEL_ROUTER":
		return c.ModelRouter
	default:
		return c.ModelPlanner
	}
}

func env(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func envInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}

	return n
}
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/solthoth/ai-dev-team/internal/agents"
	"github.com/solthoth/ai-dev-team/internal/config"
	"github.com/solthoth/ai-dev-team/internal/ollama"
	"github.com/solthoth/ai-dev-team/internal/prompts"
)

func main() {
	cfg := config.Load()
	registry := agents.Registry()
	ollamaClient := ollama.New(cfg.OllamaURL, time.Duration(cfg.TimeoutSec)*time.Second)

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"time": time.Now().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("/agents", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		names := make([]string, 0, len(registry))
		for name := range registry {
			names = append(names, name)
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"agents": names,
		})
	})

	mux.HandleFunc("/agents/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		if cfg.AgentToken != "" {
			if r.Header.Get("X-Agent-Token") != cfg.AgentToken {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
		}

		name := strings.TrimPrefix(r.URL.Path, "/agents/")
		name = strings.TrimSpace(strings.Trim(name, "/"))
		def, ok := registry[name]
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown agent"})
			return
		}

		var req agents.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
		if strings.TrimSpace(req.Task) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "task is required"})
			return
		}

		systemPrompt, err := prompts.Load(cfg.PromptsDir, def.PromptFile)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		model := cfg.ModelForEnvKey(def.ModelEnv)
		finalPrompt := buildPrompt(req)

		start := time.Now()
		ctx, cancel := context.WithTimeout(r.Context(), time.Duration(cfg.TimeoutSec)*time.Second)
		defer cancel()

		output, err := ollamaClient.Generate(ctx, model, systemPrompt, finalPrompt)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		resp := agents.Response{
			Agent:   name,
			Model:   model,
			Output:  output,
			Elapsed: time.Since(start).Round(time.Millisecond).String(),
		}
		writeJSON(w, http.StatusOK, resp)
	})

	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           loggingMiddleware(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("agent-server listening on %s", cfg.ListenAddr)
	log.Printf("ollama url: %s", cfg.OllamaURL)
	log.Fatal(server.ListenAndServe())
}

func buildPrompt(req agents.Request) string {
	var b strings.Builder

	b.WriteString("TASK:\n")
	b.WriteString(req.Task)
	b.WriteString("\n\n")

	if len(req.Constraints) > 0 {
		b.WriteString("CONSTRAINTS:\n")
		for _, c := range req.Constraints {
			b.WriteString("- ")
			b.WriteString(c)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if strings.TrimSpace(req.Context) != "" {
		b.WriteString("CONTEXT:\n")
		b.WriteString(req.Context)
		b.WriteString("\n\n")
	}

	if len(req.Meta) > 0 {
		b.WriteString("META:\n")
		for k, v := range req.Meta {
			b.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
		}
		b.WriteString("\n")
	}

	b.WriteString("RESPONSE RULES:\n")
	b.WriteString("- Be concrete and implementation-ready.\n")
	b.WriteString("- State assumptions clearly.\n")
	b.WriteString("- Use markdown headings and code blocks when helpful.\n")

	return b.String()
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/solthoth/ai-dev-team/internal/agents"
	"github.com/solthoth/ai-dev-team/internal/auth"
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

	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("/agents", func(w http.ResponseWriter, r *http.Request) {
		handleAgentsList(w, r, registry)
	})
	protectedMux.HandleFunc("/agents/", func(w http.ResponseWriter, r *http.Request) {
		handleAgentInvoke(w, r, cfg, registry, ollamaClient)
	})
	protectedMux.HandleFunc("/dispatch", func(w http.ResponseWriter, r *http.Request) {
		handleDispatch(w, r, cfg, registry, ollamaClient)
	})

	mux.Handle("/", auth.RequireToken(cfg.AgentToken, protectedMux))

	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           loggingMiddleware(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("agent-server listening on %s", cfg.ListenAddr)
	log.Printf("ollama url: %s", cfg.OllamaURL)

	log.Fatal(server.ListenAndServe())
}

func handleAgentsList(w http.ResponseWriter, r *http.Request, registry map[string]agents.Definition) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "method not allowed",
		})
		return
	}

	list := make([]agents.Definition, 0, len(registry))
	for _, def := range registry {
		list = append(list, def)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"agents": list,
	})
}

func handleAgentInvoke(
	w http.ResponseWriter,
	r *http.Request,
	cfg config.Config,
	registry map[string]agents.Definition,
	ollamaClient *ollama.Client,
) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "method not allowed",
		})
		return
	}

	name := strings.TrimPrefix(r.URL.Path, "/agents/")
	name = strings.TrimSpace(strings.Trim(name, "/"))

	def, ok := registry[name]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "unknown agent",
		})
		return
	}

	var req agents.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid json",
		})
		return
	}

	if strings.TrimSpace(req.Task) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "task is required",
		})
		return
	}

	systemPrompt, err := prompts.Load(cfg.PromptsDir, def.PromptFile)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	model := cfg.ModelForEnvKey(def.ModelEnv)
	finalPrompt := buildPrompt(req)

	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(cfg.TimeoutSec)*time.Second)
	defer cancel()

	start := time.Now()
	output, err := ollamaClient.Generate(ctx, model, systemPrompt, finalPrompt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	resp := agents.Response{
		Agent:   name,
		Model:   model,
		Output:  output,
		Elapsed: time.Since(start).Round(time.Millisecond).String(),
	}

	writeJSON(w, http.StatusOK, resp)
}

func handleDispatch(
	w http.ResponseWriter,
	r *http.Request,
	cfg config.Config,
	registry map[string]agents.Definition,
	ollamaClient *ollama.Client,
) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "method not allowed",
		})
		return
	}

	var req agents.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid json",
		})
		return
	}

	if strings.TrimSpace(req.Task) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "task is required",
		})
		return
	}

	routerPrompt, err := prompts.Load(cfg.PromptsDir, "router.md")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(cfg.TimeoutSec)*time.Second)
	defer cancel()

	decision, err := agents.RouteTask(
		ctx,
		ollamaClient,
		cfg.ModelRouter,
		routerPrompt,
		req.Task,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	def, ok := registry[decision.Agent]
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "router chose unknown agent",
		})
		return
	}

	systemPrompt, err := prompts.Load(cfg.PromptsDir, def.PromptFile)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	model := cfg.ModelForEnvKey(def.ModelEnv)
	finalPrompt := buildPrompt(req)

	start := time.Now()
	output, err := ollamaClient.Generate(ctx, model, systemPrompt, finalPrompt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	resp := agents.DispatchResponse{
		RoutedTo: decision.Agent,
		Reason:   decision.Reason,
		Response: agents.Response{
			Agent:   decision.Agent,
			Model:   model,
			Output:  output,
			Elapsed: time.Since(start).Round(time.Millisecond).String(),
		},
	}

	writeJSON(w, http.StatusOK, resp)
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

		keys := make([]string, 0, len(req.Meta))
		for k := range req.Meta {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			b.WriteString(fmt.Sprintf("- %s: %s\n", k, req.Meta[k]))
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
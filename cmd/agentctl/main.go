package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/solthoth/ai-dev-team/internal/agents"
	"github.com/solthoth/ai-dev-team/internal/contextpack"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "ask":
		runAsk(os.Args[2:])
	case "dispatch":
		runDispatch(os.Args[2:])
	case "health":
		runHealth(os.Args[2:])
	case "list":
		runList(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func runAsk(args []string) {
	fs := flag.NewFlagSet("ask", flag.ExitOnError)
	baseURL := fs.String("base-url", env("AGENT_BASE_URL", "http://127.0.0.1:4000"), "agent server base url")
	token := fs.String("token", env("AGENT_TOKEN", ""), "agent token")
	agentName := fs.String("agent", "planner", "agent name")
	task := fs.String("task", "", "task text")
	contextFile := fs.String("context-file", "", "path to context file")
	repoPath := fs.String("repo", "", "path to local git repository for context packing")
	constraints := fs.String("constraints", "", "comma separated constraints")
	jsonOut := fs.Bool("json", false, "print raw JSON response")
	fs.Parse(args)

	if strings.TrimSpace(*task) == "" {
		fmt.Fprintln(os.Stderr, "error: -task is required")
		os.Exit(2)
	}

	contextText := loadContext(*contextFile, *repoPath)

	req := agents.Request{
		Task:        *task,
		Context:     contextText,
		Constraints: splitCSV(*constraints),
		Meta:        buildMeta(*repoPath),
	}

	callAgentEndpoint(*baseURL, *token, "/agents/"+*agentName, req, *jsonOut)
}

func runDispatch(args []string) {
	fs := flag.NewFlagSet("dispatch", flag.ExitOnError)
	baseURL := fs.String("base-url", env("AGENT_BASE_URL", "http://127.0.0.1:4000"), "agent server base url")
	token := fs.String("token", env("AGENT_TOKEN", ""), "agent token")
	task := fs.String("task", "", "task text")
	contextFile := fs.String("context-file", "", "path to context file")
	repoPath := fs.String("repo", "", "path to local git repository for context packing")
	constraints := fs.String("constraints", "", "comma separated constraints")
	jsonOut := fs.Bool("json", false, "print raw JSON response")
	fs.Parse(args)

	if strings.TrimSpace(*task) == "" {
		fmt.Fprintln(os.Stderr, "error: -task is required")
		os.Exit(2)
	}

	contextText := loadContext(*contextFile, *repoPath)

	req := agents.Request{
		Task:        *task,
		Context:     contextText,
		Constraints: splitCSV(*constraints),
		Meta:        buildMeta(*repoPath),
	}

	callDispatchEndpoint(*baseURL, *token, "/dispatch", req, *jsonOut)
}

func runHealth(args []string) {
	fs := flag.NewFlagSet("health", flag.ExitOnError)
	baseURL := fs.String("base-url", env("AGENT_BASE_URL", "http://127.0.0.1:4000"), "agent server base url")
	fs.Parse(args)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(strings.TrimRight(*baseURL, "/") + "/healthz")
	if err != nil {
		fmt.Fprintln(os.Stderr, "health check failed:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	_, _ = io.Copy(os.Stdout, resp.Body)
}

func runList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	baseURL := fs.String("base-url", env("AGENT_BASE_URL", "http://127.0.0.1:4000"), "agent server base url")
	token := fs.String("token", env("AGENT_TOKEN", ""), "agent token")
	jsonOut := fs.Bool("json", false, "print raw JSON response")
	fs.Parse(args)

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(*baseURL, "/")+"/agents", nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "list request failed:", err)
		os.Exit(1)
	}
	if *token != "" {
		req.Header.Set("X-Agent-Token", *token)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "list failed:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		fmt.Fprintln(os.Stderr, string(body))
		os.Exit(1)
	}

	if *jsonOut {
		fmt.Println(string(body))
		return
	}

	var payload struct {
		Agents []agents.Definition `json:"agents"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		fmt.Println(string(body))
		return
	}

	for _, a := range payload.Agents {
		fmt.Printf("- %s: %s\n", a.Name, a.Description)
	}
}

func callAgentEndpoint(baseURL, token, path string, req agents.Request, jsonOut bool) {
	raw, err := json.Marshal(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal request failed:", err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: 300 * time.Second}
	url := strings.TrimRight(baseURL, "/") + path

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		fmt.Fprintln(os.Stderr, "request build failed:", err)
		os.Exit(1)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if token != "" {
		httpReq.Header.Set("X-Agent-Token", token)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		fmt.Fprintln(os.Stderr, "request failed:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		fmt.Fprintln(os.Stderr, string(body))
		os.Exit(1)
	}

	if jsonOut {
		fmt.Println(string(body))
		return
	}

	var out agents.Response
	if err := json.Unmarshal(body, &out); err != nil {
		fmt.Println(string(body))
		return
	}

	fmt.Printf("# Agent: %s\n", out.Agent)
	fmt.Printf("# Model: %s\n", out.Model)
	fmt.Printf("# Elapsed: %s\n\n", out.Elapsed)
	fmt.Println(out.Output)
}

func callDispatchEndpoint(baseURL, token, path string, req agents.Request, jsonOut bool) {
	raw, err := json.Marshal(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal request failed:", err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: 300 * time.Second}
	url := strings.TrimRight(baseURL, "/") + path

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		fmt.Fprintln(os.Stderr, "request build failed:", err)
		os.Exit(1)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if token != "" {
		httpReq.Header.Set("X-Agent-Token", token)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		fmt.Fprintln(os.Stderr, "request failed:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		fmt.Fprintln(os.Stderr, string(body))
		os.Exit(1)
	}

	if jsonOut {
		fmt.Println(string(body))
		return
	}

	var out agents.DispatchResponse
	if err := json.Unmarshal(body, &out); err != nil {
		fmt.Println(string(body))
		return
	}

	fmt.Printf("# Routed To: %s\n", out.RoutedTo)
	fmt.Printf("# Reason: %s\n\n", out.Reason)
	fmt.Printf("# Agent: %s\n", out.Response.Agent)
	fmt.Printf("# Model: %s\n", out.Response.Model)
	fmt.Printf("# Elapsed: %s\n\n", out.Response.Elapsed)
	fmt.Println(out.Response.Output)
}

func loadContext(contextFile, repoPath string) string {
	var sections []string

	if strings.TrimSpace(contextFile) != "" {
		b, err := os.ReadFile(contextFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error reading context file:", err)
			os.Exit(1)
		}
		sections = append(sections, string(b))
	}

	if strings.TrimSpace(repoPath) != "" {
		packed, err := contextpack.Build(contextpack.PackOptions{
			RepoPath: repoPath,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "error building repo context:", err)
			os.Exit(1)
		}
		sections = append(sections, packed)
	}

	return strings.Join(sections, "\n\n")
}

func buildMeta(repoPath string) map[string]string {
	meta := map[string]string{}

	if strings.TrimSpace(repoPath) != "" {
		meta["repo_path"] = repoPath
	}

	if len(meta) == 0 {
		return nil
	}

	return meta
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}

	return out
}

func env(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func usage() {
	fmt.Println(`agentctl commands:
  agentctl ask --agent planner --task "..." [--context-file file] [--repo path] [--constraints "a,b,c"] [--json]
  agentctl dispatch --task "..." [--context-file file] [--repo path] [--constraints "a,b,c"] [--json]
  agentctl health
  agentctl list [--json]`)
}
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
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "ask":
		runAsk(os.Args[2:])
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
	constraints := fs.String("constraints", "", "comma separated constraints")
	fs.Parse(args)

	if strings.TrimSpace(*task) == "" {
		fmt.Fprintln(os.Stderr, "error: -task is required")
		os.Exit(2)
	}

	var contextText string
	if *contextFile != "" {
		b, err := os.ReadFile(*contextFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error reading context file:", err)
			os.Exit(1)
		}
		contextText = string(b)
	}

	req := agents.Request{
		Task:        *task,
		Context:     contextText,
		Constraints: splitCSV(*constraints),
	}

	raw, _ := json.Marshal(req)

	client := &http.Client{Timeout: 300 * time.Second}
	url := strings.TrimRight(*baseURL, "/") + "/agents/" + *agentName

	httpReq, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	httpReq.Header.Set("Content-Type", "application/json")
	if *token != "" {
		httpReq.Header.Set("X-Agent-Token", *token)
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

	io.Copy(os.Stdout, resp.Body)
}

func runList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	baseURL := fs.String("base-url", env("AGENT_BASE_URL", "http://127.0.0.1:4000"), "agent server base url")
	fs.Parse(args)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(strings.TrimRight(*baseURL, "/") + "/agents")
	if err != nil {
		fmt.Fprintln(os.Stderr, "list failed:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	io.Copy(os.Stdout, resp.Body)
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
  agentctl ask --agent planner --task "..." [--context-file file] [--constraints "a,b,c"]
  agentctl health
  agentctl list`)
}

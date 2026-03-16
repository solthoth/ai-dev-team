# ai-dev-team

Local AI development team powered by Ollama and Go.

## Components

- `agent-server`: HTTP API for specialized agents
- `agentctl`: CLI for invoking agents

## Agents

- planner
- platform
- reviewer
- docs

## Build

```bash
go build -o bin/agent-server ./cmd/agent-server
go build -o bin/agentctl ./cmd/agentctl
```

### Windows

```bash
GOOS=windows GOARCH=386 go build -o bin/agent-server.exe ./cmd/agent-server
```

## Run server

```bash
AGENT_LISTEN=0.0.0.0:4000 \
OLLAMA_URL=http://127.0.0.1:11434 \
PROMPTS_DIR=./prompts \
./bin/agent-server
```

## Ask an agent

```bash
./bin/agentctl ask \
  --base-url http://127.0.0.1:4000 \
  --agent planner \
  --task "Plan a mono-repo AI agent platform in Go"
```

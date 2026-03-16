$env:AGENT_LISTEN="0.0.0.0:4000"
$env:OLLAMA_URL="http://127.0.0.1:11434"
$env:MODEL_PLANNER="llama3.1:8b"
$env:MODEL_PLATFORM="qwen2.5-coder:7b"
$env:MODEL_REVIEWER="llama3.1:8b"
$env:MODEL_DOCS="llama3.1:8b"

.\bin\agent-server.exe
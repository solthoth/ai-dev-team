package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/solthoth/ai-dev-team/internal/ollama"
)

func RouteTask(
	ctx context.Context,
	client *ollama.Client,
	model string,
	systemPrompt string,
	task string,
) (RouteDecision, error) {
	resp, err := client.Generate(ctx, model, systemPrompt, task)
	if err != nil {
		return RouteDecision{}, err
	}

	raw := strings.TrimSpace(resp)

	var decision RouteDecision
	if err := json.Unmarshal([]byte(raw), &decision); err != nil {
		return RouteDecision{}, fmt.Errorf("failed to parse router response: %w; raw=%s", err, raw)
	}

	return decision, nil
}
package agents

type Request struct {
	Task        string            `json:"task"`
	Context     string            `json:"context,omitempty"`
	Constraints []string          `json:"constraints,omitempty"`
	Meta        map[string]string `json:"meta,omitempty"`
}

type Response struct {
	Agent   string `json:"agent"`
	Model   string `json:"model"`
	Output  string `json:"output"`
	Elapsed string `json:"elapsed"`
}

type Definition struct {
	Name        string `json:"name"`
	PromptFile  string `json:"prompt_file"`
	ModelEnv    string `json:"model_env"`
	Description string `json:"description"`
}

type DispatchResponse struct {
	RoutedTo string   `json:"routed_to"`
	Reason   string   `json:"reason"`
	Response Response `json:"response"`
}

type RouteDecision struct {
	Agent  string `json:"agent"`
	Reason string `json:"reason"`
}
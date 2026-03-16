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
	Name       string
	PromptFile string
	ModelEnv   string
}
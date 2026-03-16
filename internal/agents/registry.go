package agents

func Registry() map[string]Definition {
	return map[string]Definition{
		"planner": {
			Name:        "planner",
			PromptFile:  "planner.md",
			ModelEnv:    "MODEL_PLANNER",
			Description: "Project planning, architecture, milestones, and implementation sequencing",
		},
		"platform": {
			Name:        "platform",
			PromptFile:  "platform.md",
			ModelEnv:    "MODEL_PLATFORM",
			Description: "Infrastructure, CI/CD, Docker, Kubernetes, Terraform, OpenTofu, Crossplane",
		},
		"reviewer": {
			Name:        "reviewer",
			PromptFile:  "reviewer.md",
			ModelEnv:    "MODEL_REVIEWER",
			Description: "Review, edge cases, rollout risk, reliability, validation strategy",
		},
		"docs": {
			Name:        "docs",
			PromptFile:  "docs.md",
			ModelEnv:    "MODEL_DOCS",
			Description: "README, setup guides, runbooks, troubleshooting, developer documentation",
		},
	}
}
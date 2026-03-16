package agents

func Registry() map[string]Definition {
	return map[string]Definition{
		"planner": {
			Name:       "planner",
			PromptFile: "planner.md",
			ModelEnv:   "MODEL_PLANNER",
		},
		"platform": {
			Name:       "platform",
			PromptFile: "platform.md",
			ModelEnv:   "MODEL_PLATFORM",
		},
		"reviewer": {
			Name:       "reviewer",
			PromptFile: "reviewer.md",
			ModelEnv:   "MODEL_REVIEWER",
		},
		"docs": {
			Name:       "docs",
			PromptFile: "docs.md",
			ModelEnv:   "MODEL_DOCS",
		},
	}
}
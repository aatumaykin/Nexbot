package config

func DefaultDockerConfig() DockerConfig {
	return DockerConfig{
		Enabled:                 false,
		PullPolicy:              "if-not-present",
		ContainerCount:          1,
		TaskTimeout:             300,
		WorkspaceMount:          "~/.nexbot",
		SubagentPromptsPath:     "~/.nexbot/subagent",
		SkillsMountPath:         "~/.nexbot/skills",
		MemoryLimit:             "128m",
		CPULimit:                0.5,
		PidsLimit:               50,
		LLMAPIKeyEnv:            "ZAI_API_KEY",
		MaxTasksPerMinute:       60,
		CircuitBreakerThreshold: 5,
		CircuitBreakerTimeout:   30,
		HealthCheckInterval:     30,
		MaxPendingPerContainer:  100,
		InspectTTL:              5,
		SecretsTTL:              300,
		SecurityOpt:             []string{"no-new-privileges"},
		ReadonlyRootfs:          true,
	}
}

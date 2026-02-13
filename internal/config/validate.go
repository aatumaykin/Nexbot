package config

import (
	"fmt"
	"strings"
)

func (c *DockerConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.ImageName == "" {
		return fmt.Errorf("docker.image_name is required when docker.enabled=true")
	}

	validPolicies := map[string]bool{
		"always":         true,
		"if-not-present": true,
		"never":          true,
	}
	if !validPolicies[c.PullPolicy] {
		return fmt.Errorf("docker.pull_policy must be one of: always, if-not-present, never")
	}

	if c.ContainerCount < 1 {
		return fmt.Errorf("docker.container_count must be >= 1")
	}

	if c.TaskTimeout < 1 {
		return fmt.Errorf("docker.task_timeout_seconds must be >= 1")
	}

	if c.MemoryLimit != "" && !isValidMemoryLimit(c.MemoryLimit) {
		return fmt.Errorf("docker.memory_limit format invalid (e.g., 128m, 1g)")
	}

	if c.CPULimit <= 0 || c.CPULimit > 4 {
		return fmt.Errorf("docker.cpu_limit must be between 0 and 4")
	}

	if c.MaxTasksPerMinute < 1 {
		return fmt.Errorf("docker.max_tasks_per_minute must be >= 1")
	}

	if c.InspectTTL < 0 || c.InspectTTL > 60 {
		return fmt.Errorf("docker.inspect_ttl_seconds must be between 0 and 60 (got %d)", c.InspectTTL)
	}

	if c.CircuitBreakerTimeout < 5 || c.CircuitBreakerTimeout > 300 {
		return fmt.Errorf("docker.circuit_breaker_timeout_seconds must be between 5 and 300 (got %d)", c.CircuitBreakerTimeout)
	}

	return nil
}

func isValidMemoryLimit(s string) bool {
	s = strings.ToLower(s)
	suffixes := []string{"k", "m", "g"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			num := strings.TrimSuffix(s, suffix)
			for _, c := range num {
				if c < '0' || c > '9' {
					return false
				}
			}
			return true
		}
	}
	return false
}

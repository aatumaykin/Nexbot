package docker

import (
	"context"
	"testing"
	"time"
)

type testLogger struct{}

func (l *testLogger) Info(msg string, args ...interface{})  {}
func (l *testLogger) Warn(msg string, args ...interface{})  {}
func (l *testLogger) Error(msg string, args ...interface{}) {}

// TestOnDemandContainerCreation tests that containers are created on demand.
func TestOnDemandContainerCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	client, err := NewDockerClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer client.Close()

	poolCfg := PoolConfig{
		BinaryPath:          "/tmp/nexbot-bin/nexbot",
		SubagentPromptsPath: "/Users/atumaikin/.nexbot/subagent",
		SkillsMountPath:     "/Users/atumaikin/.nexbot/skills",
		ConfigPath:          "/Users/atumaikin/.config/nexbot/config.toml",
		ContainerCount:      0,
		TaskTimeout:         30 * time.Second,
		MemoryLimit:         "128m",
		CPULimit:            0.5,
		PidsLimit:           50,
	}

	pool, err := NewContainerPoolWithClient(poolCfg, &testLogger{}, client)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Stop(context.Background())

	ctx := context.Background()

	err = pool.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}

	time.Sleep(1 * time.Second)

	pool.mu.RLock()
	initialContainerCount := len(pool.containers)
	pool.mu.RUnlock()

	if initialContainerCount != 0 {
		t.Errorf("Expected 0 containers after start, got %d", initialContainerCount)
	}

	container, err := pool.CreateContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}

	if container == nil {
		t.Fatal("Container is nil")
	}

	pool.mu.RLock()
	containerCountAfterCreate := len(pool.containers)
	pool.mu.RUnlock()

	if containerCountAfterCreate != 1 {
		t.Errorf("Expected 1 container after CreateContainer, got %d", containerCountAfterCreate)
	}

	time.Sleep(1 * time.Second)

	pool.Release(container.ID)

	time.Sleep(2 * time.Second)

	pool.mu.RLock()
	containerCountAfterRelease := len(pool.containers)
	pool.mu.RUnlock()

	if containerCountAfterRelease != 0 {
		t.Errorf("Expected 0 containers after release, got %d", containerCountAfterRelease)
	}
}

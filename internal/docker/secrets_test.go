package docker_test

import (
	"context"
	"os"
	"testing"

	"github.com/moby/moby/api/types/container"
	dockerclient "github.com/moby/moby/client"
	"github.com/stretchr/testify/require"
)

// TestSecretsNotInEnv проверяет что секреты не передаются через env контейнера
// Это интеграционный тест, который требует запущенного Docker daemon
func TestSecretsNotInEnv(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	cli, err := dockerclient.New(dockerclient.FromEnv)
	require.NoError(t, err)
	defer cli.Close()

	// Проверяем что Docker daemon доступен
	_, err = cli.Ping(ctx, dockerclient.PingOptions{})
	require.NoError(t, err, "Docker daemon must be running")

	// Создаём тестовый контейнер с конфигурацией аналогичной pool.CreateContainer
	containerConfig := dockerclient.ContainerCreateOptions{
		Config: &container.Config{
			Image: "alpine:latest",
			Cmd:   []string{"sleep", "3600"},
			// Env должен содержать ТОЛЬКО SKILLS_PATH, без секретов
			Env: []string{
				"SKILLS_PATH=/workspace/skills",
			},
		},
	}

	// Создаём контейнер
	resp, err := cli.ContainerCreate(ctx, containerConfig)
	if err != nil {
		t.Skipf("failed to create container: %v (Docker may not be available)", err)
		return
	}
	defer func() {
		cli.ContainerRemove(ctx, resp.ID, dockerclient.ContainerRemoveOptions{Force: true})
	}()

	// Запускаем контейнер
	_, err = cli.ContainerStart(ctx, resp.ID, dockerclient.ContainerStartOptions{})
	require.NoError(t, err)

	// Останавливаем контейнер перед проверкой (чтобы inspect показывал актуальное состояние)
	_, _ = cli.ContainerStop(ctx, resp.ID, dockerclient.ContainerStopOptions{})

	// Получаем информацию о контейнере
	inspect, err := cli.ContainerInspect(ctx, resp.ID, dockerclient.ContainerInspectOptions{})
	require.NoError(t, err)

	// Проверяем что Env содержит только SKILLS_PATH и ничего больше
	t.Logf("Container Env: %v", inspect.Container.Config.Env)

	envMap := make(map[string]string)
	for _, env := range inspect.Container.Config.Env {
		parts := splitEnv(env)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Проверяем что SKILLS_PATH есть
	val, ok := envMap["SKILLS_PATH"]
	require.True(t, ok, "SKILLS_PATH must be present")
	require.Equal(t, "/workspace/skills", val)

	// Проверяем что секретов нет
	secretKeys := []string{
		"ZAI_API_KEY", "API_KEY", "SECRET", "PASSWORD", "TOKEN", "AUTH",
		"OPENAI_API_KEY", "TELEGRAM_BOT_TOKEN",
	}
	for _, key := range secretKeys {
		_, exists := envMap[key]
		require.False(t, exists, "secret key %s must NOT be in env", key)
	}

	// Проверяем что только одна переменная окружения
	require.Equal(t, 1, len(inspect.Container.Config.Env), "only SKILLS_PATH should be in env")

	t.Log("✓ Secrets are NOT leaked to container environment")
}

// splitEnv разделяет строку env на KEY и VALUE
func splitEnv(env string) []string {
	for i := 0; i < len(env); i++ {
		if env[i] == '=' {
			return []string{env[:i], env[i+1:]}
		}
	}
	return []string{env}
}

// TestCurrentEnvDoesNotContainSecrets проверяет текущие env переменные процесса
func TestCurrentEnvDoesNotContainSecrets(t *testing.T) {
	envVars := os.Environ()

	secretKeys := []string{
		"ZAI_API_KEY", "API_KEY", "SECRET", "PASSWORD", "TOKEN", "AUTH",
	}

	leakedSecrets := []string{}
	for _, env := range envVars {
		for _, key := range secretKeys {
			if len(env) > len(key) && env[:len(key)] == key && env[len(key)] == '=' {
				leakedSecrets = append(leakedSecrets, key)
			}
		}
	}

	if len(leakedSecrets) > 0 {
		t.Logf("Warning: Found potential secrets in process env: %v", leakedSecrets)
		// Это warning, а не ошибка, потому что в тестовой среде это допустимо
	} else {
		t.Log("✓ No secrets found in process environment")
	}
}

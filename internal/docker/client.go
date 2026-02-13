package docker

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	dockerclient "github.com/moby/moby/client"
)

type DockerClientInterface interface {
	PullImage(ctx context.Context, cfg PoolConfig) error
	CreateContainer(ctx context.Context, cfg PoolConfig) (string, error)
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, id string, timeout *int) error
	RemoveContainer(ctx context.Context, id string) error
	AttachContainer(ctx context.Context, id string) (dockerclient.HijackedResponse, error)
	InspectContainer(ctx context.Context, id string) (container.InspectResponse, error)
	Close() error
}

type DockerClient struct {
	client *dockerclient.Client
}

func NewDockerClient() (*DockerClient, error) {
	cli, err := dockerclient.New(dockerclient.WithAPIVersionNegotiation(), dockerclient.FromEnv)
	if err != nil {
		return nil, &DockerError{Op: "connect", Err: err, Message: "failed to connect to Docker daemon"}
	}

	ctx := context.Background()
	if _, err := cli.Ping(ctx, dockerclient.PingOptions{NegotiateAPIVersion: true}); err != nil {
		return nil, &DockerError{Op: "ping", Err: err, Message: "Docker daemon not available"}
	}

	return &DockerClient{client: cli}, nil
}

func (c *DockerClient) Close() error {
	return c.client.Close()
}

func (c *DockerClient) PullImage(ctx context.Context, cfg PoolConfig) error {
	if cfg.PullPolicy == "never" {
		return nil
	}

	imageRef := cfg.ImageName
	if cfg.ImageDigest != "" {
		imageRef = cfg.ImageName + "@" + cfg.ImageDigest
	} else if cfg.ImageTag != "" && !strings.Contains(cfg.ImageName, ":") {
		imageRef = cfg.ImageName + ":" + cfg.ImageTag
	}

	resp, err := c.client.ImagePull(ctx, imageRef, dockerclient.ImagePullOptions{})
	if err != nil {
		if cfg.PullPolicy == "if-not-present" {
			return nil
		}
		return &DockerError{Op: "pull", Err: err, Message: fmt.Sprintf("failed to pull image %s", imageRef)}
	}
	defer resp.Close()

	if err := resp.Wait(ctx); err != nil {
		if cfg.PullPolicy == "if-not-present" {
			return nil
		}
		return &DockerError{Op: "pull", Err: err, Message: fmt.Sprintf("failed to pull image %s", imageRef)}
	}

	return nil
}

func (c *DockerClient) CreateContainer(ctx context.Context, cfg PoolConfig) (string, error) {
	mounts := []mount.Mount{
		{
			Type:     mount.TypeBind,
			Source:   cfg.SkillsPath,
			Target:   "/workspace/skills",
			ReadOnly: true,
		},
	}

	memoryLimit := parseMemory(cfg.MemoryLimit)
	if memoryLimit == 0 {
		memoryLimit = 128 * 1024 * 1024
	}

	cpuLimit := cfg.CPULimit
	if cpuLimit == 0 {
		cpuLimit = 0.5
	}

	pidsLimit := cfg.PidsLimit
	if pidsLimit == 0 {
		pidsLimit = 50
	}

	securityOpt := cfg.SecurityOpt
	if len(securityOpt) == 0 {
		securityOpt = []string{"no-new-privileges"}
	}

	readonlyRootfs := cfg.ReadonlyRootfs != nil && *cfg.ReadonlyRootfs

	result, err := c.client.ContainerCreate(ctx, dockerclient.ContainerCreateOptions{
		Image: cfg.ImageName,
		Config: &container.Config{
			Image: cfg.ImageName,
		},
		HostConfig: &container.HostConfig{
			Resources: container.Resources{
				Memory:    memoryLimit,
				NanoCPUs:  int64(cpuLimit * 1e9),
				PidsLimit: &pidsLimit,
			},
			Mounts:         mounts,
			SecurityOpt:    securityOpt,
			ReadonlyRootfs: readonlyRootfs,
			Tmpfs:          map[string]string{"/tmp": "rw,size=50m"},
		},
	})
	if err != nil {
		return "", &DockerError{Op: "create", Err: err, Message: "failed to create container"}
	}

	return result.ID, nil
}

func (c *DockerClient) StartContainer(ctx context.Context, id string) error {
	_, err := c.client.ContainerStart(ctx, id, dockerclient.ContainerStartOptions{})
	if err != nil {
		return &DockerError{Op: "start", Err: err, Message: fmt.Sprintf("failed to start container %s", id)}
	}
	return nil
}

func (c *DockerClient) StopContainer(ctx context.Context, id string, timeout *int) error {
	_, err := c.client.ContainerStop(ctx, id, dockerclient.ContainerStopOptions{Timeout: timeout})
	if err != nil {
		return &DockerError{Op: "stop", Err: err, Message: fmt.Sprintf("failed to stop container %s", id)}
	}
	return nil
}

func (c *DockerClient) RemoveContainer(ctx context.Context, id string) error {
	_, err := c.client.ContainerRemove(ctx, id, dockerclient.ContainerRemoveOptions{Force: true})
	if err != nil {
		return &DockerError{Op: "remove", Err: err, Message: fmt.Sprintf("failed to remove container %s", id)}
	}
	return nil
}

func (c *DockerClient) AttachContainer(ctx context.Context, id string) (dockerclient.HijackedResponse, error) {
	result, err := c.client.ContainerAttach(ctx, id, dockerclient.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return dockerclient.HijackedResponse{}, &DockerError{Op: "attach", Err: err, Message: fmt.Sprintf("failed to attach to container %s", id)}
	}
	return result.HijackedResponse, nil
}

func (c *DockerClient) InspectContainer(ctx context.Context, id string) (container.InspectResponse, error) {
	result, err := c.client.ContainerInspect(ctx, id, dockerclient.ContainerInspectOptions{})
	if err != nil {
		return container.InspectResponse{}, err
	}
	return result.Container, nil
}

func parseMemory(s string) int64 {
	if s == "" {
		return 0
	}

	s = strings.TrimSpace(strings.ToLower(s))

	multiplier := int64(1)
	if strings.HasSuffix(s, "g") {
		multiplier = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "g")
	} else if strings.HasSuffix(s, "m") {
		multiplier = 1024 * 1024
		s = strings.TrimSuffix(s, "m")
	} else if strings.HasSuffix(s, "k") {
		multiplier = 1024
		s = strings.TrimSuffix(s, "k")
	}

	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}

	return val * multiplier
}

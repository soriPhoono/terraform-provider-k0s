package provider

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type dockerClient struct {
	binary string
}

func newDockerClient(binary ...string) *dockerClient {
	b := "docker"
	if len(binary) > 0 && binary[0] != "" {
		b = binary[0]
	}
	return &dockerClient{binary: b}
}

func (d *dockerClient) run(ctx context.Context, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, d.binary, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("docker %s: %s\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (d *dockerClient) isRunning(ctx context.Context, container string) (bool, error) {
	state, err := d.run(ctx, "inspect", "--format", "{{.State.Status}}", container)
	if err != nil {
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "no such object") ||
			strings.Contains(errStr, "no such container") {
			return false, nil
		}
		return false, err
	}
	return state == "running", nil
}

func (d *dockerClient) createContainer(
	ctx context.Context,
	name, hostname, image string,
	privileged bool,
	ports, volumes, tmpfs []string,
	env map[string]string,
	network string,
	cmdArgs []string,
) (string, error) {
	args := []string{"create"}
	if name != "" {
		args = append(args, "--name", name)
	}
	if hostname != "" {
		args = append(args, "--hostname", hostname)
	}
	if privileged {
		args = append(args, "--privileged")
	}
	for _, p := range ports {
		args = append(args, "-p", p)
	}
	for _, v := range volumes {
		args = append(args, "-v", v)
	}
	for _, t := range tmpfs {
		args = append(args, "--tmpfs", t)
	}
	for k, v := range env {
		args = append(args, "-e", k+"="+v)
	}
	if network != "" {
		args = append(args, "--network", network)
	}
	args = append(args, image)
	args = append(args, cmdArgs...)
	return d.run(ctx, args...)
}

func (d *dockerClient) startContainer(ctx context.Context, name string) error {
	_, err := d.run(ctx, "start", name)
	return err
}

func (d *dockerClient) removeContainer(ctx context.Context, name string, force bool) error {
	args := []string{"rm"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, name)
	_, err := d.run(ctx, args...)
	return err
}

func (d *dockerClient) exec(
	ctx context.Context,
	container string,
	cmdArgs ...string,
) (string, error) {
	args := append([]string{"exec", container}, cmdArgs...)
	return d.run(ctx, args...)
}

func (d *dockerClient) inspectField(ctx context.Context, container, format string) (string, error) {
	return d.run(ctx, "inspect", "--format", format, container)
}

func (d *dockerClient) networkExists(ctx context.Context, name string) (bool, error) {
	_, err := d.run(ctx, "network", "inspect", name)
	if err != nil {
		if strings.Contains(err.Error(), "No such network") ||
			strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (d *dockerClient) createNetwork(ctx context.Context, name string) (string, error) {
	return d.run(ctx, "network", "create", "--driver", "bridge", name)
}

func (d *dockerClient) removeNetwork(ctx context.Context, name string) error {
	_, err := d.run(ctx, "network", "rm", name)
	return err
}

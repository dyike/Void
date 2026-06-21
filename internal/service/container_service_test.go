package service

import (
	"strings"
	"testing"

	"github.com/dyike/Void/internal/model"
)

func TestBuildRunCommandKeepsDetachedBaseImageRunning(t *testing.T) {
	svc := NewContainerService(nil, nil)

	command := svc.BuildRunCommand(model.RunContainerOptions{
		Image:  "docker.io/library/alpine:latest",
		Detach: true,
		Remove: true,
	})

	if strings.Contains(command, "--rm") {
		t.Fatalf("detached run command must not include --rm: %s", command)
	}
	if !strings.Contains(command, "--detach") {
		t.Fatalf("expected detached run command: %s", command)
	}
	if !strings.Contains(command, "docker.io/library/alpine:latest /bin/sh -lc 'sleep infinity'") {
		t.Fatalf("expected alpine keep-alive shell command, got: %s", command)
	}
}

func TestBuildRunCommandKeepsServiceImageDefaultEntrypoint(t *testing.T) {
	svc := NewContainerService(nil, nil)

	command := svc.BuildRunCommand(model.RunContainerOptions{
		Image:  "docker.io/library/nginx:latest",
		Detach: true,
	})

	if strings.Contains(command, "/bin/sh -lc") {
		t.Fatalf("service image should keep image default command: %s", command)
	}
	if !strings.HasSuffix(command, "docker.io/library/nginx:latest") {
		t.Fatalf("unexpected command: %s", command)
	}
}

func TestBuildRunCommandOmitsZeroResources(t *testing.T) {
	svc := NewContainerService(nil, nil)

	command := svc.BuildRunCommand(model.RunContainerOptions{
		Image:  "docker.io/library/alpine:latest",
		Detach: true,
		CPUs:   "0",
		Memory: "0",
	})

	if strings.Contains(command, "--cpus") {
		t.Fatalf("zero cpus should be omitted: %s", command)
	}
	if strings.Contains(command, "--memory") {
		t.Fatalf("zero memory should be omitted: %s", command)
	}

	command = svc.BuildRunCommand(model.RunContainerOptions{
		Image:  "docker.io/library/alpine:latest",
		Detach: true,
		Memory: "256M",
	})
	if !strings.Contains(command, "--memory 256M") {
		t.Fatalf("non-zero memory should be preserved: %s", command)
	}
}

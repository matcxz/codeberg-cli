package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strings"

	"github.com/matcxz/codeberg-cli/internal/config"
)

func resolveRepository(ctx context.Context, repository string) (string, error) {
	if repository != "" {
		if _, _, ok := splitRepository(repository); !ok {
			return "", fmt.Errorf("repository must be in OWNER/NAME format")
		}
		return repository, nil
	}

	cfg, err := config.Load()
	if err != nil {
		return "", err
	}
	command := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("--repo is required when the current directory has no Git origin remote")
	}
	repository, err = repositoryFromRemote(strings.TrimSpace(string(output)), configuredHost(cfg))
	if err != nil {
		return "", err
	}
	return repository, nil
}

func repositoryFromRemote(remote, instanceHost string) (string, error) {
	instance, err := url.Parse(instanceHost)
	if err != nil || instance.Hostname() == "" {
		return "", fmt.Errorf("invalid configured instance URL %q", instanceHost)
	}

	var remoteHost, remotePath string
	if strings.Contains(remote, "://") {
		parsed, err := url.Parse(remote)
		if err != nil || parsed.Hostname() == "" {
			return "", fmt.Errorf("could not parse Git origin remote")
		}
		remoteHost, remotePath = parsed.Hostname(), parsed.Path
	} else {
		hostPath := strings.SplitN(remote, ":", 2)
		if len(hostPath) != 2 {
			return "", fmt.Errorf("could not parse Git origin remote")
		}
		remoteHost = strings.TrimPrefix(hostPath[0], "git@")
		remotePath = "/" + hostPath[1]
	}
	if !strings.EqualFold(remoteHost, instance.Hostname()) {
		return "", fmt.Errorf("Git origin host %q does not match configured instance %q; pass --repo OWNER/NAME or set --host", remoteHost, instance.Host)
	}
	basePath := strings.Trim(instance.Path, "/")
	if basePath != "" {
		prefix := "/" + basePath + "/"
		if !strings.HasPrefix(remotePath, prefix) {
			return "", fmt.Errorf("Git origin path does not match configured instance base path")
		}
		remotePath = strings.TrimPrefix(remotePath, "/"+basePath)
	}
	remotePath = strings.Trim(strings.TrimSuffix(strings.TrimSpace(remotePath), ".git"), "/")
	owner, name, ok := splitRepository(remotePath)
	if !ok {
		return "", fmt.Errorf("Git origin must point to an OWNER/NAME repository")
	}
	return owner + "/" + name, nil
}

package cmd

import (
	"fmt"

	"github.com/trevorphillipscoding/nvy/internal/env"
	"github.com/trevorphillipscoding/nvy/internal/semver"
	"github.com/trevorphillipscoding/nvy/plugins"
)

func resolveInstallVersion(p plugins.Plugin, requested string) (string, error) {
	available, err := p.AvailableVersions(env.OS(), env.Arch())
	if err != nil {
		return "", err
	}
	v, err := semver.Resolve(requested, available)
	if err != nil {
		return "", err
	}
	return v, nil
}

func resolveInstalledVersion(tool, requested string) (string, error) {
	installed, err := env.InstalledVersions(tool)
	if err != nil {
		return "", fmt.Errorf("%s has no installed versions", tool)
	}
	v, err := semver.Resolve(requested, installed)
	if err != nil {
		return "", err
	}
	return v, nil
}

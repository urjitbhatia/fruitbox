// Package compose loads Docker Compose files into a fully-resolved project
// model. It wraps the official compose-spec/compose-go reference loader (the
// same library Docker Compose itself uses) so that fruitbox parses, validates,
// interpolates, merges and normalizes compose files identically to
// `docker compose`.
package compose

import (
	"context"
	"path/filepath"

	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/types"
)

// LoadOptions configures how a compose project is loaded. It mirrors the
// inputs `docker compose` accepts on the command line.
type LoadOptions struct {
	// ConfigPaths are the compose files to load, in order. When empty, the
	// loader discovers the default files (compose.yaml, docker-compose.yml, …)
	// starting from WorkingDir.
	ConfigPaths []string
	// WorkingDir is the project working directory. When empty it defaults to
	// the directory of the first config file (or the process cwd).
	WorkingDir string
	// ProjectName overrides the project name. When empty the name is derived
	// from the working directory (or COMPOSE_PROJECT_NAME).
	ProjectName string
	// Profiles enables the named compose profiles.
	Profiles []string
	// EnvFiles are additional .env files to load before OS environment.
	EnvFiles []string

	// The following toggles mirror `docker compose config` flags. Zero values
	// keep the default (full) loading behavior.

	// NoInterpolate disables ${VAR} interpolation (--no-interpolate).
	NoInterpolate bool
	// NoNormalize disables model normalization (--no-normalize).
	NoNormalize bool
	// NoConsistency skips the consistency check (--no-consistency).
	NoConsistency bool
	// NoResolvePaths leaves relative paths unresolved (--no-path-resolution).
	NoResolvePaths bool
	// NoEnvResolution skips computing service environments (--no-env-resolution).
	NoEnvResolution bool
}

// Load resolves a compose project from the given options, returning the
// canonical compose-go project model with services, networks, volumes,
// configs and secrets fully resolved.
func Load(ctx context.Context, opts LoadOptions) (*types.Project, error) {
	// Resolve config paths to absolute so that relative references inside the
	// compose files (e.g. secret/config `file:` paths, build contexts) resolve
	// against the compose file's directory regardless of the process cwd.
	configPaths := make([]string, 0, len(opts.ConfigPaths))
	for _, p := range opts.ConfigPaths {
		if abs, err := filepath.Abs(p); err == nil {
			configPaths = append(configPaths, abs)
		} else {
			configPaths = append(configPaths, p)
		}
	}

	projectOpts := []cli.ProjectOptionsFn{
		cli.WithWorkingDirectory(opts.WorkingDir),
		// Resolve the working directory before discovering config files so
		// relative default-file lookup is anchored correctly.
		cli.WithConfigFileEnv,
		cli.WithDefaultConfigPath,
	}
	if len(opts.EnvFiles) > 0 {
		projectOpts = append(projectOpts, cli.WithEnvFiles(opts.EnvFiles...))
	}
	// Load .env then OS environment (OS wins), matching docker compose.
	projectOpts = append(projectOpts,
		cli.WithDotEnv,
		cli.WithOsEnv,
	)
	if opts.ProjectName != "" {
		projectOpts = append(projectOpts, cli.WithName(opts.ProjectName))
	}
	if len(opts.Profiles) > 0 {
		projectOpts = append(projectOpts, cli.WithProfiles(opts.Profiles))
	}
	// Resolve paths by default (so build contexts / file: refs are absolute),
	// unless the caller opted out.
	projectOpts = append(projectOpts, cli.WithResolvedPaths(!opts.NoResolvePaths))
	// Drop networks/volumes/secrets/configs/models not referenced by active
	// services, matching docker compose (e.g. resources used only by a
	// profile-disabled service are pruned).
	projectOpts = append(projectOpts, cli.WithoutUnnecessaryResources)
	if opts.NoInterpolate {
		projectOpts = append(projectOpts, cli.WithInterpolation(false))
	}
	if opts.NoNormalize {
		projectOpts = append(projectOpts, cli.WithNormalization(false))
	}
	if opts.NoConsistency {
		projectOpts = append(projectOpts, cli.WithConsistency(false))
	}
	if opts.NoEnvResolution {
		projectOpts = append(projectOpts, cli.WithoutEnvironmentResolution)
	}

	options, err := cli.NewProjectOptions(configPaths, projectOpts...)
	if err != nil {
		return nil, err
	}

	project, err := cli.ProjectFromOptions(ctx, options)
	if err != nil {
		return nil, err
	}
	return project, nil
}

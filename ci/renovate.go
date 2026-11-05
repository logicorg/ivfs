package main

import (
	"context"
	"dagger/ivfs/internal/dagger"

	"time"
)

func (m *Ivfs) Renovate(
	ctx context.Context,
	githubToken *dagger.Secret,
	// +default="logicorg/ivfs"
	repo string,
	// +default=false
	dryRun bool,
) *dagger.Container {
	args := []string{}
	if dryRun {
		args = append(args, "--dry-run")
	}

	cacheHack := time.Now() // prevent dagger from caching the renovate container
	return dag.Container().From(renovateImage).
		WithSecretVariable("RENOVATE_TOKEN", githubToken).
		WithEnvVariable("RENOVATE_PLATFORM", "github").
		WithEnvVariable("RENOVATE_REQUIRE_CONFIG", "required").
		WithEnvVariable("RENOVATE_DEPENDENCY_DASHBOARD", "false").
		WithEnvVariable("RENOVATE_GIT_AUTHOR", "Renovate Bot <bot@renovateapp.com>").
		WithEnvVariable("RENOVATE_REPOSITORIES", repo).
		WithEnvVariable("CACHE_HACK", cacheHack.String()).
		WithExec(args, dagger.ContainerWithExecOpts{UseEntrypoint: true})
}

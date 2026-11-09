package main

import (
	"context"
	"dagger/ivfs/internal/dagger"
	"fmt"
)

func (m *Ivfs) UnitTests(
	ctx context.Context,
	// +defaultPath=./
	// +ignore=["**/.git", ".claude", ".vscode", "compose", "cover.out", "k8s", "ci", "web/ui/node_modules", "web/ui/dist/app", "web/ui/*.tsbuildinfo"]
	buildContext *dagger.Directory,
) (*dagger.File, error) {

	dind := dag.Container().
		From("docker:dind").
		WithEnvVariable("TINI_SUBREAPER", "true").
		WithMountedCache("/var/lib/docker", dag.CacheVolume("dind-data")).
		WithExposedPort(2375).AsService(dagger.ContainerAsServiceOpts{
		Args: []string{
			"dockerd", "--tls=false", "--host=tcp://0.0.0.0:2375",
		},
		InsecureRootCapabilities: true,
		UseEntrypoint:            true,
	})

	dindService, err := dind.Start(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start dind: %w", err)
	}
	defer dindService.Stop(ctx)

	ctr := dag.Container(dagger.ContainerOpts{}).
		From(golangImage).
		WithServiceBinding("docker-host", dindService).
		WithEnvVariable("DOCKER_HOST", "tcp://docker-host:2375").
		WithEnvVariable("TESTCONTAINERS_RYUK_DISABLED", "true").
		WithDirectory("/workspace", buildContext).
		WithWorkdir("/workspace").
		WithExec([]string{"go", "test", "./...", "-coverprofile=coverage.out"})

	coverageFile := ctr.File("coverage.out")
	return coverageFile, nil
}

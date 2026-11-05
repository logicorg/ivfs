package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"dagger/ivfs/internal/dagger"
)

const (
	// moduleArtifactType is what cuestomize's own examples are pushed as.
	moduleArtifactType = "application/vnd.cue.module.v1+json"
)

// CheckModule vets the cuestomize module and renders it with its example
// input, returning the rendered resources.
func (m *Ivfs) CheckModule(ctx context.Context) (string, error) {
	ctr := dag.Container().From(cueImage).
		WithDirectory("/module", m.Module).
		WithWorkdir("/module").
		// Non-concrete: the required inputs are a Cuestomization's to set.
		WithExec([]string{"cue", "vet", "-c=false", "."}).
		WithExec([]string{"cue", "export", "-t", "example", ".", "-e", "outputs", "--out", "yaml"})
	out, err := ctr.Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("check the cuestomize module: %w", err)
	}
	return out, nil
}

// PublishModule checks the cuestomize module, then pushes it under every
// tag, returning what oras reports.
func (m *Ivfs) PublishModule(
	ctx context.Context,
	// The repository the module goes to; a remoteModule.ref names it with a tag.
	// +default="ghcr.io/logicorg/ivfs/cuestomization"
	repository string,
	// The tags to publish, such as v1.2.0, or main and sha-1a2b3c4.
	tags []string,
	// The commit the module is published from, recorded as an annotation.
	// +optional
	revision string,
	username string,
	password *dagger.Secret,
) (string, error) {
	if len(tags) == 0 {
		return "", errors.New("no tags to publish: pass --tags, such as --tags=v1.2.0")
	}
	if _, err := m.CheckModule(ctx); err != nil {
		return "", err
	}
	host, _, _ := strings.Cut(repository, "/")

	// cuestomize unpacks an artifact that is a single tarball, which keeps
	// the push to one layer however the module's files are laid out.
	script := `set -eu
tar -czf /tmp/ivfs.tar.gz -C /module .
printf '%s' "$REGISTRY_PASSWORD" | oras login "$REGISTRY_HOST" -u "$REGISTRY_USERNAME" --password-stdin
cd /tmp
oras push --artifact-type "$ARTIFACT_TYPE" ${REVISION:+--annotation "org.opencontainers.image.revision=$REVISION"} \
	"$REF" ivfs.tar.gz:application/vnd.oci.image.layer.v1.tar+gzip`
	out, err := dag.Container().From(orasImage).
		WithDirectory("/module", m.Module).
		WithEnvVariable("REGISTRY_HOST", host).
		WithEnvVariable("REGISTRY_USERNAME", username).
		WithSecretVariable("REGISTRY_PASSWORD", password).
		WithEnvVariable("ARTIFACT_TYPE", moduleArtifactType).
		WithEnvVariable("REVISION", revision).
		WithEnvVariable("REF", repository+":"+strings.Join(tags, ",")).
		WithExec([]string{"sh", "-c", script}).
		Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("publish the cuestomize module to %s: %w", repository, err)
	}
	return out, nil
}

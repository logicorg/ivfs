// Builds and publishes ivfs's container images.
//
// Every image is built from its Dockerfile, so docker build and compose stay
// equivalent to what CI ships. Each Dockerfile cross-compiles, so building
// for every platform needs no emulation but for the final, file-only stage.
package main

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"dagger/ivfs/internal/dagger"
)

// images maps each image's name, as published under the registry, to its
// Dockerfile in the repository.
var images = map[string]string{
	"worker":         "worker/Dockerfile",
	"web":            "web/Dockerfile",
	"yahoo-provider": "providers/yahoo/Dockerfile",
}

// platforms are the ones the homelab's nodes run.
var platforms = []dagger.Platform{"linux/amd64", "linux/arm64"}

type Ivfs struct {
	// The repository, as the Dockerfiles expect it for their context.
	Source *dagger.Directory
	// The cuestomize module, published apart from the images.
	Module *dagger.Directory
}

func New(
	// The repository's root. The ignored paths mirror .dockerignore, which
	// keeps the local databases and the UI's dependencies out of the upload.
	// +defaultPath="/"
	// +ignore=["**/.git", ".claude", ".vscode", "compose", "cover.out", "k8s", "ci", "web/ui/node_modules", "web/ui/dist/app", "web/ui/*.tsbuildinfo"]
	source *dagger.Directory,
	// +defaultPath="/k8s/cuestomize"
	module *dagger.Directory,
) *Ivfs {
	return &Ivfs{Source: source, Module: module}
}

// Build builds one image for one platform.
func (m *Ivfs) Build(
	ctx context.Context,
	// One of worker, web, yahoo-provider.
	image string,
	// +default="linux/amd64"
	platform dagger.Platform,
) (*dagger.Container, error) {
	dockerfile, ok := images[image]
	if !ok {
		return nil, fmt.Errorf("unknown image %q: build one of %s", image, strings.Join(imageNames(), ", "))
	}
	// DockerBuild sets no BUILDPLATFORM of its own, so without it the
	// builder stages would run emulated on the target's platform.
	buildPlatform, err := dag.DefaultPlatform(ctx)
	if err != nil {
		return nil, fmt.Errorf("read the engine's platform: %w", err)
	}
	return m.Source.DockerBuild(dagger.DirectoryDockerBuildOpts{
		Dockerfile: dockerfile,
		Platform:   platform,
		BuildArgs:  []dagger.BuildArg{{Name: "BUILDPLATFORM", Value: string(buildPlatform)}},
	}), nil
}

// BuildAll builds every image for every platform, publishing nothing: the
// check for a pull request.
func (m *Ivfs) BuildAll(ctx context.Context) error {
	return forEachImage(func(image string) error {
		variants, err := m.variants(ctx, image, "")
		if err != nil {
			return err
		}
		for _, variant := range variants {
			if _, err := variant.Sync(ctx); err != nil {
				return fmt.Errorf("build %s: %w", image, err)
			}
		}
		return nil
	})
}

// Publish builds every image for every platform and pushes each as one
// multi-platform image under every tag, returning the published references.
func (m *Ivfs) Publish(
	ctx context.Context,
	// Where the images go, each under its own name.
	// +default="ghcr.io/logicorg/ivfs"
	registry string,
	// The tags to publish, such as v1.2.0, or main and sha-1a2b3c4.
	tags []string,
	// The commit the images are built from, recorded in their labels.
	// +optional
	revision string,
	username string,
	password *dagger.Secret,
) ([]string, error) {
	if len(tags) == 0 {
		return nil, errors.New("no tags to publish: pass --tags, such as --tags=v1.2.0")
	}
	host, _, _ := strings.Cut(registry, "/")

	var (
		mu        sync.Mutex
		published []string
	)
	err := forEachImage(func(image string) error {
		variants, err := m.variants(ctx, image, revision)
		if err != nil {
			return err
		}
		for _, tag := range tags {
			ref := fmt.Sprintf("%s/%s:%s", registry, image, tag)
			digest, err := dag.Container().
				WithRegistryAuth(host, username, password).
				Publish(ctx, ref, dagger.ContainerPublishOpts{PlatformVariants: variants})
			if err != nil {
				return fmt.Errorf("publish %s: %w", ref, err)
			}
			mu.Lock()
			published = append(published, digest)
			mu.Unlock()
		}
		return nil
	})
	return published, err
}

// variants builds image for every platform, labelled with revision when it
// is set.
func (m *Ivfs) variants(ctx context.Context, image, revision string) ([]*dagger.Container, error) {
	variants := make([]*dagger.Container, 0, len(platforms))
	for _, platform := range platforms {
		ctr, err := m.Build(ctx, image, platform)
		if err != nil {
			return nil, err
		}
		if revision != "" {
			ctr = ctr.WithLabel("org.opencontainers.image.revision", revision)
		}
		variants = append(variants, ctr)
	}
	return variants, nil
}

// forEachImage runs fn for every image concurrently, so the images build in
// parallel, and reports every failure rather than the first.
func forEachImage(fn func(image string) error) error {
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)
	for image := range images {
		wg.Go(func() {
			if err := fn(image); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	return errors.Join(errs...)
}

func imageNames() []string {
	return slices.Sorted(maps.Keys(images))
}

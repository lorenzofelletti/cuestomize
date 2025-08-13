package main

import (
	"context"
	"dagger/cuestomize/internal/dagger"
)

func (m *Cuestomize) Build(
	ctx context.Context,
	// +defaultPath=./
	buildContext *dagger.Directory,
	// +default=""
	platform string,
) (*dagger.Container, error) {
	containerOpts := dagger.ContainerOpts{}
	if platform != "" {
		containerOpts.Platform = dagger.Platform(platform)
	}

	// Build stage: compile the Go binary
	builder := cuestomizeBuilderContainer(buildContext, containerOpts)

	// Final stage: create the runtime container with distroless
	container := dag.Container().
		From(DistrolessStaticImage).
		WithDirectory("/cue-resources", dag.Directory(), dagger.ContainerWithDirectoryOpts{Owner: "nobody"}).
		WithFile("/usr/local/bin/cuestomize", builder.File("/workspace/cuestomize")).
		WithEntrypoint([]string{"/usr/local/bin/cuestomize"})

	return container, nil
}

func (m *Cuestomize) BuildAndPublish(
	ctx context.Context,
	username string,
	password *dagger.Secret,
	// +defaultPath=./
	buildContext *dagger.Directory,
	// +default="ghcr.io"
	registry string,
	repository string,
	tag string,
	// +default=false
	alsoTagAsLatest bool,
	// +default=true
	runValidations bool,
	// +default=[]
	platforms []string,
) error {
	if runValidations {
		// lint
		if _, err := m.GolangciLintRun(ctx, buildContext, GolangciLintDefaultVersion, "5m"); err != nil {
			return err
		}

		// tests
		if err := m.RunTests(ctx, buildContext); err != nil {
			return err
		}
	}

	// Publish stage: push the built image to a registry
	container, err := m.Build(ctx, buildContext, "")
	if err != nil {
		return err
	}
	container = container.WithRegistryAuth(registry, username, password)

	tags := []string{tag}
	if alsoTagAsLatest {
		tags = append(tags, "latest")
	}
	for _, t := range tags {
		_, err := container.Publish(ctx, registry+"/"+repository+":"+t)
		if err != nil {
			return err
		}
	}

	return nil
}

package main

import (
	"context"
	"dagger/cuestomize/internal/dagger"
)

// Builds the Cuestomize image
func (m *Cuestomize) Build(
	ctx context.Context,
	// +defaultPath=./
	buildContext *dagger.Directory,
	// +default=""
	platform string,
) *dagger.Container {

	containerOpts := dagger.ContainerOpts{}
	if platform != "" {
		containerOpts.Platform = dagger.Platform(platform)
	}

	// Build stage: compile the Go binary
	builder := m.cuestomizeBuilderContainer(buildContext, containerOpts)

	// now := time.Now().UTC().Format(time.RFC3339)

	distroless := dag.Container(containerOpts)
	// distroless := dag.Container(containerOpts).
	// 	From(DistrolessStaticImage)
	// baseDigest, err := distroless.Rootfs().Digest(ctx)
	// if err != nil {
	// 	// TODO: log error
	// }

	// Final stage: create the runtime container with distroless
	container := distroless.
		// WithAnnotation("org.opencontainers.image.base.name", DistrolessStaticImage).
		// WithAnnotation("org.opencontainers.image.base.digest", baseDigest).
		// WithAnnotation("org.opencontainers.image.created", now).
		// WithAnnotation("/", "https://github.com/Workday/cuestomize").
		// WithAnnotation("org.opencontainers.image.documentation", "https://workday.github.io/cuestomize/").
		// WithAnnotation("org.opencontainers.image.title", "cuestomize").
		// WithAnnotation("org.opencontainers.image.description", "Cuestomize KRM function. K8s package manager (and more) integrating CUE-Lang with Kustomize.").
		WithDirectory("/cue-resources", dag.Directory(), dagger.ContainerWithDirectoryOpts{Owner: "nobody"}).
		WithFile("/usr/local/bin/cuestomize", builder.File("/workspace/cuestomize")).
		WithEntrypoint([]string{"/usr/local/bin/cuestomize"})

	return container
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
		if _, err := m.GolangciLintRun(ctx, buildContext, "", "5m"); err != nil {
			return err
		}

		// tests
		if _, err := m.TestWithCoverage(ctx, buildContext); err != nil {
			return err
		}
	}

	if len(platforms) == 0 {
		platform, err := dag.DefaultPlatform(ctx)
		if err != nil {
			return err
		}
		platforms = append(platforms, string(platform))
	}

	platformVariants := make([]*dagger.Container, 0, len(platforms))
	for _, platform := range platforms {
		container := m.Build(ctx, buildContext, string(platform))
		platformVariants = append(platformVariants, container)
	}

	tags := []string{tag}
	if alsoTagAsLatest {
		tags = append(tags, "latest")
	}
	for _, t := range tags {
		_, err := dag.Container().WithRegistryAuth(registry, username, password).
			Publish(ctx, registry+"/"+repository+":"+t, dagger.ContainerPublishOpts{
				PlatformVariants: platformVariants,
			})
		if err != nil {
			return err
		}
	}

	return nil
}

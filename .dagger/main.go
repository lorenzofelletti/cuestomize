// Cuestomize CI/CD functions

package main

import (
	"dagger/cuestomize/internal/dagger"
)

type Cuestomize struct {
	ldflags string
}

// constructor
func (m *Cuestomize) New(
	// +default=""
	ldflags string,
) *Cuestomize {
	return &Cuestomize{
		ldflags: ldflags,
	}
}

// repoBaseContainer creates a container with the repository files in it and go dependencies installed.
// The working directory is set to `/workspace` and contains the root of the repository.
func (m *Cuestomize) repoBaseContainer(buildContext *dagger.Directory, dirOpts *dagger.ContainerWithDirectoryOpts, containerOpts ...dagger.ContainerOpts) *dagger.Container {
	if dirOpts == nil {
		dirOpts = &DefaultExcludedOpts
	}

	// Create a container to run the tests
	return dag.Container(containerOpts...).
		From(GolangImage).
		WithWorkdir("/workspace").
		WithFile("/workspace/go.mod", buildContext.File("go.mod")).
		WithFile("/workspace/go.sum", buildContext.File("go.sum")).
		WithExec([]string{"go", "mod", "download"}).
		WithDirectory("/workspace", buildContext, *dirOpts)
}

// cuestomizeBuilderContainer returns a container that can be used to build the cuestomize binary.
func (m *Cuestomize) cuestomizeBuilderContainer(buildContext *dagger.Directory, containerOpts ...dagger.ContainerOpts) *dagger.Container {
	buildCmd := []string{"go", "build", "-o", "cuestomize"}
	if m.ldflags != "" {
		buildCmd = append(buildCmd, "-ldflags", m.ldflags)
	}
	buildCmd = append(buildCmd, "main.go")

	return m.repoBaseContainer(buildContext, nil, containerOpts...).
		WithEnvVariable("CGO_ENABLED", "0").
		WithEnvVariable("GO111MODULE", "on").
		WithExec(buildCmd)
}

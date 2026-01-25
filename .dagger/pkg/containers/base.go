package containers

import (
	"dagger/cuestomize/common"
	"dagger/cuestomize/internal/dagger"
)

var dag = dagger.Connect()

var (
	DefaultExcludedOpts = dagger.ContainerWithDirectoryOpts{
		Exclude: []string{
			".go-version", "README.md",
			".vscode", "examples",
			".dagger", "docs",
		},
		Gitignore: true,
	}
)

// repoBaseContainer creates a container with the repository files in it and go dependencies installed.
// The working directory is set to `/workspace` and contains the root of the repository.
func repoBaseContainer(buildContext *dagger.Directory, dirOpts *dagger.ContainerWithDirectoryOpts, containerOpts ...dagger.ContainerOpts) *dagger.Container {
	if dirOpts == nil {
		dirOpts = &DefaultExcludedOpts
	}

	return dag.Container(containerOpts...).
		From(common.GolangImage).
		WithWorkdir("/workspace").
		WithFile("/workspace/go.mod", buildContext.File("go.mod")).
		WithFile("/workspace/go.sum", buildContext.File("go.sum")).
		WithExec([]string{"go", "mod", "download"}).
		WithDirectory("/workspace", buildContext, *dirOpts)
}

// cuestomizeBuilderContainer returns a container that can be used to build the cuestomize binary.
func cuestomizeBuilderContainer(buildContext *dagger.Directory, ldflags string, containerOpts ...dagger.ContainerOpts) *dagger.Container {
	buildCmd := []string{"go", "build", "-o", "cuestomize"}
	if ldflags != "" {
		buildCmd = append(buildCmd, "-ldflags", ldflags)
	}
	buildCmd = append(buildCmd, "main.go")

	return repoBaseContainer(buildContext, nil, containerOpts...).
		WithEnvVariable("CGO_ENABLED", "0").
		WithEnvVariable("GO111MODULE", "on").
		WithExec(buildCmd)
}

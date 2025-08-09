package main

import (
	"context"
	"dagger/cuestomize/internal/dagger"
	"fmt"
)

func (m *Cuestomize) GolangciLintRun(
	ctx context.Context,
	// +defaultPath=./
	src *dagger.Directory,
	// +default="v2.1.6"
	version string,
	// +default="5m"
	timeout string,
) (*dagger.Container, error) {
	image := fmt.Sprintf(GolangciLingImageFmt, version)
	linter := dag.Container().From(image).
		WithMountedDirectory("/src", src).
		WithWorkdir("/src")

	return linter.WithExec([]string{
		"golangci-lint",
		"run",
		"-v",
		fmt.Sprintf("--timeout=%s", timeout),
	}).Sync(ctx)
}

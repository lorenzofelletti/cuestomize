package main

import (
	"context"
	"dagger/cuestomize/internal/dagger"
	"dagger/cuestomize/shared/oci"
	"fmt"
	"os"
	"path"

	"oras.land/oras-go/v2/registry/remote/auth"
)

const (
	CueModuleArtifactType     = "application/vnd.cue.module.v1+json"
	CueModuleFileArtifactType = "application/vnd.cue.modulefile.v1"
)

func (m *Cuestomize) PublishExamples(
	ctx context.Context,
	username string,
	password *dagger.Secret,
	// +defaultPath=./
	buildContext *dagger.Directory,
	// +default="ghcr.io"
	registry string,
	repositoryPrefix string,
	tag string,
	// +default=[]
	platforms []string,
	latest bool,
) error {
	container, err := repoBaseContainer(buildContext, nil).
		WithExec([]string{"go", "generate", "./..."}).Sync(ctx)
	if err != nil {
		return err
	}

	examplesDir := container.Directory("/workspace/examples")

	tempDir, err := os.MkdirTemp("", "examples")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	if _, err := examplesDir.Export(ctx, tempDir); err != nil {
		return err
	}

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return err
	}

	repositoryDirMap := make(map[string]string)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		repoName := repositoryPrefix + "/" + entry.Name()
		repositoryDirMap[repoName] = path.Join(tempDir, entry.Name())
	}

	pwd, err := password.Plaintext(ctx)
	if err != nil {
		return err
	}

	tags := []string{tag}
	if latest {
		tags = append(tags, "latest")
	}

	for repoName, dir := range repositoryDirMap {
		for _, t := range tags {
			_, err := oci.PushDirectoryToOCIRegistry(ctx, repoName, dir, CueModuleArtifactType, t, &auth.Client{
				Credential: auth.StaticCredential(registry, auth.Credential{
					Username: username,
					Password: pwd,
				}),
			})
			if err != nil {
				return fmt.Errorf("failed to push %s to OCI registry: %w", repoName, err)
			}
		}
	}

	return nil
}

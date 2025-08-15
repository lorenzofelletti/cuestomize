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
) ([]string, error) {

	container := m.GoGenerate(ctx, buildContext)

	examplesDir := container.Directory("/workspace/examples")

	tempDir, err := os.MkdirTemp("", "examples")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	if _, err := examplesDir.Export(ctx, tempDir); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return nil, err
	}

	repositoryDirMap := make(map[string]string)

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "*" {
			continue
		}
		repoName := registry + "/" + repositoryPrefix + "-" + entry.Name()
		repositoryDirMap[repoName] = path.Join(tempDir, entry.Name())
	}

	pwd, err := password.Plaintext(ctx)
	if err != nil {
		return nil, err
	}

	tags := []string{tag}
	if latest {
		tags = append(tags, "latest")
	}

	pushedRefs := make([]string, 0, len(repositoryDirMap)*len(tags))
	for repoName, dir := range repositoryDirMap {
		for _, t := range tags {
			_, err := oci.PushDirectoryToOCIRegistry(ctx, repoName, dir, CueModuleArtifactType, t, &auth.Client{
				Credential: auth.StaticCredential(registry, auth.Credential{
					Username: username,
					Password: pwd,
				}),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to push %s to OCI registry: %w", repoName, err)
			}
			pushedRefs = append(pushedRefs, fmt.Sprintf("%s:%s", repoName, t))
		}
	}

	return pushedRefs, nil
}

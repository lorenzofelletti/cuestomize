package oci

import (
	"context"
	"fmt"

	"github.com/Workday/cuestomize/api"
	"github.com/Workday/cuestomize/internal/pkg/fetcher"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

// FetchFromRegistry fetches a CUE module from the remote defined in the config, and places it in the working directory.
func FetchFromRegistry(ctx context.Context, config *api.KRMInput, items []*kyaml.RNode, workingDir string) error {
	client, err := config.GetRemoteClient(items)
	if err != nil {
		return fmt.Errorf("failed to configure remote client: %w", err)
	}
	if err := fetcher.FetchFromOCIRegistry(
		context.TODO(),
		client,
		workingDir,
		config.RemoteModule.Registry,
		config.RemoteModule.Repo,
		config.RemoteModule.Tag,
		config.RemoteModule.PlainHTTP,
	); err != nil {
		return fmt.Errorf("failed to fetch from OCI registry: %w", err)
	}
	return nil
}

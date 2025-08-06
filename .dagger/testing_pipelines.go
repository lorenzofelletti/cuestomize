package main

import (
	"context"
	"dagger/cuestomize/internal/dagger"
	"dagger/cuestomize/shared"
	"fmt"
)

const (
	e2eCredSecretContentFmt = `
export username=%s
export password=%s
`
)

func (m *Cuestomize) UnitTest(
	ctx context.Context,
	// +defaultPath=./
	buildContext *dagger.Directory,
) error {

	// Create a container to run the unit tests
	container := repoBaseContainer(buildContext, nil).
		WithExec([]string{"go", "test", "./..."})

	exitCode, err := container.ExitCode(ctx)
	if err != nil {
		return fmt.Errorf("failed to run unit tests: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("unit tests failed with exit code %d", exitCode)
	}
	return nil
}

func (m *Cuestomize) IntegrationTest(
	ctx context.Context,
	// +defaultPath=./
	buildContext *dagger.Directory,
) error {
	// Setup registryNoAuth without authentication
	registryService, err := setupRegistryServiceNoAuth(ctx)
	if err != nil {
		return fmt.Errorf("failed to start registry service: %w", err)
	}
	defer registryService.Stop(ctx)

	// Setup registryWithAuth with authentication
	username := "registryuser"
	password := "password"
	registryWithAuthService, err := setupRegistryServiceWithAuth(ctx, username, password)
	if err != nil {
		return fmt.Errorf("failed to start registry with auth service: %w", err)
	}
	defer registryWithAuthService.Stop(ctx)

	// Create a container to run the integration tests
	exitCode, err := testContainerWithRegistryServices(
		buildContext, registryService, registryWithAuthService, username, password).
		WithEnvVariable(shared.IntegrationTestingVarName, "true").
		WithExec([]string{"go", "test", "./integration"}).ExitCode(ctx)

	if err != nil {
		return fmt.Errorf("failed to run integration tests: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("integration tests failed with exit code %d", exitCode)
	}
	return nil
}

func (m *Cuestomize) E2E_Test(
	ctx context.Context,
	// +defaultPath=./
	buildContext *dagger.Directory,
) error {
	// build cuestomize
	cuestomize, err := m.Build(ctx, buildContext)
	if err != nil {
		return fmt.Errorf("failed to build cuestomize: %w", err)
	}

	testdataDir := buildContext.Directory("e2e/testdata")

	// setup registryNoAuth without authentication
	registryService, err := setupRegistryServiceNoAuth(ctx)
	if err != nil {
		return fmt.Errorf("failed to start registry service: %w", err)
	}
	defer registryService.Stop(ctx)

	// push cuestomize to registry
	// execOpts := dagger.ContainerWithExecOpts{
	// 	InsecureRootCapabilities: true,
	// }
	certCache := dag.CacheVolume("node")
	dockerState := dag.CacheVolume("docker-state")

	// create the container with Dind with the docker daemon we will be using
	// dockerCertsCA := dag.CacheVolume("docker-certs-ca")
	// dockerCertsClient := dag.CacheVolume("docker-certs-client")

	// docker, err := dag.Container().
	// 	From("docker:dind").
	// 	// From("docker:23.0.4-dind-rootless").
	// 	WithMountedCache("/certs/ca", dockerCertsCA, dagger.ContainerWithMountedCacheOpts{
	// 		Owner: "nobody",
	// 	}).
	// 	WithMountedCache("/certs/client", dockerCertsClient).
	// 	WithEnvVariable("DOCKER_TLS_CERTDIR", "/certs").
	// 	WithExec(nil, dagger.ContainerWithExecOpts{
	// 		InsecureRootCapabilities: true,
	// 		UseEntrypoint:            true,
	// 	}).
	// 	WithExposedPort(2376).AsService().Start(ctx)
	docker, err := dag.Container().
		From("docker:dind").
		WithExposedPort(2376).
		WithMountedCache("/var/lib/docker", dockerState, dagger.ContainerWithMountedCacheOpts{
			Sharing: dagger.CacheSharingModePrivate,
		}).
		WithMountedCache("/certs", certCache).
		AsService(dagger.ContainerAsServiceOpts{
			UseEntrypoint:            true,
			InsecureRootCapabilities: true,
		}).
		Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start docker service: %w", err)
	}
	defer docker.Stop(ctx)
	customizeTar := cuestomize.AsTarball()
	if _, err := dag.Container().From("docker:latest").WithFile("/cuestomize.tar", customizeTar).
		WithServiceBinding("docker", docker).
		WithServiceBinding("registry", registryService).
		WithMountedCache("/certs", certCache).
		WithEnvVariable("DOCKER_HOST", "tcp://docker:2376").
		WithEnvVariable("DOCKER_TLS_CERTDIR", "/certs").
		WithEnvVariable("DOCKER_CERT_PATH", "/certs/client").
		WithEnvVariable("DOCKER_TLS_VERIFY", "1").
		WithExec([]string{"docker", "load", "-i", "/cuestomize.tar"}).
		WithExec([]string{"docker", "tag", "cuestomize", "registry:5000/cuestomize:latest"}).
		WithExec([]string{"docker", "push", "registry:5000/cuestomize:latest"}).
		Sync(ctx); err != nil {
		return fmt.Errorf("failed to push cuestomize to registry: %w", err)
	}

	// setup registryWithAuth with authentication
	username := "registryuser"
	password := "password"
	registryWithAuthService, err := setupRegistryServiceWithAuth(ctx, username, password)
	if err != nil {
		return fmt.Errorf("failed to start registry with auth service: %w", err)
	}
	defer registryWithAuthService.Stop(ctx)

	// e2e setup (pushing cue module to registries)
	if _, err := repoBaseContainer(buildContext, nil).
		WithServiceBinding("registry", registryService).
		WithServiceBinding("registry_auth", registryWithAuthService).
		WithEnvVariable(shared.RegistryHostVarName, "registry:5000").
		WithEnvVariable(shared.RegistryAuthHostVarName, "registry_auth:5000").
		WithExec([]string{"go", "run", "./e2e/main.go"}).Sync(ctx); err != nil {
		return fmt.Errorf("failed to run e2e tests: %w", err)
	}

	// run e2e tests
	// TODO: save output to file and extract it for comparison
	kustomize, err := dag.Container().From("registry.k8s.io/kustomize/kustomize:v5.6.0").
		WithServiceBinding("registry", registryService).
		WithServiceBinding("registry_auth", registryWithAuthService).
		WithDirectory("/testdata", testdataDir).
		WithNewFile(
			"/testdata/kustomize-auth/.env.secret",
			fmt.Sprintf(e2eCredSecretContentFmt, username, password),
		).
		WithExec([]string{"kustomize", "build", "--enable-alpha-plugins", "--network", "/testdata/kustomize"}).Sync(ctx)
	if err != nil {
		return fmt.Errorf("kustomize with no auth e2e failed: %w", err)
	}

	if _, err := kustomize.WithExec([]string{"kustomize", "build", "--enable-alpha-plugins", "--network", "/testdata/kustomize-auth"}).Sync(ctx); err != nil {
		return fmt.Errorf("kustomize with auth e2e failed: %w", err)
	}

	return nil
}

func (m *Cuestomize) RunTests(
	ctx context.Context,
	// +defaultPath=./
	buildContext *dagger.Directory,
) error {
	if err := m.UnitTest(ctx, buildContext); err != nil {
		return fmt.Errorf("unit tests failed: %w", err)
	}
	if err := m.IntegrationTest(ctx, buildContext); err != nil {
		return fmt.Errorf("integration tests failed: %w", err)
	}
	return nil
}

func setupRegistryServiceNoAuth(ctx context.Context) (*dagger.Service, error) {
	registryNoAuth := dag.Container().From(RegistryImage).WithExposedPort(5000)
	return registryNoAuth.AsService().Start(ctx)
}

func setupRegistryServiceWithAuth(ctx context.Context, username, password string) (*dagger.Service, error) {
	htpasswdUtil := dag.Container().From("httpd:2.4").
		WithExec([]string{"htpasswd", "-Bbc", "/tmp/htpasswd", username, password})
	htpasswdFile := htpasswdUtil.File("/tmp/htpasswd")
	registryWithAuth := dag.Container().From(RegistryImage).
		WithFile("/auth/htpasswd", htpasswdFile).
		WithExposedPort(5000).
		WithEnvVariable("REGISTRY_AUTH", "htpasswd").
		WithEnvVariable("REGISTRY_AUTH_HTPASSWD_PATH", "/auth/htpasswd").
		WithEnvVariable("REGISTRY_AUTH_HTPASSWD_REALM", "Dagger Registry")
	return registryWithAuth.AsService().Start(ctx)
}

func testContainerWithRegistryServices(buildContext *dagger.Directory, registryService, registryWithAuthService *dagger.Service, username, password string) *dagger.Container {
	return repoBaseContainer(buildContext, nil).
		WithServiceBinding("registry", registryService).
		WithServiceBinding("registry_auth", registryWithAuthService).
		WithEnvVariable(shared.RegistryHostVarName, "registry:5000").
		WithEnvVariable(shared.RegistryAuthHostVarName, "registry_auth:5000").
		WithEnvVariable(shared.RegistryUsernameVarName, username).
		WithEnvVariable(shared.RegistryPasswordVarName, password)
}

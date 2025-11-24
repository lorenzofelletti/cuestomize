// Package registryauth provides utilities for configuring authentication
// credentials for OCI registry clients, supporting both Kubernetes Secret
// and environment variable-based credential sources.
package registryauth

import (
	"fmt"
	"os"
	"path"

	corev1 "k8s.io/api/core/v1"
	"oras.land/oras-go/v2/registry/remote/auth"
)

const (
	// UsernameEnvVar is the environment variable name for the username to use for registry authentication.
	UsernameEnvVar = "REGISTRY_USERNAME"
	// PasswordEnvVar is the environment variable name for the password to use for registry authentication.
	PasswordEnvVar = "REGISTRY_PASSWORD"
	// AccessTokenEnvVar is the environment variable name for the access token to use for registry authentication.
	AccessTokenEnvVar = "REGISTRY_ACCESS_TOKEN"
	// RefreshTokenEnvVar is the environment variable name for the refresh token to use for registry authentication.
	RefreshTokenEnvVar = "REGISTRY_REFRESH_TOKEN"
)

// ConfigureClient configures a remote client to fetch from the specified registry, with authentication if
// any is found.
func ConfigureClient(registry string, authSecret *corev1.Secret) (*auth.Client, error) {
	client := auth.DefaultClient

	if authSecret != nil && authSecret.Type == corev1.SecretTypeDockerConfigJson {
		err := copyDockerConfigJson(authSecret)
		if err != nil {
			return nil, err
		}
	}

	creds, err := configureAuth(authSecret)
	if err != nil {
		return nil, err
	}

	if creds != nil {
		client.Credential = auth.StaticCredential(registry, *creds)
	}
	return client, nil
}

// TODO(dev): test dockerconfigjson passing
func copyDockerConfigJson(authSecret *corev1.Secret) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	dockerDir := path.Join(homeDir, ".docker")
	err = os.Mkdir(dockerDir, 0755)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create docker config directory: %w", err)
	}

	dockerConfigJsonPath := path.Join(dockerDir, "config.json")
	err = os.WriteFile(dockerConfigJsonPath, authSecret.Data[corev1.DockerConfigJsonKey], 0600)
	if err != nil {
		return fmt.Errorf("failed to write docker config file: %w", err)
	}
	return nil
}

// configureAuth configures authentication based on the provided authSecret or environment variables.
// If no authentication is found, it returns nil, nil (no error).
func configureAuth(authSecret *corev1.Secret) (*auth.Credential, error) {
	if authSecret == nil {
		return getAuthFromEnv(), nil
	}

	creds := &auth.Credential{}

	for k, v := range authSecret.Data {
		switch k {
		case "username", UsernameEnvVar:
			creds.Username = string(v)
		case "password", PasswordEnvVar:
			creds.Password = string(v)
		case "accessToken", AccessTokenEnvVar:
			creds.AccessToken = string(v)
		case "refreshToken", RefreshTokenEnvVar:
			creds.RefreshToken = string(v)
		}
	}

	return creds, nil
}

func getAuthFromEnv() *auth.Credential {
	username := os.Getenv(UsernameEnvVar)
	password := os.Getenv(PasswordEnvVar)
	accessToken := os.Getenv(AccessTokenEnvVar)
	refreshToken := os.Getenv(RefreshTokenEnvVar)

	if username == "" && password == "" && accessToken == "" && refreshToken == "" {
		return nil
	}

	return &auth.Credential{
		Username:     os.Getenv(UsernameEnvVar),
		Password:     os.Getenv(PasswordEnvVar),
		AccessToken:  os.Getenv(AccessTokenEnvVar),
		RefreshToken: os.Getenv(RefreshTokenEnvVar),
	}
}

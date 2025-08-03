package registryauth

import (
	"os"

	corev1 "k8s.io/api/core/v1"
	"oras.land/oras-go/v2/registry/remote/auth"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	UsernameEnvVar     = "REGISTRY_USERNAME"
	PasswordEnvVar     = "REGISTRY_PASSWORD"
	AccessTokenEnvVar  = "REGISTRY_ACCESS_TOKEN"
	RefreshTokenEnvVar = "REGISTRY_REFRESH_TOKEN"
)

func ConfigureAuth(authSecret *corev1.Secret, items []*kyaml.RNode) (*auth.Credential, error) {
	if authSecret == nil {
		return GetAuthFromEnv(), nil
	}

	creds := &auth.Credential{}

	for k, v := range authSecret.Data {
		switch k {
		case "username":
			creds.Username = string(v)
		case "password":
			creds.Password = string(v)
		case "accessToken":
			creds.AccessToken = string(v)
		case "refreshToken":
			creds.RefreshToken = string(v)
		}
	}

	return creds, nil
}

func GetAuthFromEnv() *auth.Credential {
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

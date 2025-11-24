package registryauth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"oras.land/oras-go/v2/registry/remote/auth"
)

func TestConfigureClient(t *testing.T) {
	testCases := []struct {
		name          string
		registry      string
		authSecret    *corev1.Secret
		envVars       map[string]string
		expectedCreds *auth.Credential
		expectError   bool
	}{
		{
			name:          "No auth",
			registry:      "my-registry",
			authSecret:    nil,
			envVars:       nil,
			expectedCreds: nil,
			expectError:   false,
		},
		{
			name:     "Auth secret with basic auth",
			registry: "my-registry",
			authSecret: &corev1.Secret{
				Data: map[string][]byte{
					"username": []byte("user"),
					"password": []byte("pass"),
				},
			},
			envVars:       nil,
			expectedCreds: &auth.Credential{Username: "user", Password: "pass"},
			expectError:   false,
		},
		{
			name:       "Auth from env vars",
			registry:   "my-registry",
			authSecret: nil,
			envVars: map[string]string{
				UsernameEnvVar: "user",
				PasswordEnvVar: "pass",
			},
			expectedCreds: &auth.Credential{Username: "user", Password: "pass"},
			expectError:   false,
		},
	}

	// remove any envvar value from the test host environment that may interfere with the test
	cleanupConflictingEnvVars(t)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setEnvVars(t, tc.envVars)

			client, err := ConfigureClient(tc.registry, tc.authSecret)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.expectedCreds == nil {
					assert.Nil(t, client.Credential)
					return
				}

				staticCreds, err := client.Credential(t.Context(), tc.registry)
				assert.NoError(t, err)
				assert.Equal(t, *tc.expectedCreds, staticCreds)
			}
		})
	}
}

func TestConfigureAuth(t *testing.T) {
	testCases := []struct {
		name          string
		authSecret    *corev1.Secret
		envVars       map[string]string
		expectedCreds *auth.Credential
	}{
		{
			name:          "No auth",
			authSecret:    nil,
			envVars:       nil,
			expectedCreds: nil,
		},
		{
			name: "Auth secret with basic auth",
			authSecret: &corev1.Secret{
				Data: map[string][]byte{
					"username": []byte("user"),
					"password": []byte("pass"),
				},
			},
			envVars:       nil,
			expectedCreds: &auth.Credential{Username: "user", Password: "pass"},
		},
		{
			name: "Auth secret with token auth",
			authSecret: &corev1.Secret{
				Data: map[string][]byte{
					"accessToken":  []byte("access-token"),
					"refreshToken": []byte("refresh-token"),
				},
			},
			envVars:       nil,
			expectedCreds: &auth.Credential{AccessToken: "access-token", RefreshToken: "refresh-token"},
		},
		{
			name:       "Auth from env vars",
			authSecret: nil,
			envVars: map[string]string{
				UsernameEnvVar: "user",
				PasswordEnvVar: "pass",
			},
			expectedCreds: &auth.Credential{Username: "user", Password: "pass"},
		},
	}

	// remove any envvar value from the test host environment that may interfere with the test
	cleanupConflictingEnvVars(t)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setEnvVars(t, tc.envVars)
			creds, err := configureAuth(tc.authSecret)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedCreds, creds)
		})
	}
}

func TestGetAuthFromEnv(t *testing.T) {
	testCases := []struct {
		name          string
		envVars       map[string]string
		expectedCreds *auth.Credential
	}{
		{
			name:          "No env vars",
			envVars:       nil,
			expectedCreds: nil,
		},
		{
			name: "Username and password",
			envVars: map[string]string{
				UsernameEnvVar: "user",
				PasswordEnvVar: "pass",
			},
			expectedCreds: &auth.Credential{Username: "user", Password: "pass"},
		},
		{
			name: "Access and refresh tokens",
			envVars: map[string]string{
				AccessTokenEnvVar:  "access-token",
				RefreshTokenEnvVar: "refresh-token",
			},
			expectedCreds: &auth.Credential{AccessToken: "access-token", RefreshToken: "refresh-token"},
		},
	}

	// remove any envvar value from the test host environment that may interfere with the test
	cleanupConflictingEnvVars(t)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setEnvVars(t, tc.envVars)
			creds := getAuthFromEnv()
			assert.Equal(t, tc.expectedCreds, creds)
		})
	}
}

func cleanupConflictingEnvVars(t *testing.T) {
	t.Helper()
	setEnvVars(t, map[string]string{
		UsernameEnvVar:     "",
		"username":         "",
		PasswordEnvVar:     "",
		"password":         "",
		AccessTokenEnvVar:  "",
		"accessToken":      "",
		RefreshTokenEnvVar: "",
		"refreshToken":     "",
	})
}

func setEnvVars(t *testing.T, envVars map[string]string) {
	t.Helper()
	for k, v := range envVars {
		t.Setenv(k, v)
	}
}

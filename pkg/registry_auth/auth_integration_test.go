package registryauth

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Workday/cuestomize/internal/pkg/testhelpers"
	"github.com/stretchr/testify/assert"
)

func TestCopyDockerConfigJson(t *testing.T) {
	if os.Getenv(testhelpers.IntegrationTestingVarName) != "true" {
		t.Skipf("Skipping test because %s is not set", testhelpers.IntegrationTestingVarName)
	}
}

func dockerConfigJson(t *testing.T, registry, username, password string) []byte {
	t.Helper()
	config := map[string]interface{}{
		"auths": map[string]interface{}{
			registry: map[string]string{
				"username": username,
				"password": password,
			},
		},
	}
	data, err := json.Marshal(config)
	assert.NoError(t, err)
	return data
}

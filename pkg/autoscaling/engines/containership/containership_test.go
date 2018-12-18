package containership

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// fakeAutoscalingEngine creates a fake autoscaling engine that can be used for
// testing containership autoscaling engine functions
func fakeAutoscalingEngine() *Engine {
	return &Engine{
		name: "containership",
		config: &cloudConfig{
			Address:         "https://provision-test.containership.io",
			TokenEnvVarName: "TOKEN_ENV_VAR",
			OrganizationID:  "organization-uuid",
			ClusterID:       "cluster-uuid",
		},
	}
}

func TestNewClient(t *testing.T) {
	name := "containership"
	configuration := map[string]string{
		"address":         "https://provision-test.containership.io",
		"tokenEnvVarName": "TOKEN_ENV_VAR",
		"organizationID":  "organization-uuid",
		"clusterID":       "cluster-uuid",
	}

	c, err := NewClient(name, configuration)
	assert.Error(t, err, "Testing that an error is returned when the token environment variable is not defined")

	os.Setenv(configuration["tokenEnvVarName"], "token")
	c, err = NewClient(name, configuration)
	assert.NoError(t, err, "Testing that no error is returned when client is successfully created")
	assert.NotNil(t, c, "Testing that client is not nil when successfully created")
	os.Unsetenv(configuration["tokenEnvVarName"])
}

func TestName(t *testing.T) {
	c := fakeAutoscalingEngine()
	assert.Equal(t, c.name, c.Name())
}

func TestSetTargetNodeCount(t *testing.T) {
	c := fakeAutoscalingEngine()

	emptyLabels := make(map[string]string, 0)

	result, err := c.SetTargetNodeCount(emptyLabels, -1, "")
	assert.Error(t, err, "Testing that an error is returned if there is a request to scale below 0")
	assert.False(t, result)

	result, err = c.SetTargetNodeCount(emptyLabels, 0, "")
	assert.Error(t, err, "Testing that an error is returned if there is a request to scale below 0")
	assert.False(t, result)

	// TODO add a test checking that providing an empty string for strategy is ok
	// (containership engine should default) when Containership Cloud client is easily
	// mockable
}

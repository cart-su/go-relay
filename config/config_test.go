package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigLoading(t *testing.T) {
	// create a temporary config file
	configJSON := []byte(`{
        "listen_ip": "127.0.0.1",
        "port": 8080
    }`)

	err := os.WriteFile("config.json", configJSON, 0644)
	assert.NoError(t, err)
	defer os.Remove("config.json")

	// reset variable
	Config = Configuration{}

	LoadConfig()

	// verify config was loaded correctly
	assert.Equal(t, "127.0.0.1", Config.ListenRange)
	assert.Equal(t, 8080, Config.Port)
}

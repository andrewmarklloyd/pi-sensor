package test

import (
	"testing"

	"github.com/andrewmarklloyd/pi-app-deployer/api/v1/manifest"
	"github.com/stretchr/testify/assert"
)

func Test_Manifest(t *testing.T) {
	_, err := manifest.GetManifest("../.pi-app-deployer.yaml", "pi-sensor-agent")
	assert.NoError(t, err)
}

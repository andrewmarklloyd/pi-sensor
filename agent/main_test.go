package main

import (
	"testing"

	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/config"
	"github.com/stretchr/testify/assert"
)

func Test_Status(t *testing.T) {
	tmpStatusFile := "/tmp/.pi-sensor-status"
	err := writeStatus(tmpStatusFile, config.OPEN)
	assert.NoError(t, err)

	status, err := getLastStatus(tmpStatusFile)
	assert.NoError(t, err)
	assert.Equal(t, "OPEN", status)

	status, err = getLastStatus("bogus-file")
	assert.Error(t, err)
	assert.Equal(t, "", status)
}

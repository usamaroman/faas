package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewClient(t *testing.T) {
	client, err := NewClient("unix:///Users/romanchechyotkin/.colima/docker.sock")
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

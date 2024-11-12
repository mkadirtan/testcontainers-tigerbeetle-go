package tigerbeetle_test

import (
	"context"
	"testing"

	tigerbeetle "github.com/mkadirtan/testcontainers-tigerbeetle-go"
	"github.com/stretchr/testify/assert"
)

func TestTigerBeetleContainer(t *testing.T) {
	ctx := context.Background()

	tbContainer, err := tigerbeetle.RunContainer(ctx)
	assert.NoError(t, err)
	defer func() { _ = tbContainer.Terminate(ctx) }()

	// Add test logic to interact with TigerBeetle
	assert.NotEmpty(t, tbContainer.Host)
	assert.NotEmpty(t, tbContainer.Port)
}

package tigerbeetle_test

import (
	"context"
	"testing"

	tigerbeetle "github.com/mkadirtan/testcontainers-tigerbeetle-go"
	"github.com/stretchr/testify/require"
	tigerbeetle_go "github.com/tigerbeetle/tigerbeetle-go"
	"github.com/tigerbeetle/tigerbeetle-go/pkg/types"
)

func TestTigerBeetleContainer(t *testing.T) {
	ctx := context.Background()

	tbContainer, err := tigerbeetle.Run(ctx, tigerbeetle.DefaultImage)
	require.NoError(t, err)
	defer func() { _ = tbContainer.Terminate(ctx) }()

	address, err := tbContainer.Address(ctx)
	require.NoError(t, err)

	tbClient, err := tigerbeetle_go.NewClient(types.ToUint128(0), []string{address})
	require.NoError(t, err)

	_, err = tbClient.CreateAccounts([]types.Account{
		{
			ID:     types.ID(),
			Ledger: 1,
			Code:   1,
		},
	})
	require.NoError(t, err)
}

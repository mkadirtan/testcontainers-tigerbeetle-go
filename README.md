# testcontainers-tigerbeetle-go

Go library for **[Tigerbeetle](https://tigerbeetle.com/) integration testing via
[Testcontainers](https://testcontainers.com/)**.

## Install

Use `go get` to install the latest version of the library.

```bash
go get -u github.com/mkadirtan/testcontainers-tigerbeetle-go@latest
```

## Usage

```go
package tigerbeetle_test

import (
	"context"
	"testing"

	tigerbeetle "github.com/mkadirtan/testcontainers-tigerbeetle-go"
	"github.com/stretchr/testify/require"
	tigerbeetle_go "github.com/tigerbeetle/tigerbeetle-go"
	"github.com/tigerbeetle/tigerbeetle-go/pkg/types"
)

func TestTigerbeetleContainer(t *testing.T) {
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
```
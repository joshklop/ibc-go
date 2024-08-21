package v7

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"cosmossdk.io/store"

	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// ClientKeeper expected IBC client keeper
type ClientKeeper interface {
	GetClientState(ctx sdk.Context, clientID string) (exported.ClientState, bool)
	SetClientState(ctx sdk.Context, clientID string, clientState exported.ClientState)
	ClientStore(ctx sdk.Context, clientID string) store.KVStore
	CreateLocalhostClient(ctx sdk.Context) error
}

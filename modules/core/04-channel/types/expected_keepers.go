package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"cosmossdk.io/store"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"

	connectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

type KeyValueGenFunc func(*MsgMultihopProofs, *connectiontypes.ConnectionEnd) (string, []byte, error)
type KeyGenFunc func(*MsgMultihopProofs, *connectiontypes.ConnectionEnd) (string, error)

// ClientKeeper expected account IBC client keeper
type ClientKeeper interface {
	GetClientStatus(ctx sdk.Context, clientState exported.ClientState, clientID string) exported.Status
	GetClientState(ctx sdk.Context, clientID string) (exported.ClientState, bool)
	GetClientConsensusState(ctx sdk.Context, clientID string, height exported.Height) (exported.ConsensusState, bool)
	ClientStore(ctx sdk.Context, clientID string) store.KVStore
}

// ConnectionKeeper expected account IBC connection keeper
type ConnectionKeeper interface {
	GetConnection(ctx sdk.Context, connectionID string) (connectiontypes.ConnectionEnd, bool)
	GetTimestampAtHeight(
		ctx sdk.Context,
		connection connectiontypes.ConnectionEnd,
		height exported.Height,
	) (uint64, error)
	VerifyChannelState(
		ctx sdk.Context,
		connection exported.ConnectionI,
		height exported.Height,
		proof []byte,
		portID,
		channelID string,
		channel exported.ChannelI,
	) error
	VerifyPacketCommitment(
		ctx sdk.Context,
		connection exported.ConnectionI,
		height exported.Height,
		proof []byte,
		portID,
		channelID string,
		sequence uint64,
		commitmentBytes []byte,
	) error
	VerifyPacketAcknowledgement(
		ctx sdk.Context,
		connection exported.ConnectionI,
		height exported.Height,
		proof []byte,
		portID,
		channelID string,
		sequence uint64,
		acknowledgement []byte,
	) error
	VerifyPacketReceiptAbsence(
		ctx sdk.Context,
		connection exported.ConnectionI,
		height exported.Height,
		proof []byte,
		portID,
		channelID string,
		sequence uint64,
	) error
	VerifyNextSequenceRecv(
		ctx sdk.Context,
		connection exported.ConnectionI,
		height exported.Height,
		proof []byte,
		portID,
		channelID string,
		nextSequenceRecv uint64,
	) error
	VerifyMultihopMembership(
		ctx sdk.Context,
		connection exported.ConnectionI,
		height exported.Height,
		proof []byte,
		connectionHops []string,
		kvGenerator KeyValueGenFunc,
	) error
	VerifyMultihopNonMembership(
		ctx sdk.Context,
		connection exported.ConnectionI,
		height exported.Height,
		proof []byte,
		connectionHops []string,
		keyGenerator KeyGenFunc,
	) error
}

// PortKeeper expected account IBC port keeper
type PortKeeper interface {
	Authenticate(ctx sdk.Context, key *capabilitytypes.Capability, portID string) bool
}

// VibcKeeper is the expected keeper interface for the virtual ibc module
type VibcKeeper interface {
	// Ensure a virtual port is bound to the vibc module
	EnsurePortBound(ctx sdk.Context, portID string) error
}

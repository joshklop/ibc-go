package multihop

import (
	"fmt"
	"math"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	connectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	"cosmossdk.io/store"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	tmclient "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
)

// VerifyDelayPeriodPassed will ensure that at least delayTimePeriod amount of time and delayBlockPeriod number of blocks have passed
// since consensus state was submitted before allowing verification to continue.
func VerifyDelayPeriodPassed(
	ctx sdk.Context,
	store store.KVStore,
	proofHeight exported.Height,
	timeDelay uint64,
	expectedTimePerBlock uint64,
) error {
	// get time and block delays
	blockDelay := getBlockDelay(ctx, timeDelay, expectedTimePerBlock)
	return tmclient.VerifyDelayPeriodPassed(ctx, store, proofHeight, timeDelay, blockDelay)
}

// getBlockDelay calculates the block delay period from the time delay of the connection
// and the maximum expected time per block.
func getBlockDelay(ctx sdk.Context, timeDelay uint64, expectedTimePerBlock uint64) uint64 {
	// expectedTimePerBlock should never be zero, however if it is then return a 0 block delay for safety
	// as the expectedTimePerBlock parameter was not set.
	if expectedTimePerBlock == 0 {
		return 0
	}
	return uint64(math.Ceil(float64(timeDelay) / float64(expectedTimePerBlock)))
}

// VerifyMultihopMembership verifies a multihop proof. A nil value indicates a non-inclusion proof (proof of absence).
func VerifyMultihopMembership(
	cdc codec.BinaryCodec,
	consensusState exported.ConsensusState,
	connectionHops []string,
	proofs *channeltypes.MsgMultihopProofs,
	prefix exported.Prefix,
	key string,
	value []byte,
) error {

	// verify proof lengths
	if len(proofs.ConnectionProofs) < 1 || len(proofs.ConsensusProofs) < 1 {
		return fmt.Errorf("the number of connection (%d) and consensus (%d) proofs must be > 0",
			len(proofs.ConnectionProofs), len(proofs.ConsensusProofs))
	}

	if len(proofs.ConsensusProofs) != len(proofs.ConnectionProofs) {
		return fmt.Errorf("the number of connection (%d) and consensus (%d) proofs must be equal",
			len(proofs.ConnectionProofs), len(proofs.ConsensusProofs))
	}

	// verify connection states and ordering
	if err := verifyConnectionStates(cdc, proofs.ConnectionProofs, connectionHops); err != nil {
		return err
	}

	// verify intermediate consensus and connection states from destination --> source
	if err := verifyIntermediateStateProofs(cdc, consensusState, proofs.ConsensusProofs, proofs.ConnectionProofs); err != nil {
		return fmt.Errorf("failed to verify consensus state proof: %w", err)
	}

	// verify the keyproof on source chain's consensus state.
	return verifyKeyValueMembership(cdc, proofs, prefix, key, value)
}

// VerifyMultihopNonMembership verifies a multihop proof. A nil value indicates a non-inclusion proof (proof of absence).
func VerifyMultihopNonMembership(
	cdc codec.BinaryCodec,
	consensusState exported.ConsensusState,
	connectionHops []string,
	proofs *channeltypes.MsgMultihopProofs,
	prefix exported.Prefix,
	key string,
) error {

	// verify proof lengths
	if len(proofs.ConnectionProofs) < 1 || len(proofs.ConsensusProofs) < 1 {
		return fmt.Errorf("the number of connection (%d) and consensus (%d) proofs must be > 0",
			len(proofs.ConnectionProofs), len(proofs.ConsensusProofs))
	}

	if len(proofs.ConsensusProofs) != len(proofs.ConnectionProofs) {
		return fmt.Errorf("the number of connection (%d) and consensus (%d) proofs must be equal",
			len(proofs.ConnectionProofs), len(proofs.ConsensusProofs))
	}

	// verify connection states and ordering
	if err := verifyConnectionStates(cdc, proofs.ConnectionProofs, connectionHops); err != nil {
		return err
	}

	// verify intermediate consensus and connection states from destination --> source
	if err := verifyIntermediateStateProofs(cdc, consensusState, proofs.ConsensusProofs, proofs.ConnectionProofs); err != nil {
		return fmt.Errorf("failed to verify consensus state proof: %w", err)
	}

	// verify the keyproof on source chain's consensus state.
	return verifyKeyNonMembership(cdc, proofs, prefix, key)
}

// verifyConnectionStates verifies that the provided connections match the connectionHops field of the channel and are in OPEN state
func verifyConnectionStates(
	cdc codec.BinaryCodec,
	connectionProofData []*channeltypes.MultihopProof,
	connectionHops []string,
) error {
	if len(connectionProofData) != len(connectionHops)-1 {
		return sdkerrors.Wrapf(connectiontypes.ErrInvalidLengthConnection,
			"connectionHops length (%d) must match the connectionProofData length (%d)",
			len(connectionHops)-1, len(connectionProofData))
	}

	// check all connections are in OPEN state and that the connection IDs match and are in the right order
	for i, connData := range connectionProofData {
		var connectionEnd connectiontypes.ConnectionEnd
		if err := cdc.Unmarshal(connData.Value, &connectionEnd); err != nil {
			return err
		}

		// Verify the rest of the connectionHops (first hop already verified)
		// 1. check the connectionHop values match the proofs and are in the same order.
		parts := strings.Split(connData.PrefixedKey.GetKeyPath()[len(connData.PrefixedKey.KeyPath)-1], "/")
		if parts[len(parts)-1] != connectionHops[i+1] {
			return sdkerrors.Wrapf(
				connectiontypes.ErrConnectionPath,
				"connectionHops (%s) does not match connection proof hop (%s) for hop %d",
				connectionHops[i+1], parts[len(parts)-1], i)
		}

		// 2. check that the connectionEnd's are in the OPEN state.
		if connectionEnd.GetState() != int32(connectiontypes.OPEN) {
			return sdkerrors.Wrapf(
				connectiontypes.ErrInvalidConnectionState,
				"connection state is not OPEN for connectionID=%s (got %s)",
				connectionEnd.Counterparty.ConnectionId,
				connectiontypes.State(connectionEnd.GetState()).String(),
			)
		}
	}
	return nil
}

// verifyIntermediateStateProofs verifies the intermediate consensus, connection, client states in the multi-hop proof.
func verifyIntermediateStateProofs(
	cdc codec.BinaryCodec,
	consensusStateI exported.ConsensusState,
	consensusProofs []*channeltypes.MultihopProof,
	connectionProofs []*channeltypes.MultihopProof,
) error {
	// trusted consensusState on verifier
	consensusState, ok := consensusStateI.(*tmclient.ConsensusState)
	if !ok {
		return fmt.Errorf("expected consensus state to be tendermint consensus state, got: %T", consensusStateI)
	}

	var connectionEnd connectiontypes.ConnectionEnd
	var expectedClientID string
	for i := 0; i < len(consensusProofs); i++ {
		consensusProof := consensusProofs[i]
		connectionProof := connectionProofs[i]

		// verify consensus state client id matches previous connection counterparty client id
		if len(expectedClientID) > 0 {
			if len(consensusProof.PrefixedKey.KeyPath) < 2 {
				return fmt.Errorf(
					"invalid consensus proof key path length: %d",
					len(consensusProof.PrefixedKey.KeyPath),
				)
			}
			parts := strings.Split(consensusProof.PrefixedKey.KeyPath[1], "/")
			if len(parts) < 2 {
				return fmt.Errorf("invalid consensus proof key path: %s", consensusProof.PrefixedKey.KeyPath[1])
			}
			if parts[1] != expectedClientID {
				return fmt.Errorf(
					"consensus state client id (%s) does not match expected client id (%s)",
					parts[1],
					expectedClientID,
				)
			}
		}

		// prove consensus state
		var proof commitmenttypes.MerkleProof
		if err := cdc.Unmarshal(consensusProof.Proof, &proof); err != nil {
			return fmt.Errorf("failed to unmarshal consensus state proof: %w", err)
		}
		if err := proof.VerifyMembership(
			commitmenttypes.GetSDKSpecs(),
			consensusState.GetRoot(),
			*consensusProof.PrefixedKey,
			consensusProof.Value,
		); err != nil {
			return fmt.Errorf("failed to verify consensus proof: %w", err)
		}

		// prove connection state
		proof.Reset()
		if err := cdc.Unmarshal(connectionProof.Proof, &proof); err != nil {
			return fmt.Errorf("failed to unmarshal connection state proof: %w", err)
		}
		if err := proof.VerifyMembership(
			commitmenttypes.GetSDKSpecs(),
			consensusState.GetRoot(),
			*connectionProof.PrefixedKey,
			connectionProof.Value,
		); err != nil {
			return fmt.Errorf("failed to verify connection proof: %w", err)
		}

		// determine the next expected client id
		if err := cdc.Unmarshal(connectionProof.Value, &connectionEnd); err != nil {
			return fmt.Errorf("failed to unmarshal connection end: %w", err)
		}
		expectedClientID = connectionEnd.ClientId

		// set the next consensus state
		if err := cdc.UnmarshalInterface(consensusProof.Value, &consensusStateI); err != nil {
			return fmt.Errorf("failed to unpack consesnsus state: %w", err)
		}
		consensusState, ok = consensusStateI.(*tmclient.ConsensusState)
		if !ok {
			return fmt.Errorf("expected consensus state to be tendermint consensus state, got: %T", consensusStateI)
		}
	}
	return nil
}

// verifyKeyValueMembership verifies a multihop membership proof including all intermediate state proofs.
func verifyKeyValueMembership(
	cdc codec.BinaryCodec,
	proofs *channeltypes.MsgMultihopProofs,
	prefix exported.Prefix,
	key string,
	value []byte,
) error {

	// no keyproof provided, nothing to verify
	if proofs.KeyProof == nil {
		return nil
	}

	prefixedKey, err := commitmenttypes.ApplyPrefix(prefix, commitmenttypes.NewMerklePath(key))
	if err != nil {
		return err
	}

	var keyProof commitmenttypes.MerkleProof
	if err := cdc.Unmarshal(proofs.KeyProof.Proof, &keyProof); err != nil {
		return fmt.Errorf("failed to unmarshal key proof: %w", err)
	}

	var consensusStateI exported.ConsensusState
	index := uint32(len(proofs.ConsensusProofs)) - proofs.KeyProofIndex - 1
	if err := cdc.UnmarshalInterface(proofs.ConsensusProofs[index].Value, &consensusStateI); err != nil {
		return fmt.Errorf("failed to unpack consensus state: %w", err)
	}
	consensusState, ok := consensusStateI.(*tmclient.ConsensusState)
	if !ok {
		return fmt.Errorf("expected consensus state to be tendermint consensus state, got: %T", consensusStateI)
	}

	err = keyProof.VerifyMembership(
		commitmenttypes.GetSDKSpecs(),
		consensusState.GetRoot(),
		prefixedKey,
		value,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to verify key membership with key: %s, root %X: %w",
			key,
			consensusState.GetRoot().GetHash(),
			err,
		)
	}
	return nil
}

// verifyKeyNonMembership verifies a multihop non-membership proof including all intermediate state proofs.
func verifyKeyNonMembership(
	cdc codec.BinaryCodec,
	proofs *channeltypes.MsgMultihopProofs,
	prefix exported.Prefix,
	key string,
) error {
	// no keyproof provided, nothing to verify
	if proofs.KeyProof == nil {
		return nil
	}

	prefixedKey, err := commitmenttypes.ApplyPrefix(prefix, commitmenttypes.NewMerklePath(key))
	if err != nil {
		return err
	}

	var keyProof commitmenttypes.MerkleProof
	if err := cdc.Unmarshal(proofs.KeyProof.Proof, &keyProof); err != nil {
		return fmt.Errorf("failed to unmarshal key proof: %w", err)
	}

	var consensusStateI exported.ConsensusState
	index := uint32(len(proofs.ConsensusProofs)) - proofs.KeyProofIndex - 1
	if err := cdc.UnmarshalInterface(proofs.ConsensusProofs[index].Value, &consensusStateI); err != nil {
		return fmt.Errorf("failed to unpack consensus state: %w", err)
	}
	consensusState, ok := consensusStateI.(*tmclient.ConsensusState)
	if !ok {
		return fmt.Errorf("expected consensus state to be tendermint consensus state, got: %T", consensusStateI)
	}

	err = keyProof.VerifyNonMembership(
		commitmenttypes.GetSDKSpecs(),
		consensusState.GetRoot(),
		prefixedKey,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to verify key non-membership with key: %s, root %X: %w",
			key,
			consensusState.GetRoot().GetHash(),
			err,
		)
	}
	return nil
}

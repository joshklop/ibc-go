package types_test

import (
	"fmt"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	tmstate "github.com/cometbft/cometbft/state"
	sdkerrors "cosmossdk.io/errors"
	sdkerrortypes "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
)

const (
	gasUsed   = uint64(100)
	gasWanted = uint64(100)
)

// tests acknowledgement.ValidateBasic and acknowledgement.GetBytes
func (suite TypesTestSuite) TestAcknowledgement() { //nolint:govet // this is a test, we are okay with copying locks
	testCases := []struct {
		name       string
		ack        types.Acknowledgement
		expSuccess bool // indicate if this is a success or failed ack
		expPass    bool
	}{
		{
			"valid successful ack",
			types.NewResultAcknowledgement([]byte("success")),
			true,
			true,
		},
		{
			"valid failed ack",
			types.NewErrorAcknowledgement(fmt.Errorf("error")),
			false,
			true,
		},
		{
			"empty successful ack",
			types.NewResultAcknowledgement([]byte{}),
			true,
			false,
		},
		{
			"empty failed ack",
			types.NewErrorAcknowledgement(fmt.Errorf("  ")),
			false,
			true,
		},
		{
			"nil response",
			types.Acknowledgement{
				Response: nil,
			},
			false,
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			err := tc.ack.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}

			// expect all acks to be able to be marshaled
			suite.NotPanics(func() {
				bz := tc.ack.Acknowledgement()
				suite.Require().NotNil(bz)
			})

			suite.Require().Equal(tc.expSuccess, tc.ack.Success())
		})
	}
}

// The safety of including ABCI error codes in the acknowledgement rests
// on the inclusion of these ABCI error codes in the abcitypes.ResposneDeliverTx
// hash. If the ABCI codes get removed from consensus they must no longer be used
// in the packet acknowledgement.
//
// This test acts as an indicator that the ABCI error codes may no longer be deterministic.
func (suite *TypesTestSuite) TestABCICodeDeterminism() {
	// same ABCI error code used
	err := sdkerrors.Wrap(sdkerrortypes.ErrOutOfGas, "error string 1")
	errSameABCICode := sdkerrors.Wrap(sdkerrortypes.ErrOutOfGas, "error string 2")

	// different ABCI error code used
	errDifferentABCICode := sdkerrortypes.ErrNotFound

	deliverTx := sdkerrortypes.ResponseExecTxResultWithEvents(err, gasUsed, gasWanted, []abcitypes.Event{}, false)
	execTxResults := []*abcitypes.ExecTxResult{deliverTx}

	deliverTxSameABCICode := sdkerrortypes.ResponseExecTxResultWithEvents(errSameABCICode, gasUsed, gasWanted, []abcitypes.Event{}, false)
	resultsSameABCICode := []*abcitypes.ExecTxResult{deliverTxSameABCICode}

	deliverTxDifferentABCICode := sdkerrortypes.ResponseExecTxResultWithEvents(errDifferentABCICode, gasUsed, gasWanted, []abcitypes.Event{}, false)
	resultsDifferentABCICode := []*abcitypes.ExecTxResult{deliverTxDifferentABCICode}

	hash := tmstate.TxResultsHash(execTxResults)
	hashSameABCICode := tmstate.TxResultsHash(resultsSameABCICode)
	hashDifferentABCICode := tmstate.TxResultsHash(resultsDifferentABCICode)

	suite.Require().Equal(hash, hashSameABCICode)
	suite.Require().NotEqual(hash, hashDifferentABCICode)
}

// TestAcknowledgementError will verify that only a constant string and
// ABCI error code are used in constructing the acknowledgement error string
func (suite *TypesTestSuite) TestAcknowledgementError() {
	// same ABCI error code used
	err := sdkerrors.Wrap(sdkerrortypes.ErrOutOfGas, "error string 1")
	errSameABCICode := sdkerrors.Wrap(sdkerrortypes.ErrOutOfGas, "error string 2")

	// different ABCI error code used
	errDifferentABCICode := sdkerrortypes.ErrNotFound

	ack := types.NewErrorAcknowledgement(err)
	ackSameABCICode := types.NewErrorAcknowledgement(errSameABCICode)
	ackDifferentABCICode := types.NewErrorAcknowledgement(errDifferentABCICode)

	suite.Require().Equal(ack, ackSameABCICode)
	suite.Require().NotEqual(ack, ackDifferentABCICode)
}

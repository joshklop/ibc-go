package types_test

import (
	"fmt"
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	"cosmossdk.io/log"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"cosmossdk.io/store/metrics"
	"cosmossdk.io/store/iavl"
	"cosmossdk.io/store/rootmulti"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
	errorsmod "cosmossdk.io/errors"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v8/testing/simapp"
)

const (
	// valid constatns used for testing
	portid   = "testportid"
	chanid   = "channel-0"
	cpportid = "testcpport"
	cpchanid = "testcpchannel"

	version = "1.0"

	// invalid constants used for testing
	invalidPort      = "(invalidport1)"
	invalidShortPort = "p"
	// 195 characters
	invalidLongPort = "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Duis eros neque, ultricies vel ligula ac, convallis porttitor elit. Maecenas tincidunt turpis elit, vel faucibus nisl pellentesque sodales"

	invalidChannel      = "(invalidchannel1)"
	invalidShortChannel = "invalid"
	invalidLongChannel  = "invalidlongchannelinvalidlongchannelinvalidlongchannelinvalidlongchannel"

	invalidConnection      = "(invalidconnection1)"
	invalidShortConnection = "invalidcn"
	invalidLongConnection  = "invalidlongconnectioninvalidlongconnectioninvalidlongconnectioninvalid"
)

// define variables used for testing
var (
	height            = clienttypes.NewHeight(0, 1)
	timeoutHeight     = clienttypes.NewHeight(0, 100)
	timeoutTimestamp  = uint64(100)
	disabledTimeout   = clienttypes.ZeroHeight()
	validPacketData   = []byte("testdata")
	unknownPacketData = []byte("unknown")

	packet        = types.NewPacket(validPacketData, 1, portid, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp)
	invalidPacket = types.NewPacket(unknownPacketData, 0, portid, chanid, cpportid, cpchanid, timeoutHeight, timeoutTimestamp)

	emptyProof = []byte{}

	addr      = sdk.AccAddress("testaddr111111111111").String()
	emptyAddr string

	connHops             = []string{"testconnection"}
	invalidConnHops      = []string{"testconnection", "testconnection"}
	invalidShortConnHops = []string{invalidShortConnection}
	invalidLongConnHops  = []string{invalidLongConnection}
)

type TypesTestSuite struct {
	suite.Suite

	proof []byte
}

func (suite *TypesTestSuite) SetupTest() {
	app := simapp.Setup(false)
	db := dbm.NewMemDB()
	dblog := log.NewTestLogger(suite.T())
	store := rootmulti.NewStore(db, dblog, metrics.NewNoOpMetrics())
	storeKey := storetypes.NewKVStoreKey("iavlStoreKey")

	store.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, nil)
	err := store.LoadVersion(0)
	suite.Require().NoError(err)
	iavlStore := store.GetCommitStore(storeKey).(*iavl.Store)

	iavlStore.Set([]byte("KEY"), []byte("VALUE"))
	_ = store.Commit()

	res, err := store.Query(&storetypes.RequestQuery{
		Path:  fmt.Sprintf("/%s/key", storeKey.Name()), // required path to get key/value+proof
		Data:  []byte("KEY"),
		Prove: true,
	})
	suite.Require().NoError(err)

	merkleProof, err := commitmenttypes.ConvertProofs(res.ProofOps)
	suite.Require().NoError(err)
	proof, err := app.AppCodec().Marshal(&merkleProof)
	suite.Require().NoError(err)

	suite.proof = proof
}

func TestTypesTestSuite(t *testing.T) {
	suite.Run(t, new(TypesTestSuite))
}

func (suite *TypesTestSuite) TestMsgChannelOpenInitValidateBasic() {
	counterparty := types.NewCounterparty(cpportid, cpchanid)
	tryOpenChannel := types.NewChannel(types.TRYOPEN, types.ORDERED, counterparty, connHops, version)

	testCases := []struct {
		name    string
		msg     *types.MsgChannelOpenInit
		expPass bool
	}{
		{"", types.NewMsgChannelOpenInit(portid, version, types.ORDERED, connHops, cpportid, addr), true},
		{"too short port id", types.NewMsgChannelOpenInit(invalidShortPort, version, types.ORDERED, connHops, cpportid, addr), false},
		{"too long port id", types.NewMsgChannelOpenInit(invalidLongPort, version, types.ORDERED, connHops, cpportid, addr), false},
		{"port id contains non-alpha", types.NewMsgChannelOpenInit(invalidPort, version, types.ORDERED, connHops, cpportid, addr), false},
		{"invalid channel order", types.NewMsgChannelOpenInit(portid, version, types.Order(3), connHops, cpportid, addr), false},
		{"connection hops more than 1 ", types.NewMsgChannelOpenInit(portid, version, types.ORDERED, invalidConnHops, cpportid, addr), true},
		{"too short connection id", types.NewMsgChannelOpenInit(portid, version, types.UNORDERED, invalidShortConnHops, cpportid, addr), false},
		{"too long connection id", types.NewMsgChannelOpenInit(portid, version, types.UNORDERED, invalidLongConnHops, cpportid, addr), false},
		{"connection id contains non-alpha", types.NewMsgChannelOpenInit(portid, version, types.UNORDERED, []string{invalidConnection}, cpportid, addr), false},
		{"", types.NewMsgChannelOpenInit(portid, "", types.UNORDERED, connHops, cpportid, addr), true},
		{"invalid counterparty port id", types.NewMsgChannelOpenInit(portid, version, types.UNORDERED, connHops, invalidPort, addr), false},
		{"channel not in INIT state", &types.MsgChannelOpenInit{portid, tryOpenChannel, addr}, false},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()
			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgChannelOpenTryValidateBasic() {
	counterparty := types.NewCounterparty(cpportid, cpchanid)
	initChannel := types.NewChannel(types.INIT, types.ORDERED, counterparty, connHops, version)

	testCases := []struct {
		name    string
		msg     *types.MsgChannelOpenTry
		expPass bool
	}{
		{"", types.NewMsgChannelOpenTry(portid, version, types.ORDERED, connHops, cpportid, cpchanid, version, suite.proof, height, addr), true},
		{"too short port id", types.NewMsgChannelOpenTry(invalidShortPort, version, types.ORDERED, connHops, cpportid, cpchanid, version, suite.proof, height, addr), false},
		{"too long port id", types.NewMsgChannelOpenTry(invalidLongPort, version, types.ORDERED, connHops, cpportid, cpchanid, version, suite.proof, height, addr), false},
		{"port id contains non-alpha", types.NewMsgChannelOpenTry(invalidPort, version, types.ORDERED, connHops, cpportid, cpchanid, version, suite.proof, height, addr), false},
		{"", types.NewMsgChannelOpenTry(portid, version, types.ORDERED, connHops, cpportid, cpchanid, "", suite.proof, height, addr), true},
		{"invalid channel order", types.NewMsgChannelOpenTry(portid, version, types.Order(4), connHops, cpportid, cpchanid, version, suite.proof, height, addr), false},
		{"connection hops more than 1 ", types.NewMsgChannelOpenTry(portid, version, types.UNORDERED, invalidConnHops, cpportid, cpchanid, version, suite.proof, height, addr), true},
		{"too short connection id", types.NewMsgChannelOpenTry(portid, version, types.UNORDERED, invalidShortConnHops, cpportid, cpchanid, version, suite.proof, height, addr), false},
		{"too long connection id", types.NewMsgChannelOpenTry(portid, version, types.UNORDERED, invalidLongConnHops, cpportid, cpchanid, version, suite.proof, height, addr), false},
		{"connection id contains non-alpha", types.NewMsgChannelOpenTry(portid, version, types.UNORDERED, []string{invalidConnection}, cpportid, cpchanid, version, suite.proof, height, addr), false},
		{"", types.NewMsgChannelOpenTry(portid, "", types.UNORDERED, connHops, cpportid, cpchanid, version, suite.proof, height, addr), true},
		{"invalid counterparty port id", types.NewMsgChannelOpenTry(portid, version, types.UNORDERED, connHops, invalidPort, cpchanid, version, suite.proof, height, addr), false},
		{"invalid counterparty channel id", types.NewMsgChannelOpenTry(portid, version, types.UNORDERED, connHops, cpportid, invalidChannel, version, suite.proof, height, addr), false},
		{"empty proof", types.NewMsgChannelOpenTry(portid, version, types.UNORDERED, connHops, cpportid, cpchanid, version, emptyProof, height, addr), false},
		{"channel not in TRYOPEN state", &types.MsgChannelOpenTry{portid, "", initChannel, version, suite.proof, height, addr}, false},
		{"previous channel id is not empty", &types.MsgChannelOpenTry{portid, chanid, initChannel, version, suite.proof, height, addr}, false},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgChannelOpenAckValidateBasic() {
	testCases := []struct {
		name    string
		msg     *types.MsgChannelOpenAck
		expPass bool
	}{
		{"", types.NewMsgChannelOpenAck(portid, chanid, chanid, version, suite.proof, height, addr), true},
		{"too short port id", types.NewMsgChannelOpenAck(invalidShortPort, chanid, chanid, version, suite.proof, height, addr), false},
		{"too long port id", types.NewMsgChannelOpenAck(invalidLongPort, chanid, chanid, version, suite.proof, height, addr), false},
		{"port id contains non-alpha", types.NewMsgChannelOpenAck(invalidPort, chanid, chanid, version, suite.proof, height, addr), false},
		{"too short channel id", types.NewMsgChannelOpenAck(portid, invalidShortChannel, chanid, version, suite.proof, height, addr), false},
		{"too long channel id", types.NewMsgChannelOpenAck(portid, invalidLongChannel, chanid, version, suite.proof, height, addr), false},
		{"channel id contains non-alpha", types.NewMsgChannelOpenAck(portid, invalidChannel, chanid, version, suite.proof, height, addr), false},
		{"", types.NewMsgChannelOpenAck(portid, chanid, chanid, "", suite.proof, height, addr), true},
		{"empty proof", types.NewMsgChannelOpenAck(portid, chanid, chanid, version, emptyProof, height, addr), false},
		{"invalid counterparty channel id", types.NewMsgChannelOpenAck(portid, chanid, invalidShortChannel, version, suite.proof, height, addr), false},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgChannelOpenConfirmValidateBasic() {
	testCases := []struct {
		name    string
		msg     *types.MsgChannelOpenConfirm
		expPass bool
	}{
		{"", types.NewMsgChannelOpenConfirm(portid, chanid, suite.proof, height, addr), true},
		{"too short port id", types.NewMsgChannelOpenConfirm(invalidShortPort, chanid, suite.proof, height, addr), false},
		{"too long port id", types.NewMsgChannelOpenConfirm(invalidLongPort, chanid, suite.proof, height, addr), false},
		{"port id contains non-alpha", types.NewMsgChannelOpenConfirm(invalidPort, chanid, suite.proof, height, addr), false},
		{"too short channel id", types.NewMsgChannelOpenConfirm(portid, invalidShortChannel, suite.proof, height, addr), false},
		{"too long channel id", types.NewMsgChannelOpenConfirm(portid, invalidLongChannel, suite.proof, height, addr), false},
		{"channel id contains non-alpha", types.NewMsgChannelOpenConfirm(portid, invalidChannel, suite.proof, height, addr), false},
		{"empty proof", types.NewMsgChannelOpenConfirm(portid, chanid, emptyProof, height, addr), false},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgChannelCloseInitValidateBasic() {
	testCases := []struct {
		name    string
		msg     *types.MsgChannelCloseInit
		expPass bool
	}{
		{"", types.NewMsgChannelCloseInit(portid, chanid, addr), true},
		{"too short port id", types.NewMsgChannelCloseInit(invalidShortPort, chanid, addr), false},
		{"too long port id", types.NewMsgChannelCloseInit(invalidLongPort, chanid, addr), false},
		{"port id contains non-alpha", types.NewMsgChannelCloseInit(invalidPort, chanid, addr), false},
		{"too short channel id", types.NewMsgChannelCloseInit(portid, invalidShortChannel, addr), false},
		{"too long channel id", types.NewMsgChannelCloseInit(portid, invalidLongChannel, addr), false},
		{"channel id contains non-alpha", types.NewMsgChannelCloseInit(portid, invalidChannel, addr), false},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgChannelCloseConfirmValidateBasic() {
	testCases := []struct {
		name    string
		msg     *types.MsgChannelCloseConfirm
		expPass bool
	}{
		{"", types.NewMsgChannelCloseConfirm(portid, chanid, suite.proof, height, addr), true},
		{"too short port id", types.NewMsgChannelCloseConfirm(invalidShortPort, chanid, suite.proof, height, addr), false},
		{"too long port id", types.NewMsgChannelCloseConfirm(invalidLongPort, chanid, suite.proof, height, addr), false},
		{"port id contains non-alpha", types.NewMsgChannelCloseConfirm(invalidPort, chanid, suite.proof, height, addr), false},
		{"too short channel id", types.NewMsgChannelCloseConfirm(portid, invalidShortChannel, suite.proof, height, addr), false},
		{"too long channel id", types.NewMsgChannelCloseConfirm(portid, invalidLongChannel, suite.proof, height, addr), false},
		{"channel id contains non-alpha", types.NewMsgChannelCloseConfirm(portid, invalidChannel, suite.proof, height, addr), false},
		{"empty proof", types.NewMsgChannelCloseConfirm(portid, chanid, emptyProof, height, addr), false},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgRecvPacketValidateBasic() {
	testCases := []struct {
		name    string
		msg     *types.MsgRecvPacket
		expPass bool
	}{
		{"success", types.NewMsgRecvPacket(packet, suite.proof, height, addr), true},
		{"missing signer address", types.NewMsgRecvPacket(packet, suite.proof, height, emptyAddr), false},
		{"proof contain empty proof", types.NewMsgRecvPacket(packet, emptyProof, height, addr), false},
		{"invalid packet", types.NewMsgRecvPacket(invalidPacket, suite.proof, height, addr), false},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expPass {
				suite.NoError(err)
			} else {
				suite.Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgRecvPacketGetSigners() {
	msg := types.NewMsgRecvPacket(packet, suite.proof, height, addr)
	res := msg.GetSigners()

	expected := "[7465737461646472313131313131313131313131]"
	suite.Equal(expected, fmt.Sprintf("%v", res))
}

func (suite *TypesTestSuite) TestMsgTimeoutValidateBasic() {
	testCases := []struct {
		name    string
		msg     *types.MsgTimeout
		expPass bool
	}{
		{"success", types.NewMsgTimeout(packet, 1, suite.proof, height, addr), true},
		{"seq 0", types.NewMsgTimeout(packet, 0, suite.proof, height, addr), false},
		{"missing signer address", types.NewMsgTimeout(packet, 1, suite.proof, height, emptyAddr), false},
		{"cannot submit an empty proof", types.NewMsgTimeout(packet, 1, emptyProof, height, addr), false},
		{"invalid packet", types.NewMsgTimeout(invalidPacket, 1, suite.proof, height, addr), false},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgTimeoutOnCloseValidateBasic() {
	counterparty := types.NewCounterparty(cpportid, cpchanid)
	tryOpenChannel := types.NewChannel(types.TRYOPEN, types.ORDERED, counterparty, connHops, version)

	testCases := []struct {
		name   string
		msg    *types.MsgChannelOpenInit
		expErr error
	}{
		{
			"success",
			types.NewMsgChannelOpenInit(portid, version, types.ORDERED, connHops, cpportid, addr),
			nil,
		},
		{
			"success: empty version",
			types.NewMsgChannelOpenInit(portid, "", types.UNORDERED, connHops, cpportid, addr),
			nil,
		},
		{
			"too short port id",
			types.NewMsgChannelOpenInit(invalidShortPort, version, types.ORDERED, connHops, cpportid, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s has invalid length: %d, must be between %d-%d characters",
					invalidShortPort,
					len(invalidShortPort),
					2,
					host.DefaultMaxPortCharacterLength,
				), "invalid port ID"),
		},
		{
			"too long port id",
			types.NewMsgChannelOpenInit(invalidLongPort, version, types.ORDERED, connHops, cpportid, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s has invalid length: %d, must be between %d-%d characters",
					invalidLongPort,
					len(invalidLongPort),
					2,
					host.DefaultMaxPortCharacterLength,
				), "invalid port ID"),
		},
		{
			"port id contains non-alpha",
			types.NewMsgChannelOpenInit(invalidPort, version, types.ORDERED, connHops, cpportid, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s must contain only alphanumeric or the following characters: '.', '_', '+', '-', '#', '[', ']', '<', '>'",
					invalidPort,
				), "invalid port ID"),
		},
		{
			"invalid channel order",
			types.NewMsgChannelOpenInit(portid, version, types.Order(3),
				connHops, cpportid, addr),
			errorsmod.Wrap(types.ErrInvalidChannelOrdering, types.Order(3).String()),
		},
		{
			"connection hops more than 1 ",
			types.NewMsgChannelOpenInit(portid, version, types.ORDERED, invalidConnHops, cpportid, addr),
			errorsmod.Wrap(
				types.ErrTooManyConnectionHops,
				"current IBC version only supports one connection hop",
			),
		},
		{
			"too short connection id",
			types.NewMsgChannelOpenInit(portid, version, types.UNORDERED, invalidShortConnHops, cpportid, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s has invalid length: %d, must be between %d-%d characters",
					invalidShortConnection, len(invalidShortConnection), 10, host.DefaultMaxCharacterLength),
				"invalid connection hop ID",
			),
		},
		{
			"too long connection id",
			types.NewMsgChannelOpenInit(portid, version, types.UNORDERED, invalidLongConnHops, cpportid, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s has invalid length: %d, must be between %d-%d characters",
					invalidLongConnection, len(invalidLongConnection), 10, host.DefaultMaxCharacterLength),
				"invalid connection hop ID",
			),
		},
		{
			"connection id contains non-alpha",
			types.NewMsgChannelOpenInit(portid, version, types.UNORDERED, []string{invalidConnection}, cpportid, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s must contain only alphanumeric or the following characters: '.', '_', '+', '-', '#', '[', ']', '<', '>'",
					invalidConnection,
				), "invalid connection hop ID",
			),
		},
		{
			"invalid counterparty port id",
			types.NewMsgChannelOpenInit(portid, version, types.UNORDERED, connHops, invalidPort, addr),
			errorsmod.Wrap(
				errorsmod.Wrapf(
					host.ErrInvalidID,
					"identifier %s must contain only alphanumeric or the following characters: '.', '_', '+', '-', '#', '[', ']', '<', '>'",
					invalidPort,
				), "invalid counterparty port ID",
			),
		},
		{
			"channel not in INIT state",
			&types.MsgChannelOpenInit{portid, tryOpenChannel, addr},
			errorsmod.Wrapf(types.ErrInvalidChannelState,
				"channel state must be INIT in MsgChannelOpenInit. expected: %s, got: %s",
				types.INIT, tryOpenChannel.State,
			),
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().Equal(err.Error(), tc.expErr.Error())
			}
		})
	}
}

func (suite *TypesTestSuite) TestMsgAcknowledgementValidateBasic() {
	testCases := []struct {
		name    string
		msg     *types.MsgAcknowledgement
		expPass bool
	}{
		{"success", types.NewMsgAcknowledgement(packet, packet.GetData(), suite.proof, height, addr), true},
		{"empty ack", types.NewMsgAcknowledgement(packet, nil, suite.proof, height, addr), false},
		{"missing signer address", types.NewMsgAcknowledgement(packet, packet.GetData(), suite.proof, height, emptyAddr), false},
		{"cannot submit an empty proof", types.NewMsgAcknowledgement(packet, packet.GetData(), emptyProof, height, addr), false},
		{"invalid packet", types.NewMsgAcknowledgement(invalidPacket, packet.GetData(), suite.proof, height, addr), false},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

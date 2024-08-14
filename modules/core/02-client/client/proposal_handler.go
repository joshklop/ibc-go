package client

import (
	govclient "cosmossdk.io/x/gov/client"

	"github.com/cosmos/ibc-go/v8/modules/core/02-client/client/cli"
)

var (
	UpdateClientProposalHandler = govclient.NewProposalHandler(cli.NewCmdSubmitUpdateClientProposal)
	UpgradeProposalHandler      = govclient.NewProposalHandler(cli.NewCmdSubmitUpgradeProposal)
)

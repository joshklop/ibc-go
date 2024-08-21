package types

import (
	"time"
	"context"

	//sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
)

// StakingKeeper expected staking keeper
type StakingKeeper interface {
	GetHistoricalInfo(ctx context.Context, height int64) (stakingtypes.HistoricalInfo, error)
	UnbondingTime(ctx context.Context) (time.Duration, error)
}

// UpgradeKeeper expected upgrade keeper
type UpgradeKeeper interface {
	ClearIBCState(ctx context.Context, lastHeight int64) error
	GetUpgradePlan(ctx context.Context) (plan upgradetypes.Plan, err error)
	GetUpgradedClient(ctx context.Context, height int64) ([]byte, error)
	SetUpgradedClient(ctx context.Context, planHeight int64, bz []byte) error
	GetUpgradedConsensusState(ctx context.Context, lastHeight int64) ([]byte, error)
	SetUpgradedConsensusState(ctx context.Context, planHeight int64, bz []byte) error
	ScheduleUpgrade(ctx context.Context, plan upgradetypes.Plan) error
}

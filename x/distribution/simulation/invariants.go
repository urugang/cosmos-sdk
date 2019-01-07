package simulation

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/mock/simulation"
	"github.com/cosmos/cosmos-sdk/x/stake"
)

// AllInvariants runs all invariants of the distribution module
func AllInvariants(d distr.Keeper, stk stake.Keeper) simulation.Invariant {
	return func(ctx sdk.Context) error {
		err := NonNegativeOutstandingInvariant(d)(ctx)
		if err != nil {
			return err
		}
		err = CanWithdrawInvariant(d, stk)(ctx)
		if err != nil {
			return err
		}
		return nil
	}
}

// NonNegativeOutstandingInvariant checks that outstanding unwithdrawn fees are never negative
func NonNegativeOutstandingInvariant(k distr.Keeper) simulation.Invariant {
	return func(ctx sdk.Context) error {
		outstanding := k.GetOutstandingRewards(ctx)
		if outstanding.HasNegative() {
			return fmt.Errorf("Negative outstanding coins: %v", outstanding)
		}
		return nil
	}
}

// CanWithdrawInvariant checks that current rewards can be completely withdrawn
func CanWithdrawInvariant(k distr.Keeper, sk stake.Keeper) simulation.Invariant {
	return func(ctx sdk.Context) error {

		// cache, we don't want to write changes
		ctx, _ = ctx.CacheContext()

		outstanding := k.GetOutstandingRewards(ctx)
		fmt.Printf("Outstanding rewards: %v\n", outstanding)

		// iterate over all bonded validators, withdraw commission
		sk.IterateBondedValidatorsByPower(ctx, func(_ int64, val sdk.Validator) (stop bool) {
			_ = k.WithdrawValidatorCommission(ctx, val.GetOperator())
			return false
		})

		// iterate over all current delegations, withdraw rewards
		dels := sk.GetAllDelegations(ctx)
		for _, delegation := range dels {
			_ = k.WithdrawDelegationRewards(ctx, delegation.DelegatorAddr, delegation.ValidatorAddr)
		}

		remaining := k.GetOutstandingRewards(ctx)
		if len(remaining) > 0 && remaining[0].Amount.LT(sdk.ZeroDec()) {
			return fmt.Errorf("Negative remaining coins: %v", remaining)
		}

		fmt.Printf("remaining coins: %v\n", remaining)
		// TODO somehow cap error
		// permit up to 10% error due to rounding
		// if len(remaining) > 0 && remaining[0].Amount.Quo(outstanding[0].Amount).GT(sdk.OneDec().Quo(sdk.NewDec(100))) {
		//	return fmt.Errorf("High error - outstanding %v, remaining %v", outstanding, remaining)
		//}

		return nil
	}
}

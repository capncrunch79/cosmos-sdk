package gov

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov/keeper"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

// InitGenesis - store genesis parameters
func InitGenesis(ctx sdk.Context, ak types.AccountKeeper, bk types.BankKeeper, k *keeper.Keeper, data *v1.GenesisState) {
	err := k.SetProposalID(ctx, data.StartingProposalId)
	if err != nil {
		panic(err)
	}

	err = k.Params.Set(ctx, *data.Params)
	if err != nil {
		panic(err)
	}

	err = k.Constitution.Set(ctx, data.Constitution)
	if err != nil {
		panic(err)
	}

	// check if the deposits pool account exists
	moduleAcc := k.GetGovernanceAccount(ctx)
	if moduleAcc == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.ModuleName))
	}

	var totalDeposits sdk.Coins
	for _, deposit := range data.Deposits {
		k.SetDeposit(ctx, *deposit)
		totalDeposits = totalDeposits.Add(deposit.Amount...)
	}

	for _, vote := range data.Votes {
		k.SetVote(ctx, *vote)
	}

	for _, proposal := range data.Proposals {
		switch proposal.Status {
		case v1.StatusDepositPeriod:
			k.InsertInactiveProposalQueue(ctx, proposal.Id, *proposal.DepositEndTime)
		case v1.StatusVotingPeriod:
			k.InsertActiveProposalQueue(ctx, proposal.Id, *proposal.VotingEndTime)
		}
		k.SetProposal(ctx, *proposal)
	}

	// if account has zero balance it probably means it's not set, so we set it
	balance := bk.GetAllBalances(ctx, moduleAcc.GetAddress())
	if balance.IsZero() {
		ak.SetModuleAccount(ctx, moduleAcc)
	}

	// check if total deposits equals balance, if it doesn't panic because there were export/import errors
	if !balance.Equal(totalDeposits) {
		panic(fmt.Sprintf("expected module account was %s but we got %s", balance.String(), totalDeposits.String()))
	}
}

// ExportGenesis - output genesis parameters
func ExportGenesis(ctx sdk.Context, k *keeper.Keeper) (*v1.GenesisState, error) {
	startingProposalID, err := k.GetProposalID(ctx)
	if err != nil {
		return nil, err
	}

	proposals, err := k.GetProposals(ctx)
	if err != nil {
		return nil, err
	}

	constitution, err := k.Constitution.Get(ctx)
	if err != nil {
		return nil, err
	}

	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	var proposalsDeposits v1.Deposits
	var proposalsVotes v1.Votes
	for _, proposal := range proposals {
		deposits, err := k.GetDeposits(ctx, proposal.Id)
		if err != nil {
			return nil, err
		}

		proposalsDeposits = append(proposalsDeposits, deposits...)

		votes, err := k.GetVotes(ctx, proposal.Id)
		if err != nil {
			return nil, err
		}

		proposalsVotes = append(proposalsVotes, votes...)
	}

	return &v1.GenesisState{
		StartingProposalId: startingProposalID,
		Deposits:           proposalsDeposits,
		Votes:              proposalsVotes,
		Proposals:          proposals,
		Params:             &params,
		Constitution:       constitution,
	}, nil
}

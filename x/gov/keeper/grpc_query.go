package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store/prefix"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	v3 "github.com/cosmos/cosmos-sdk/x/gov/migrations/v3"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

var _ v1.QueryServer = queryServer{}

type queryServer struct{ k Keeper }

func NewQueryServer(k Keeper) v1.QueryServer {
	return queryServer{k: k}
}

func (q queryServer) Constitution(ctx context.Context, _ *v1.QueryConstitutionRequest) (*v1.QueryConstitutionResponse, error) {
	constitution, err := q.k.Constitution.Get(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.QueryConstitutionResponse{Constitution: constitution}, nil
}

// Proposal returns proposal details based on ProposalID
func (q queryServer) Proposal(ctx context.Context, req *v1.QueryProposalRequest) (*v1.QueryProposalResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ProposalId == 0 {
		return nil, status.Error(codes.InvalidArgument, "proposal id can not be 0")
	}

	proposal, err := q.k.GetProposal(ctx, req.ProposalId)
	if err != nil {
		if errors.IsOf(err, types.ErrProposalNotFound) {
			return nil, status.Errorf(codes.NotFound, "proposal %d doesn't exist", req.ProposalId)
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.QueryProposalResponse{Proposal: &proposal}, nil
}

// Proposals implements the Query/Proposals gRPC method
func (q queryServer) Proposals(ctx context.Context, req *v1.QueryProposalsRequest) (*v1.QueryProposalsResponse, error) {
	store := q.k.storeService.OpenKVStore(ctx)
	proposalStore := prefix.NewStore(runtime.KVStoreAdapter(store), types.ProposalsKeyPrefix)

	filteredProposals, pageRes, err := query.GenericFilteredPaginate(
		q.k.cdc,
		proposalStore,
		req.Pagination,
		func(key []byte, p *v1.Proposal) (*v1.Proposal, error) {
			matchVoter, matchDepositor, matchStatus := true, true, true

			// match status (if supplied/valid)
			if v1.ValidProposalStatus(req.ProposalStatus) {
				matchStatus = p.Status == req.ProposalStatus
			}

			// match voter address (if supplied)
			if len(req.Voter) > 0 {
				voter, err := q.k.authKeeper.StringToBytes(req.Voter)
				if err != nil {
					return nil, err
				}

				_, err = q.k.GetVote(ctx, p.Id, voter)
				// if no error, vote found, matchVoter = true
				matchVoter = err == nil
			}

			// match depositor (if supplied)
			if len(req.Depositor) > 0 {
				depositor, err := q.k.authKeeper.StringToBytes(req.Depositor)
				if err != nil {
					return nil, err
				}
				_, err = q.k.GetDeposit(ctx, p.Id, depositor)
				// if no error, deposit found, matchDepositor = true
				matchDepositor = err == nil
			}

			if matchVoter && matchDepositor && matchStatus {
				return p, nil
			}

			return nil, nil
		}, func() *v1.Proposal {
			return &v1.Proposal{}
		})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.QueryProposalsResponse{Proposals: filteredProposals, Pagination: pageRes}, nil
}

// Vote returns Voted information based on proposalID, voterAddr
func (q queryServer) Vote(ctx context.Context, req *v1.QueryVoteRequest) (*v1.QueryVoteResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ProposalId == 0 {
		return nil, status.Error(codes.InvalidArgument, "proposal id can not be 0")
	}

	if req.Voter == "" {
		return nil, status.Error(codes.InvalidArgument, "empty voter address")
	}

	voter, err := q.k.authKeeper.StringToBytes(req.Voter)
	if err != nil {
		return nil, err
	}
	vote, err := q.k.GetVote(ctx, req.ProposalId, voter)
	if err != nil {
		if errors.IsOf(err, types.ErrVoteNotFound) {
			return nil, status.Errorf(codes.InvalidArgument,
				"voter: %v not found for proposal: %v", req.Voter, req.ProposalId)
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.QueryVoteResponse{Vote: &vote}, nil
}

// Votes returns single proposal's votes
func (q queryServer) Votes(ctx context.Context, req *v1.QueryVotesRequest) (*v1.QueryVotesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ProposalId == 0 {
		return nil, status.Error(codes.InvalidArgument, "proposal id can not be 0")
	}

	var votes v1.Votes
	store := q.k.storeService.OpenKVStore(ctx)
	votesStore := prefix.NewStore(runtime.KVStoreAdapter(store), types.VotesKey(req.ProposalId))

	pageRes, err := query.Paginate(votesStore, req.Pagination, func(key, value []byte) error {
		var vote v1.Vote
		if err := q.k.cdc.Unmarshal(value, &vote); err != nil {
			return err
		}

		votes = append(votes, &vote)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.QueryVotesResponse{Votes: votes, Pagination: pageRes}, nil
}

// Params queries all params
func (q queryServer) Params(ctx context.Context, req *v1.QueryParamsRequest) (*v1.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	params, err := q.k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	response := &v1.QueryParamsResponse{}

	//nolint:staticcheck // needed for legacy parameters
	switch req.ParamsType {
	case v1.ParamDeposit:
		depositParams := v1.NewDepositParams(params.MinDeposit, params.MaxDepositPeriod)
		response.DepositParams = &depositParams

	case v1.ParamVoting:
		votingParams := v1.NewVotingParams(params.VotingPeriod)
		response.VotingParams = &votingParams

	case v1.ParamTallying:
		tallyParams := v1.NewTallyParams(params.Quorum, params.Threshold, params.VetoThreshold)
		response.TallyParams = &tallyParams

	default:
		return nil, status.Errorf(codes.InvalidArgument,
			"%s is not a valid parameter type", req.ParamsType)

	}
	response.Params = &params

	return response, nil
}

// Deposit queries single deposit information based on proposalID, depositAddr.
func (q queryServer) Deposit(ctx context.Context, req *v1.QueryDepositRequest) (*v1.QueryDepositResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ProposalId == 0 {
		return nil, status.Error(codes.InvalidArgument, "proposal id can not be 0")
	}

	if req.Depositor == "" {
		return nil, status.Error(codes.InvalidArgument, "empty depositor address")
	}

	depositor, err := q.k.authKeeper.StringToBytes(req.Depositor)
	if err != nil {
		return nil, err
	}
	deposit, err := q.k.GetDeposit(ctx, req.ProposalId, depositor)
	if err != nil {
		if errors.IsOf(err, types.ErrDepositNotFound) {
			return nil, status.Errorf(codes.InvalidArgument,
				"depositer: %v not found for proposal: %v", req.Depositor, req.ProposalId)
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.QueryDepositResponse{Deposit: &deposit}, nil
}

// Deposits returns single proposal's all deposits
func (q queryServer) Deposits(ctx context.Context, req *v1.QueryDepositsRequest) (*v1.QueryDepositsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ProposalId == 0 {
		return nil, status.Error(codes.InvalidArgument, "proposal id can not be 0")
	}

	var deposits []*v1.Deposit

	store := q.k.storeService.OpenKVStore(ctx)
	depositStore := prefix.NewStore(runtime.KVStoreAdapter(store), types.DepositsKey(req.ProposalId))

	pageRes, err := query.Paginate(depositStore, req.Pagination, func(key, value []byte) error {
		var deposit v1.Deposit
		if err := q.k.cdc.Unmarshal(value, &deposit); err != nil {
			return err
		}

		deposits = append(deposits, &deposit)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.QueryDepositsResponse{Deposits: deposits, Pagination: pageRes}, nil
}

// TallyResult queries the tally of a proposal vote
func (q queryServer) TallyResult(ctx context.Context, req *v1.QueryTallyResultRequest) (*v1.QueryTallyResultResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ProposalId == 0 {
		return nil, status.Error(codes.InvalidArgument, "proposal id can not be 0")
	}

	proposal, err := q.k.GetProposal(ctx, req.ProposalId)
	if err != nil {
		if errors.IsOf(err, types.ErrProposalNotFound) {
			return nil, status.Errorf(codes.NotFound, "proposal %d doesn't exist", req.ProposalId)
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	var tallyResult v1.TallyResult

	switch {
	case proposal.Status == v1.StatusDepositPeriod:
		tallyResult = v1.EmptyTallyResult()

	case proposal.Status == v1.StatusPassed || proposal.Status == v1.StatusRejected:
		tallyResult = *proposal.FinalTallyResult

	default:
		// proposal is in voting period
		var err error
		_, _, tallyResult, err = q.k.Tally(ctx, proposal)
		if err != nil {
			return nil, err
		}
	}

	return &v1.QueryTallyResultResponse{Tally: &tallyResult}, nil
}

var _ v1beta1.QueryServer = legacyQueryServer{}

type legacyQueryServer struct{ qs v1.QueryServer }

// NewLegacyQueryServer returns an implementation of the v1beta1 legacy QueryServer interface.
func NewLegacyQueryServer(k *Keeper) v1beta1.QueryServer {
	return &legacyQueryServer{qs: NewQueryServer(*k)}
}

func (q legacyQueryServer) Proposal(ctx context.Context, req *v1beta1.QueryProposalRequest) (*v1beta1.QueryProposalResponse, error) {
	resp, err := q.qs.Proposal(ctx, &v1.QueryProposalRequest{
		ProposalId: req.ProposalId,
	})
	if err != nil {
		return nil, err
	}

	proposal, err := v3.ConvertToLegacyProposal(*resp.Proposal)
	if err != nil {
		return nil, err
	}

	return &v1beta1.QueryProposalResponse{Proposal: proposal}, nil
}

func (q legacyQueryServer) Proposals(ctx context.Context, req *v1beta1.QueryProposalsRequest) (*v1beta1.QueryProposalsResponse, error) {
	resp, err := q.qs.Proposals(ctx, &v1.QueryProposalsRequest{
		ProposalStatus: v1.ProposalStatus(req.ProposalStatus),
		Voter:          req.Voter,
		Depositor:      req.Depositor,
		Pagination:     req.Pagination,
	})
	if err != nil {
		return nil, err
	}

	legacyProposals := make([]v1beta1.Proposal, len(resp.Proposals))
	for idx, proposal := range resp.Proposals {
		legacyProposals[idx], err = v3.ConvertToLegacyProposal(*proposal)
		if err != nil {
			return nil, err
		}
	}

	return &v1beta1.QueryProposalsResponse{
		Proposals:  legacyProposals,
		Pagination: resp.Pagination,
	}, nil
}

func (q legacyQueryServer) Vote(ctx context.Context, req *v1beta1.QueryVoteRequest) (*v1beta1.QueryVoteResponse, error) {
	resp, err := q.qs.Vote(ctx, &v1.QueryVoteRequest{
		ProposalId: req.ProposalId,
		Voter:      req.Voter,
	})
	if err != nil {
		return nil, err
	}

	vote, err := v3.ConvertToLegacyVote(*resp.Vote)
	if err != nil {
		return nil, err
	}

	return &v1beta1.QueryVoteResponse{Vote: vote}, nil
}

func (q legacyQueryServer) Votes(ctx context.Context, req *v1beta1.QueryVotesRequest) (*v1beta1.QueryVotesResponse, error) {
	resp, err := q.qs.Votes(ctx, &v1.QueryVotesRequest{
		ProposalId: req.ProposalId,
		Pagination: req.Pagination,
	})
	if err != nil {
		return nil, err
	}

	votes := make([]v1beta1.Vote, len(resp.Votes))
	for i, v := range resp.Votes {
		votes[i], err = v3.ConvertToLegacyVote(*v)
		if err != nil {
			return nil, err
		}
	}

	return &v1beta1.QueryVotesResponse{
		Votes:      votes,
		Pagination: resp.Pagination,
	}, nil
}

//nolint:staticcheck // this is needed for legacy param support
func (q legacyQueryServer) Params(ctx context.Context, req *v1beta1.QueryParamsRequest) (*v1beta1.QueryParamsResponse, error) {
	resp, err := q.qs.Params(ctx, &v1.QueryParamsRequest{
		ParamsType: req.ParamsType,
	})
	if err != nil {
		return nil, err
	}

	response := &v1beta1.QueryParamsResponse{}

	if resp.DepositParams != nil {
		minDeposit := sdk.NewCoins(resp.DepositParams.MinDeposit...)
		response.DepositParams = v1beta1.NewDepositParams(minDeposit, *resp.DepositParams.MaxDepositPeriod)
	}

	if resp.VotingParams != nil {
		response.VotingParams = v1beta1.NewVotingParams(*resp.VotingParams.VotingPeriod)
	}

	if resp.TallyParams != nil {
		quorum, err := sdkmath.LegacyNewDecFromStr(resp.TallyParams.Quorum)
		if err != nil {
			return nil, err
		}
		threshold, err := sdkmath.LegacyNewDecFromStr(resp.TallyParams.Threshold)
		if err != nil {
			return nil, err
		}
		vetoThreshold, err := sdkmath.LegacyNewDecFromStr(resp.TallyParams.VetoThreshold)
		if err != nil {
			return nil, err
		}

		response.TallyParams = v1beta1.NewTallyParams(quorum, threshold, vetoThreshold)
	}

	return response, nil
}

func (q legacyQueryServer) Deposit(ctx context.Context, req *v1beta1.QueryDepositRequest) (*v1beta1.QueryDepositResponse, error) {
	resp, err := q.qs.Deposit(ctx, &v1.QueryDepositRequest{
		ProposalId: req.ProposalId,
		Depositor:  req.Depositor,
	})
	if err != nil {
		return nil, err
	}

	deposit := v3.ConvertToLegacyDeposit(resp.Deposit)
	return &v1beta1.QueryDepositResponse{Deposit: deposit}, nil
}

func (q legacyQueryServer) Deposits(ctx context.Context, req *v1beta1.QueryDepositsRequest) (*v1beta1.QueryDepositsResponse, error) {
	resp, err := q.qs.Deposits(ctx, &v1.QueryDepositsRequest{
		ProposalId: req.ProposalId,
		Pagination: req.Pagination,
	})
	if err != nil {
		return nil, err
	}
	deposits := make([]v1beta1.Deposit, len(resp.Deposits))
	for idx, deposit := range resp.Deposits {
		deposits[idx] = v3.ConvertToLegacyDeposit(deposit)
	}

	return &v1beta1.QueryDepositsResponse{Deposits: deposits, Pagination: resp.Pagination}, nil
}

func (q legacyQueryServer) TallyResult(ctx context.Context, req *v1beta1.QueryTallyResultRequest) (*v1beta1.QueryTallyResultResponse, error) {
	resp, err := q.qs.TallyResult(ctx, &v1.QueryTallyResultRequest{
		ProposalId: req.ProposalId,
	})
	if err != nil {
		return nil, err
	}

	tally, err := v3.ConvertToLegacyTallyResult(resp.Tally)
	if err != nil {
		return nil, err
	}

	return &v1beta1.QueryTallyResultResponse{Tally: tally}, nil
}

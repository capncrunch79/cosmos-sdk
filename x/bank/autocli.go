package bank

import (
	"fmt"
	"strings"

	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	bankv1beta1 "cosmossdk.io/api/cosmos/bank/v1beta1"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/cosmos/cosmos-sdk/x/bank/types"
)

const (
	FlagDenom        = "denom"
	FlagResolveDenom = "resolve-denom"

	FlagSplit = "split"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: bankv1beta1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Balance",
					Use:       "balance [address]",
					Short:     "Query for account balances by address",
					Long: strings.TrimSpace(
						fmt.Sprintf(`Query the total balance of an account or of a specific denomination.

Example:
  $ %s query %s balances [address]
  $ %s query %s balances [address] --denom=[denom]
  $ %s query %s balances [address] --resolve-denom
`,
							version.AppName, types.ModuleName, version.AppName, types.ModuleName, version.AppName, types.ModuleName,
						),
					),
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "address"}},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						FlagDenom: {
							Usage: "The specific balance denomination to query for",
						},
						FlagResolveDenom: {
							Usage: "Resolve denom to human-readable denom from metadata",
						},
					},
				},
				{
					RpcMethod:      "SpentableBalance",
					Use:            "spendable-balance [address]",
					Short:          "Query for account spendable balances by address",
					Example:        fmt.Sprintf("$ %s query %s spendable-balances [address]", version.AppName, types.ModuleName),
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "address"}},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						FlagDenom: {
							Usage: "The specific balance denomination to query for",
						},
					},
				},
				{
					RpcMethod: "DenomMetadata",
					Use:       "denom-metadata [denom]",
					Short:     "Query for metadata of a specific denomination",
					Long: strings.TrimSpace(
						fmt.Sprintf(`Query the client metadata for all the registered coin denominations

Example:
  To query for the client metadata of all coin denominations use:
  $ %s query %s denom-metadata

To query for the client metadata of a specific coin denomination use:
  $ %s query %s denom-metadata --denom=[denom]
`,
							version.AppName, types.ModuleName, version.AppName, types.ModuleName,
						),
					),
					FlagOptions: map[string]*autocliv1.FlagOptions{
						FlagDenom: {
							Usage: "The specific balance denomination to query for",
						},
					},
				},
				{
					RpcMethod: "TotalSupply",
					Use:       "total",
					Short:     "Query the total supply of coins of the chain",
					Long: strings.TrimSpace(
						fmt.Sprintf(`Query total supply of coins that are held by accounts in the chain.

Example:
  $ %s query %s total

To query for the total supply of a specific coin denomination use:
  $ %s query %s total --denom=[denom]
`,
							version.AppName, types.ModuleName, version.AppName, types.ModuleName,
						),
					),
					FlagOptions: map[string]*autocliv1.FlagOptions{
						FlagDenom: {
							Usage: "The specific balance denomination to query for",
						},
					},
				},
				{
					RpcMethod: "SendEnabled",
					Use:       "send-enabled [denom1 ...]",
					Short:     "Query for send enabled entries",
					Long: strings.TrimSpace(`Query for send enabled entries that have been specifically set.

To look up one or more specific denoms, supply them as arguments to this command.
To look up all denoms, do not provide any arguments.
`,
					),
					Example: strings.TrimSpace(
						fmt.Sprintf(`Getting one specific entry:
  $ %[1]s query %[2]s send-enabled foocoin

Getting two specific entries:
  $ %[1]s query %[2]s send-enabled foocoin barcoin

Getting all entries:
  $ %[1]s query %[2]s send-enabled
`,
							version.AppName, types.ModuleName,
						),
					),
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "denoms", Varargs: true}},
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service: bankv1beta1.Msg_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Send",
					Use:       "send [from_key_or_address] [to_address] [amount]",
					Short:     "Send funds from one account to another",
					Long: `Send funds from one account to another.
Note, the '--from' flag is ignored as it is implied from [from_key_or_address].
When using '--dry-run' a key name cannot be used, only a bech32 address.`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "from"}, {ProtoField: "to"}, {ProtoField: "amount"}},
				},
				{
					RpcMethod: "MultiSend",
					Use:       "multi-send [from_key_or_address] [to_address_1, to_address_2, ...] [amount]",
					Short:     "Send funds from one account to two or more accounts.",
					Long: `Send funds from one account to two or more accounts.
By default, sends the [amount] to each address of the list.
Using the '--split' flag
, the [amount] is split equally between the addresses.
Note, the '--from' flag is ignored as it is implied from [from_key_or_address].
When using '--dry-run' a key name cannot be used, only a bech32 address.
`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "from"}, {ProtoField: "tos", Varargs: true}, {ProtoField: "amount"}},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						FlagSplit: {
							Usage: "Send the equally split token amount to each address",
						},
					},
				},
			},
		},
	}
}

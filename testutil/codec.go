package testutil

import (
	"github.com/cosmos/gogoproto/proto"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

// CodecOptions are options for creating a test codec.
type CodecOptions struct {
	AccAddressPrefix string
	ValAddressPrefix string
}

// NewInterfaceRegistry returns a new InterfaceRegistry with the given options.
func (o CodecOptions) NewInterfaceRegistry() codectypes.InterfaceRegistry {
	accAddressPrefix := o.AccAddressPrefix
	if accAddressPrefix == "" {
		accAddressPrefix = "sim"
	}

	valAddressPrefix := o.ValAddressPrefix
	if valAddressPrefix == "" {
		valAddressPrefix = "simvaloper"
	}

	ir, err := codectypes.NewInterfaceRegistryWithOptions(codectypes.InterfaceRegistryOptions{
		ProtoFiles:            proto.HybridResolver,
		AddressCodec:          address.NewBech32Codec(accAddressPrefix),
		ValidatorAddressCodec: address.NewBech32Codec(valAddressPrefix),
	})
	if err != nil {
		panic(err)
	}
	return ir
}

// NewCodec returns a new codec with the given options.
func (o CodecOptions) NewCodec() *codec.ProtoCodec {
	return codec.NewProtoCodec(o.NewInterfaceRegistry())
}

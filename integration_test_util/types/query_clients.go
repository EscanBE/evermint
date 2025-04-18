package types

//goland:noinspection SpellCheckingInspection
import (
	rpctypes "github.com/EscanBE/evermint/rpc/types"
	cpctypes "github.com/EscanBE/evermint/x/cpc/types"
	evmtypes "github.com/EscanBE/evermint/x/evm/types"
	httpclient "github.com/cometbft/cometbft/rpc/client/http"
	"github.com/cosmos/cosmos-sdk/client"
	sdktxtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	disttypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govtypeslegacy "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	grpc1 "github.com/cosmos/gogoproto/grpc"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
)

type QueryClients struct {
	GrpcConnection        grpc1.ClientConn
	ClientQueryCtx        client.Context
	CometBFTRpcHttpClient *httpclient.HTTP
	Auth                  authtypes.QueryClient
	Bank                  banktypes.QueryClient
	Distribution          disttypes.QueryClient
	EVM                   evmtypes.QueryClient
	CPC                   cpctypes.QueryClient
	GovV1                 govtypesv1.QueryClient
	GovLegacy             govtypeslegacy.QueryClient
	IbcTransfer           ibctransfertypes.QueryClient
	Slashing              slashingtypes.QueryClient
	Staking               stakingtypes.QueryClient
	ServiceClient         sdktxtypes.ServiceClient
	Rpc                   *rpctypes.QueryClient
}

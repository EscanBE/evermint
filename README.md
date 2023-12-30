This is branch [rename-chain/after](https://github.com/EscanBE/evermint/tree/rename-chain/after), the new chain Nevermind after renamed from Escan/Evermint as example in [PR #1](https://github.com/EscanBE/evermint/pull/1)

Summary new symbols of the chain after renamed:
```golang
const (
	ApplicationName = "nevermind" // renamed from "evermint"
	ApplicationBinaryName = "nvmd" // renamed from "evmd"
	ApplicationHome = ".nevermind" // renamed from ".evermint"

	GitHubRepo = "https://github.com/VictorTrustyDev/nevermind" // renamed from "https://github.com/EscanBE/evermint"

	BaseDenom = "uever" // renamed from "wei"
	DisplayDenom = "ever" // renamed from "ether"
	SymbolDenom = "EVER" // renamed from "ETH"
	BaseDenomExponent = 18

	Bech32Prefix = "ever" // renamed from "evm"

	MainnetFullChainId = "nevermind_123567-1" // renamed from "evermint_90909-1"
	TestnetFullChainId = "nevermind_5678-1" // renamed from "evermint_80808-1"
	DevnetFullChainId  = "nevermind_1234-1" // renamed from "evermint_70707-1"

	MainnetEIP155ChainId = 123567 // renamed from 90909
	TestnetEIP155ChainId = 5678 // renamed from 80808
	DevnetEIP155ChainId  = 1234 // renamed from 70707
)
```
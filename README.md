<!--
parent:
  order: false
-->

<div align="center">
  <h1>Evermint</h1>
</div>

<div align="center">
  <a href="https://github.com/EscanBE/evermint/blob/main/LICENSE">
    <img alt="License: LGPL-3.0" src="https://img.shields.io/github/license/EscanBE/evermint.svg" />
  </a>
  <a href="https://pkg.go.dev/github.com/evmos/evmos">
    <img alt="GoDoc" src="https://godoc.org/github.com/evmos/evmos?status.svg" />
  </a>
</div>

### Create your own fully customized EVM-enabled blockchain network in just 2 steps

[> Quick rename](https://github.com/EscanBE/evermint/blob/main/RENAME_CHAIN.md)

[> View example after rename](https://github.com/EscanBE/evermint/pull/1)

### About Evermint

Evermint is a fork of open source Evmos v12.1.6, maintained by Escan team with bug fixes, customization and enable developers to fork and transform to their chain, fully customized, in just 2 steps.

_Important Note: Evermint was born for development and research purpose so maintainers do not support migration for new upgrade/breaking changes._

### About Evmos

Evmos is a scalable, high-throughput Proof-of-Stake blockchain
that is fully compatible and interoperable with Ethereum.
It's built using the [Cosmos SDK](https://github.com/cosmos/cosmos-sdk/)
which runs on top of the [CometBFT](https://github.com/cometbft/cometbft) consensus engine.

### Different of Evermint & Evmos

- Evermint is for research and development purpose.
- Evermint is fork of open source Evmos v12.1.6 plus bug fixes.
- Evermint is [currently removing some modules](https://github.com/EscanBE/evermint/issues/41) from Evmos codebase and only keep `x/evm`, `x/erc20`, `x/feemarket`. The goal is to make it more simple for research and only focus on the skeleton of Evmos.
- After [upgraded to Cosmos-SDK v0.50](https://github.com/EscanBE/evermint/pull/148), Evermint is now has many breaking change to the Legacy Evmos v12.1.6. If you want to use the open-source libraries previously developed for Evmos v12.1.6, you might need to use [Evermint version which uses Cosmos-SDK v0.47 and below](https://github.com/EscanBE/evermint/tree/v12.3.0-cosmos47).

## Documentation

Evermint does not maintain its own documentation site, user can refer to Evmos documentation hosted at [evmos/docs](https://github.com/evmos/docs) and can be found at [docs.evmos.org](https://docs.evmos.org).
Head over there and check it out.

**Note**: Requires [Go 1.22+](https://golang.org/dl/)

## Quick Start

To learn how the Evmos works from a high-level perspective,
go to the [Protocol Overview](https://docs.evmos.org/protocol) section from the documentation.
You can also check the instructions to [Run a Node](https://docs.evmos.org/protocol/evmos-cli#run-an-evmos-node).

### Additional feature provided by Evermint:
1. Command convert between 0x address and bech32 address, or any custom bech32 HRP
```bash
evmd convert-address evm1sv9m0g7ycejwr3s369km58h5qe7xj77hxrsmsz evmos
# alias: "ca"
```
2. [Rename chain](https://github.com/EscanBE/evermint/blob/main/RENAME_CHAIN.md)
3. [Support vesting account creation](https://github.com/EscanBE/evermint/pull/144) with help from module `x/vauth`
4. [`snapshots` command](https://github.com/EscanBE/evermint/pull/12)
5. [`inspect` command](https://github.com/EscanBE/evermint/pull/14)
6. [Flag `--allow-insecure-unlock`](https://github.com/EscanBE/evermint/pull/142)
7. Dependencies updated: `Cosmos-SDK v0.50.9`, `CometBFT v0.38.12`, `ibc-go v8.5.0`, `go-ethereum v1.10.26`
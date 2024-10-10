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

Requires [Go 1.22+](https://golang.org/dl/)

### Create your own fully customized EVM-enabled blockchain network in just 2 steps

[> Quick rename](https://github.com/EscanBE/evermint/blob/main/RENAME_CHAIN.md)

[> View example after rename](https://github.com/EscanBE/evermint/pull/1)

### About Evermint

Evermint originally a fork of open source Evmos v12.1.6 plus lots of magic.

Many important pieces of code was replaced by Evermint, such as:
- Replaced legacy AnteHandler with [Dual-Lane AnteHandler](https://github.com/EscanBE/evermint/pull/164).
- Replaced legacy StateDB with [Context-based StateDB](https://github.com/EscanBE/evermint/pull/167).
- Used go-ethereum code for [state-transition](https://github.com/EscanBE/evermint/pull/156).
- Some project structure was replaced during upgrade to Cosmos-SDK v0.50, CometBFT v0.38, ibc-go v8.5.
- Data structure, code logic of `MsgEthereumTx` and some proto msgs were replaced.

#### Some other features provided by Evermint:
1. Support [stateful precompiled contracts](https://github.com/EscanBE/evermint/pull/175)
2. [Support vesting account creation](https://github.com/EscanBE/evermint/pull/144) with help from module `x/vauth`
3. [Rename chain](https://github.com/EscanBE/evermint/blob/main/RENAME_CHAIN.md)
4. [`snapshots` command](https://github.com/EscanBE/evermint/pull/12)
5. [`inspect` command](https://github.com/EscanBE/evermint/pull/14)
6. [Flag `--allow-insecure-unlock`](https://github.com/EscanBE/evermint/pull/142)
7. Command convert between 0x address and bech32 address, or any custom bech32 HRP
```bash
evmd convert-address evm1sv9m0g7ycejwr3s369km58h5qe7xj77hxrsmsz evmos
# alias: "ca"
```

### About Evmos
_v12.1.6_

Evmos is a scalable, high-throughput Proof-of-Stake blockchain
that is fully compatible and interoperable with Ethereum.
It's built using the [Cosmos SDK](https://github.com/cosmos/cosmos-sdk/)
which runs on top of the [Tendermint](https://github.com/tendermint/tendermint) consensus engine.

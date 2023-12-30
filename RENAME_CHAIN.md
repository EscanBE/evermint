### How to make your own EVM-enabled chain from this fork:
_(support Linux & MacOS only)_

Step 1: Open `constants.go` and change every constant in there.
Do `git commit` (+ `git push`) to save your changes.

Step 2: Run `./rename-chain.sh`

Done, you have your own chain now. Try running tests to make sure everything is working fine.

[View example after rename](https://github.com/VictorTrustyDev/nevermind/pull/1)
___
Cleanup notes to be checked after running script:
- Directory `./rename_chain` is deleted.
- Script `./rename-chain.sh` is deleted.
- Make sure the following words, which belong to definition of nevermind, are no longer exists:
  - nvmd (binary name)
  - nevermind (git repo + application name)
  - evm1 (bech32 prefix)
  - EscanBE (git owner name)

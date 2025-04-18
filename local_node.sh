#!/bin/bash

KEYS[0]="validator"
KEYS[1]="dev1"
KEYS[2]="dev2"
KEYS[3]="dev3"
# 0xc032bfb0a7f4d79f8bd0d4d6c6169f58e702817a
MNEMONICS[0]="camera foster skate whisper faith opera axis false van urban clean pet shove census surface injury phone alley cup school pet edge trial pony"
# 0x89760f514DCfCCCf1E4c5eDC6Bf6041931c4c183
MNEMONICS[1]="curtain hat remain song receive tower stereo hope frog cheap brown plate raccoon post reflect wool sail salmon game salon group glimpse adult shift"
# 0x21b661c8A270ed83D2826aD49b1E3B78F515E25C
MNEMONICS[2]="coral drink glow assist canyon ankle hole buffalo vendor foster void clip welcome slush cherry omit member legal account lunar often hen winter culture"
# 0x6479D25261A74B1b058778d3F69Ad7cC557341A8
MNEMONICS[3]="depth skull anxiety weasel pulp interest seek junk trumpet orbit glance drink comfort much alarm during lady strong matrix enable write pledge alcohol buzz"

CHAINID="evermint_70707-1" # devnet
MONIKER="localtestnet"
BINARY="evmd"
MIN_DENOM="wei"
# Remember to change to other types of keyring like 'file' in-case exposing to outside world,
# otherwise your balance will be wiped quickly
# The keyring test does not require private key to steal tokens from you
KEYRING="test"
KEYALGO="eth_secp256k1"
LOGLEVEL="info"
# Set dedicated home directory for the temp instance
HOMEDIR="$HOME/.tmp-evermint"
# to trace evm
#TRACE="--trace"
TRACE=""

# Path variables
CONFIG=$HOMEDIR/config/config.toml
APP_TOML=$HOMEDIR/config/app.toml
GENESIS=$HOMEDIR/config/genesis.json
TMP_GENESIS=$HOMEDIR/config/tmp_genesis.json

# validate dependencies are installed
command -v jq >/dev/null 2>&1 || {
	echo >&2 "jq not installed. More info: https://stedolan.github.io/jq/download/"
	exit 1
}

# used to exit on first error (any non-zero exit code)
set -e

# Reinstall daemon
make install

# User prompt if an existing local node configuration is found.
if [ -d "$HOMEDIR" ]; then
	printf "\nAn existing folder at '%s' was found. You can choose to delete this folder and start a new local node with new keys from genesis. When declined, the existing local node is started. \n" "$HOMEDIR"
	echo "Overwrite the existing configuration and start a new local node? [y/n]"
	read -r overwrite
else
	overwrite="Y"
fi


# Setup local node if overwrite is set to Yes, otherwise skip setup
if [[ $overwrite == "y" || $overwrite == "Y" ]]; then
	# Remove the previous folder
	rm -rf "$HOMEDIR"

	# Set client config
	"$BINARY" config set client chain-id $CHAINID --home "$HOMEDIR"
	"$BINARY" config set client keyring-backend $KEYRING --home "$HOMEDIR"

	# Recover keys from mnemonics
  echo "${MNEMONICS[0]}" | "$BINARY" keys add "${KEYS[0]}" --recover --keyring-backend $KEYRING --algo $KEYALGO --home "$HOMEDIR"
  echo "${MNEMONICS[1]}" | "$BINARY" keys add "${KEYS[1]}" --recover --keyring-backend $KEYRING --algo $KEYALGO --home "$HOMEDIR"
  echo "${MNEMONICS[2]}" | "$BINARY" keys add "${KEYS[2]}" --recover --keyring-backend $KEYRING --algo $KEYALGO --home "$HOMEDIR"
  echo "${MNEMONICS[3]}" | "$BINARY" keys add "${KEYS[3]}" --recover --keyring-backend $KEYRING --algo $KEYALGO --home "$HOMEDIR"

	# Set moniker for the node
	"$BINARY" init $MONIKER -o --chain-id $CHAINID --home "$HOMEDIR"

	# Improve genesis
  "$BINARY" genesis improve --home "$HOMEDIR"

	# Set deployer authority
	DEPLOYER_ADDRESS=$("$BINARY" keys show "${KEYS[0]}" -a --keyring-backend $KEYRING --home "$HOMEDIR")
	jq '.app_state["cpc"]["params"]["whitelisted_deployers"][0]="'$DEPLOYER_ADDRESS'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	# Automatically deploy Custom-Precompiled contracts
	jq '.app_state["cpc"]["deploy_erc20_native"]=true' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["cpc"]["deploy_staking_contract"]=true' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

  # enable prometheus metrics
  if [[ "$OSTYPE" == "darwin"* ]]; then
      sed -i '' 's/prometheus = false/prometheus = true/' "$CONFIG"
      sed -i '' 's/prometheus-retention-time = 0/prometheus-retention-time  = 1000000000000/g' "$APP_TOML"
      sed -i '' 's/enabled = false/enabled = true/g' "$APP_TOML"
  else
      sed -i 's/prometheus = false/prometheus = true/' "$CONFIG"
      sed -i 's/prometheus-retention-time  = "0"/prometheus-retention-time  = "1000000000000"/g' "$APP_TOML"
      sed -i 's/enabled = false/enabled = true/g' "$APP_TOML"
  fi
	
	# Change proposal periods to pass within a reasonable time for local testing
	sed -i.bak 's/"max_deposit_period": "172800s"/"max_deposit_period": "60s"/g' "$HOMEDIR"/config/genesis.json
	sed -i.bak 's/"voting_period": "172800s"/"voting_period": "120s"/g' "$HOMEDIR"/config/genesis.json
	sed -i.bak 's/"expedited_voting_period": "86400s"/"expedited_voting_period": "30s"/g' "$HOMEDIR"/config/genesis.json

	# set custom pruning settings
	sed -i.bak 's/pruning = "default"/pruning = "custom"/g' "$APP_TOML"
	sed -i.bak 's/pruning-keep-recent = "0"/pruning-keep-recent = "2"/g' "$APP_TOML"
	sed -i.bak 's/pruning-interval = "0"/pruning-interval = "10"/g' "$APP_TOML"

	# Allocate genesis accounts (cosmos formatted addresses)
	GENESIS_BALANCE="100000000000000000000000000"
  "$BINARY" add-genesis-account "${KEYS[0]}" "$GENESIS_BALANCE$MIN_DENOM" --keyring-backend $KEYRING --home "$HOMEDIR"
  "$BINARY" add-genesis-account "${KEYS[1]}" "$GENESIS_BALANCE$MIN_DENOM" --keyring-backend $KEYRING --home "$HOMEDIR"
  "$BINARY" add-genesis-account "${KEYS[2]}" "$GENESIS_BALANCE$MIN_DENOM" --keyring-backend $KEYRING --home "$HOMEDIR"
  "$BINARY" add-genesis-account "${KEYS[3]}" "$GENESIS_BALANCE$MIN_DENOM" --keyring-backend $KEYRING --home "$HOMEDIR"

	# bc is required to add these big numbers
	total_supply=$(echo "${#KEYS[@]} * $GENESIS_BALANCE" | bc)
	jq -r --arg total_supply "$total_supply" '.app_state["bank"]["supply"][0]["amount"]=$total_supply' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# Sign genesis transaction
	BASE_FEE="$(cat "$GENESIS" | jq -r .app_state.feemarket.params.base_fee)"
	"$BINARY" gentx "${KEYS[0]}" "1000000000000000000000$MIN_DENOM" --gas-prices "$BASE_FEE$MIN_DENOM" --keyring-backend $KEYRING --chain-id $CHAINID --home "$HOMEDIR"
	## In case you want to create multiple validators at genesis
	## 1. Back to `"$BINARY" keys add` step, init more keys
	## 2. Back to `"$BINARY" add-genesis-account` step, add balance for those
	## 3. Clone this ~/.evermint home directory into some others, let's say `~/.clonedHome`
	## 4. Run `gentx` in each of those folders
	## 5. Copy the `gentx-*` folders under `~/.clonedHome/config/gentx/` folders into the original `~/.evermint/config/gentx`

	# Collect genesis tx
	"$BINARY" collect-gentxs --home "$HOMEDIR"

	# Run this to ensure everything worked and that the genesis file is setup correctly
	"$BINARY" validate-genesis --home "$HOMEDIR"
fi

# Start the node (remove the --pruning=nothing flag if historical queries are not needed)
"$BINARY" start \
  --metrics "$TRACE" --log_level "$LOGLEVEL" \
  --minimum-gas-prices="0.0001$MIN_DENOM" \
  --json-rpc.api eth,txpool,personal,net,debug,web3 \
  --allow-insecure-unlock true \
  --api.enable \
  --grpc.enable true \
  --home "$HOMEDIR" \
  --chain-id "$CHAINID"

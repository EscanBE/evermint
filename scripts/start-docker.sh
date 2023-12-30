#!/bin/bash

KEY="dev0"
CHAINID="nevermind_80808-1"
MONIKER="mymoniker"
BINARY="./nvmd"
MIN_DENOM="uever"
DATA_DIR=$(mktemp -d -t nevermind-datadir.XXXXX)

echo "create and add new keys"
"$BINARY" keys add $KEY --home $DATA_DIR --no-backup --chain-id $CHAINID --algo "eth_secp256k1" --keyring-backend test
echo "init node with moniker=$MONIKER and chain-id=$CHAINID"
"$BINARY" init $MONIKER --chain-id $CHAINID --home $DATA_DIR
echo "prepare genesis: Allocate genesis accounts"
"$BINARY" add-genesis-account \
  "$("$BINARY" keys show $KEY -a --home $DATA_DIR --keyring-backend test)" "1000000000000000000$MIN_DENOM,1000000000000000000stake" \
  --home $DATA_DIR --keyring-backend test
echo "prepare genesis: Sign genesis transaction"
"$BINARY" gentx $KEY 1000000000000000000stake --keyring-backend test --home $DATA_DIR --keyring-backend test --chain-id $CHAINID
echo "prepare genesis: Collect genesis tx"
"$BINARY" collect-gentxs --home $DATA_DIR
echo "prepare genesis: Run validate-genesis to ensure everything worked and that the genesis file is setup correctly"
"$BINARY" validate-genesis --home $DATA_DIR

echo "starting node $i in background ..."
"$BINARY" start --pruning=nothing --rpc.unsafe \
--keyring-backend test --home $DATA_DIR \
>$DATA_DIR/node.log 2>&1 & disown

echo "started node"
tail -f /dev/null
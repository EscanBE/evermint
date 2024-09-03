package backend

import (
	"cosmossdk.io/errors"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	tmrpctypes "github.com/cometbft/cometbft/rpc/core/types"

	"github.com/EscanBE/evermint/v12/rpc/types"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	"github.com/cometbft/cometbft/proto/tendermint/crypto"
)

type txGasAndReward struct {
	gasUsed uint64
	reward  *big.Int
}

type sortGasAndReward []txGasAndReward

func (s sortGasAndReward) Len() int { return len(s) }
func (s sortGasAndReward) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s sortGasAndReward) Less(i, j int) bool {
	return s[i].reward.Cmp(s[j].reward) < 0
}

// getAccountNonce returns the account nonce for the given account address.
// If the pending value is true, it will iterate over the mempool (pending)
// txs in order to compute and return the pending tx sequence.
// Todo: include the ability to specify a blockNumber
func (b *Backend) getAccountNonce(accAddr common.Address, pending bool, height int64, logger log.Logger) (uint64, error) {
	queryClient := authtypes.NewQueryClient(b.clientCtx)
	adr := sdk.AccAddress(accAddr.Bytes()).String()
	ctx := types.ContextWithHeight(height)
	res, err := queryClient.Account(ctx, &authtypes.QueryAccountRequest{Address: adr})
	if err != nil {
		st, ok := status.FromError(err)
		// treat as account doesn't exist yet
		if ok && st.Code() == codes.NotFound {
			return 0, nil
		}
		return 0, err
	}
	var acc authtypes.AccountI
	if err := b.clientCtx.InterfaceRegistry.UnpackAny(res.Account, &acc); err != nil {
		return 0, err
	}

	nonce := acc.GetSequence()

	if !pending {
		return nonce, nil
	}

	// the account retriever doesn't include the uncommitted transactions on the nonce so we need to
	// to manually add them.
	pendingTxs, err := b.PendingTransactions()
	if err != nil {
		logger.Error("failed to fetch pending transactions", "error", err.Error())
		return nonce, nil
	}

	// add the uncommitted txs to the nonce counter
	// only supports `MsgEthereumTx` style tx
	for _, tx := range pendingTxs {
		for _, msg := range (*tx).GetMsgs() {
			ethMsg, ok := msg.(*evmtypes.MsgEthereumTx)
			if !ok {
				// not ethereum tx
				break
			}

			sender, err := ethMsg.GetSender(b.chainID)
			if err != nil {
				continue
			}
			if sender == accAddr {
				nonce++
			}
		}
	}

	return nonce, nil
}

// output: targetOneFeeHistory
func (b *Backend) processBlock(
	tendermintBlock *tmrpctypes.ResultBlock,
	ethBlock *map[string]interface{},
	rewardPercentiles []float64,
	tendermintBlockResult *tmrpctypes.ResultBlockResults,
	targetOneFeeHistory *types.OneFeeHistory,
) error {
	blockHeight := tendermintBlock.Block.Height
	blockBaseFee, err := b.BaseFee(tendermintBlockResult)
	if err != nil {
		return err
	}

	// set basefee
	targetOneFeeHistory.BaseFee = blockBaseFee
	cfg := b.ChainConfig()
	if cfg.IsLondon(big.NewInt(blockHeight + 1)) {
		targetOneFeeHistory.NextBaseFee = misc.CalcBaseFee(cfg, b.CurrentHeader())
	} else {
		targetOneFeeHistory.NextBaseFee = new(big.Int)
	}
	// set gas used ratio
	gasLimitUint64, ok := (*ethBlock)["gasLimit"].(hexutil.Uint64)
	if !ok {
		return fmt.Errorf("invalid gas limit type: %T", (*ethBlock)["gasLimit"])
	}

	gasUsedBig, ok := (*ethBlock)["gasUsed"].(*hexutil.Big)
	if !ok {
		return fmt.Errorf("invalid gas used type: %T", (*ethBlock)["gasUsed"])
	}

	gasusedfloat, _ := new(big.Float).SetInt(gasUsedBig.ToInt()).Float64()

	if gasLimitUint64 <= 0 {
		return fmt.Errorf("gasLimit of block height %d should be bigger than 0 , current gaslimit %d", blockHeight, gasLimitUint64)
	}

	gasUsedRatio := gasusedfloat / float64(gasLimitUint64)
	blockGasUsed := gasusedfloat
	targetOneFeeHistory.GasUsedRatio = gasUsedRatio

	rewardCount := len(rewardPercentiles)
	targetOneFeeHistory.Reward = make([]*big.Int, rewardCount)
	for i := 0; i < rewardCount; i++ {
		targetOneFeeHistory.Reward[i] = big.NewInt(0)
	}

	// check tendermintTxs
	tendermintTxs := tendermintBlock.Block.Txs
	tendermintTxResults := tendermintBlockResult.TxsResults
	tendermintTxCount := len(tendermintTxs)

	var sorter sortGasAndReward

	for i := 0; i < tendermintTxCount; i++ {
		eachTendermintTx := tendermintTxs[i]
		eachTendermintTxResult := tendermintTxResults[i]

		tx, err := b.clientCtx.TxConfig.TxDecoder()(eachTendermintTx)
		if err != nil {
			b.logger.Debug("failed to decode transaction in block", "height", blockHeight, "error", err.Error())
			continue
		}
		txGasUsed := uint64(eachTendermintTxResult.GasUsed) // #nosec G701
		for _, msg := range tx.GetMsgs() {
			ethMsg, ok := msg.(*evmtypes.MsgEthereumTx)
			if !ok {
				continue
			}
			tx := ethMsg.AsTransaction()
			reward := tx.EffectiveGasTipValue(blockBaseFee)
			if reward == nil {
				reward = big.NewInt(0)
			}
			sorter = append(sorter, txGasAndReward{gasUsed: txGasUsed, reward: reward})
		}
	}

	// return an all zero row if there are no transactions to gather data from
	ethTxCount := len(sorter)
	if ethTxCount == 0 {
		return nil
	}

	sort.Sort(sorter)

	var txIndex int
	sumGasUsed := sorter[0].gasUsed

	for i, p := range rewardPercentiles {
		thresholdGasUsed := uint64(blockGasUsed * p / 100) // #nosec G701
		for sumGasUsed < thresholdGasUsed && txIndex < ethTxCount-1 {
			txIndex++
			sumGasUsed += sorter[txIndex].gasUsed
		}
		targetOneFeeHistory.Reward[i] = sorter[txIndex].reward
	}

	return nil
}

// AllTxLogsFromEvents parses all ethereum logs from cosmos events
func AllTxLogsFromEvents(events []abci.Event) ([][]*ethtypes.Log, error) {
	allLogs := make([][]*ethtypes.Log, 0, 4)
	for _, event := range events {
		if event.Type != evmtypes.EventTypeTxReceipt {
			continue
		}

		receipt, err := ParseTxReceiptFromEvent(event)
		if err != nil {
			return nil, err
		}

		allLogs = append(allLogs, receipt.Logs)
	}
	return allLogs, nil
}

// InCompletedEthReceipt holds an in-completed Ethereum receipt, missing:
// - Block hash in receipt.
// - Block hash and log index in logs.
type InCompletedEthReceipt struct {
	*ethtypes.Receipt
}

// Fill the missing fields for the receipt
func (r *InCompletedEthReceipt) Fill(blockHash common.Hash, startLogIndex uint) {
	r.Receipt.BlockHash = blockHash

	idx := startLogIndex
	for _, log := range r.Receipt.Logs {
		log.BlockHash = blockHash
		log.Index = idx
		idx++
	}
}

// TxReceiptFromEvent parses ethereum receipt from cosmos events
func TxReceiptFromEvent(events []abci.Event) (*InCompletedEthReceipt, error) {
	for _, event := range events {
		if event.Type == evmtypes.EventTypeTxReceipt {
			return ParseTxReceiptFromEvent(event)
		}
	}

	return nil, nil
}

// ParseTxReceiptFromEvent parse tx receipt from one event.
// The output receipt will be:
// - Missing block hash in receipt.
// - Missing block hash and log index in logs.
func ParseTxReceiptFromEvent(event abci.Event) (*InCompletedEthReceipt, error) {
	if event.Type != evmtypes.EventTypeTxReceipt {
		panic(fmt.Sprintf("wrong event, expected: %s, got: %s", evmtypes.EventTypeTxReceipt, event.Type))
	}

	marshalledReceiptRaw, found := findAttribute(event.Attributes, evmtypes.AttributeKeyReceiptMarshalled)
	if !found {
		return nil, fmt.Errorf("missing event attribute: %s", evmtypes.AttributeKeyReceiptMarshalled)
	}
	bzReceipt, err := hexutil.Decode(marshalledReceiptRaw)
	if err != nil {
		return nil, err
	}
	receipt := &ethtypes.Receipt{}
	if err := receipt.UnmarshalBinary(bzReceipt); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal receipt")
	}

	txHashRaw, found := findAttribute(event.Attributes, evmtypes.AttributeKeyReceiptTxHash)
	if !found {
		return nil, fmt.Errorf("missing event attribute: %s", evmtypes.AttributeKeyReceiptTxHash)
	}
	txHash := common.HexToHash(txHashRaw)

	blockNumberRaw, found := findAttribute(event.Attributes, evmtypes.AttributeKeyReceiptBlockNumber)
	if !found {
		return nil, fmt.Errorf("missing event attribute: %s", evmtypes.AttributeKeyReceiptBlockNumber)
	}
	blockNumber, err := strconv.ParseUint(blockNumberRaw, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("bad event attribute value: %s = %s", evmtypes.AttributeKeyReceiptBlockNumber, blockNumberRaw)
	}

	txIndexRaw, found := findAttribute(event.Attributes, evmtypes.AttributeKeyReceiptTxIndex)
	if !found {
		return nil, fmt.Errorf("missing event attribute: %s", evmtypes.AttributeKeyReceiptTxIndex)
	}
	txIndex, err := strconv.ParseUint(txIndexRaw, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("bad event attribute value: %s = %s", evmtypes.AttributeKeyReceiptTxIndex, txIndexRaw)
	}

	contractAddrRaw, found := findAttribute(event.Attributes, evmtypes.AttributeKeyReceiptContractAddress)
	if !found {
		return nil, fmt.Errorf("missing event attribute: %s", evmtypes.AttributeKeyReceiptContractAddress)
	}
	var contractAddr common.Address
	if contractAddrRaw != "" {
		contractAddr = common.HexToAddress(contractAddrRaw)
	}

	gasUsedRaw, found := findAttribute(event.Attributes, evmtypes.AttributeKeyReceiptGasUsed)
	if !found {
		return nil, fmt.Errorf("missing event attribute: %s", evmtypes.AttributeKeyReceiptGasUsed)
	}
	gasUsed, err := strconv.ParseUint(gasUsedRaw, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("bad event attribute value: %s = %s", evmtypes.AttributeKeyReceiptGasUsed, gasUsedRaw)
	}

	// fill data
	receipt.TxHash = txHash
	receipt.ContractAddress = contractAddr
	receipt.GasUsed = gasUsed
	receipt.BlockNumber = new(big.Int).SetUint64(blockNumber)
	receipt.TransactionIndex = uint(txIndex)

	for _, log := range receipt.Logs {
		log.BlockNumber = blockNumber
		log.TxHash = receipt.TxHash
		log.TxIndex = receipt.TransactionIndex
	}

	return &InCompletedEthReceipt{receipt}, nil
}

// TxLogsFromEvent parses ethereum logs from cosmos events
// TODO LOG: take all from event
func TxLogsFromEvent(events []abci.Event) ([]*ethtypes.Log, error) {
	for _, event := range events {
		if event.Type == evmtypes.EventTypeTxReceipt {
			receipt, err := ParseTxReceiptFromEvent(event)
			if err != nil {
				return nil, err
			}
			return receipt.Logs, nil
		}
	}

	return nil, nil
}

func findAttribute(attrs []abci.EventAttribute, key string) (value string, found bool) {
	for _, attr := range attrs {
		if attr.Key == key {
			value = attr.Value
			found = true
			break
		}
	}
	return
}

// ShouldIgnoreGasUsed returns true if the gasUsed in result should be ignored
// workaround for issue: https://github.com/cosmos/cosmos-sdk/issues/10832
func ShouldIgnoreGasUsed(res *abci.ResponseDeliverTx) bool {
	return res.GetCode() == 11 && strings.Contains(res.GetLog(), "no block gas left to run tx: out of gas")
}

// GetLogsFromBlockResults returns the list of event logs from the tendermint block result response
func GetLogsFromBlockResults(blockRes *tmrpctypes.ResultBlockResults) ([][]*ethtypes.Log, error) {
	blockLogs := [][]*ethtypes.Log{}
	for _, txResult := range blockRes.TxsResults {
		logs, err := AllTxLogsFromEvents(txResult.Events)
		if err != nil {
			return nil, err
		}

		blockLogs = append(blockLogs, logs...)
	}
	return blockLogs, nil
}

// GetHexProofs returns list of hex data of proof op
func GetHexProofs(proof *crypto.ProofOps) []string {
	if proof == nil {
		return []string{""}
	}
	proofs := []string{}
	// check for proof
	for _, p := range proof.Ops {
		proof := ""
		if len(p.Data) > 0 {
			proof = hexutil.Encode(p.Data)
		}
		proofs = append(proofs, proof)
	}
	return proofs
}

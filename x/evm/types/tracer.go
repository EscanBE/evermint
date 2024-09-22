package types

import (
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/eth/tracers/logger"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	corevm "github.com/ethereum/go-ethereum/core/vm"
	ethparams "github.com/ethereum/go-ethereum/params"
)

const (
	TracerAccessList = "access_list"
	TracerJSON       = "json"
	TracerStruct     = "struct"
	TracerMarkdown   = "markdown"
)

// NewTracer creates a new Logger tracer to collect execution traces from an
// EVM transaction.
func NewTracer(tracer string, msg core.Message, cfg *ethparams.ChainConfig, height int64) corevm.EVMLogger {
	// TODO: enable additional log configuration
	logCfg := &logger.Config{
		Debug: true,
	}

	switch tracer {
	case TracerAccessList:
		const mergeNetsplit = true
		preCompiles := corevm.ActivePrecompiles(cfg.Rules(big.NewInt(height), mergeNetsplit))
		return logger.NewAccessListTracer(msg.AccessList(), msg.From(), *msg.To(), preCompiles)
	case TracerJSON:
		return logger.NewJSONLogger(logCfg, os.Stderr)
	case TracerMarkdown:
		return logger.NewMarkdownLogger(logCfg, os.Stdout) // TODO: Stderr ?
	case TracerStruct:
		return logger.NewStructLogger(logCfg)
	default:
		return NewNoOpTracer()
	}
}

// TxTraceResult is the result of a single transaction trace during a block trace.
type TxTraceResult struct {
	Result interface{} `json:"result,omitempty"` // Trace results produced by the tracer
	Error  string      `json:"error,omitempty"`  // Trace failure produced by the tracer
}

var _ corevm.EVMLogger = &NoOpTracer{}

// NoOpTracer is an empty implementation of vm.Tracer interface
type NoOpTracer struct{}

// NewNoOpTracer creates a no-op vm.Tracer
func NewNoOpTracer() *NoOpTracer {
	return &NoOpTracer{}
}

// CaptureStart implements vm.Tracer interface
func (dt NoOpTracer) CaptureStart(_ *corevm.EVM, _, _ common.Address, _ bool, _ []byte, _ uint64, _ *big.Int) {
}

// CaptureState implements vm.Tracer interface
func (dt NoOpTracer) CaptureState(_ uint64, _ corevm.OpCode, _, _ uint64, _ *corevm.ScopeContext, _ []byte, _ int, _ error) {
}

// CaptureFault implements vm.Tracer interface
func (dt NoOpTracer) CaptureFault(_ uint64, _ corevm.OpCode, _, _ uint64, _ *corevm.ScopeContext, _ int, _ error) {
}

// CaptureEnd implements vm.Tracer interface
func (dt NoOpTracer) CaptureEnd(_ []byte, _ uint64, _ time.Duration, _ error) {}

// CaptureEnter implements vm.Tracer interface
func (dt NoOpTracer) CaptureEnter(_ corevm.OpCode, _ common.Address, _ common.Address, _ []byte, _ uint64, _ *big.Int) {
}

// CaptureExit implements vm.Tracer interface
func (dt NoOpTracer) CaptureExit(_ []byte, _ uint64, _ error) {}

// CaptureTxStart implements vm.Tracer interface
func (dt NoOpTracer) CaptureTxStart(_ uint64) {}

// CaptureTxEnd implements vm.Tracer interface
func (dt NoOpTracer) CaptureTxEnd(_ uint64) {}

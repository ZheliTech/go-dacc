// Copyright 2015 The go-dacc Authors
// This file is part of the go-dacc library.
//
// The go-dacc library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-dacc library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-dacc library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"github.com/daccproject/go-dacc/common"
	"github.com/daccproject/go-dacc/consensus"
	"github.com/daccproject/go-dacc/core/state"
	"github.com/daccproject/go-dacc/core/types"
	"github.com/daccproject/go-dacc/core/vm"
	"github.com/daccproject/go-dacc/crypto"
	"github.com/daccproject/go-dacc/log"
	"github.com/daccproject/go-dacc/params"
	"time"
)

// StateProcessor is a basic Processor, which takes care of transitioning
// state from one point to another.
//
// StateProcessor implements Processor.
type StateProcessor struct {
	config *params.ChainConfig // Chain configuration options
	bc     *BlockChain         // Canonical block chain
	engine consensus.Engine    // Consensus engine used for block rewards
}

// NewStateProcessor initialises a new StateProcessor.
func NewStateProcessor(config *params.ChainConfig, bc *BlockChain, engine consensus.Engine) *StateProcessor {
	return &StateProcessor{
		config: config,
		bc:     bc,
		engine: engine,
	}
}

// Process processes the state changes according to the Ethereum rules by running
// the transaction messages using the statedb and applying any rewards to both
// the processor (coinbase) and any included uncles.
//
// Process returns the receipts and logs accumulated during the process and
// returns the amount of gas that was used in the process. If any of the
// transactions failed to execute due to insufficient gas it will return an error.
func (p *StateProcessor) Process(block *types.Block, statedb *state.StateDB, cfg vm.Config) (types.Receipts, []*types.Log, uint64, error) {
	t1 := time.Now()
	var (
		receipts types.Receipts
		usedGas  = new(uint64)
		header   = block.Header()
		allLogs  []*types.Log
		gp       = new(GasPool).AddGas(block.GasLimit())
	)
	// Mutate the block and state according to any hard-fork specs
	//if p.config.DAOForkSupport && p.config.DAOForkBlock != nil && p.config.DAOForkBlock.Cmp(block.Number()) == 0 {
	//	misc.ApplyDAOHardFork(statedb)
	//}
	// Set block dpos context
	// Iterate over and process the individual transactions
	t2 := time.Now()
	var ft int64 = 0
	var at int64 = 0
	var bt int64 = 0
	for i, tx := range block.Transactions() {

		statedb.Prepare(tx.Hash(), block.Hash(), i)

		//receipt, _, err := ApplyTransaction(p.config, p.bc, nil, gp, statedb, header, tx, usedGas, cfg)
		receipt, _, err,tf,ta,tb := ApplyTransaction(p.config, block.DposCtx(), p.bc, nil, gp, statedb, header, tx, usedGas, cfg)
		ft += tf
		at += ta
		bt += tb
		if err != nil {
			return nil, nil, 0, err
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
	}
	t3 := time.Now()
	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	//p.engine.Finalize(p.bc, header, statedb, block.Transactions(), block.Uncles(), receipts)
	// TODO(Corbin) [deprecated the uncle block logic]
	// p.engine.Finalize(p.bc, header, statedb, block.Transactions(), block.Uncles(), receipts, block.DposCtx())
	p.engine.Finalize(p.bc, header, statedb, block.Transactions(), receipts, block.DposCtx())
	// TODO(Corbin) [deprecated the uncle block logic]
	t4 := time.Now()
	log.Info("Process","t1-2",t2.Sub(t1),"t2-3",t3.Sub(t2),"ft",ft,"at",at,"bt",bt,"t3-4",t4.Sub(t3))
	return receipts, allLogs, *usedGas, nil
}

// ApplyTransaction attempts to apply a transaction to the given state database
// and uses the input parameters for its environment. It returns the receipt
// for the transaction, gas used and an error if the transaction failed,
// indicating the block was invalid.
//func ApplyTransaction(config *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, cfg vm.Config) (*types.Receipt, uint64, error) {
func ApplyTransaction(config *params.ChainConfig, dposContext *types.DposContext, bc *BlockChain, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, cfg vm.Config) (*types.Receipt, uint64, error,int64,int64,int64) {
	t1 := time.Now()
	msg, err := tx.AsMessage(types.MakeSigner(config, header.Number))
	if err != nil {
		return nil, 0, err,0,0,0
	}
	t2 := time.Now()
	if msg.To() == nil && msg.Type() != types.Binary {
		return nil, 0, types.ErrInvalidType,0,0,0
	}

	// Create a new context to be used in the EVM environment
	context := NewEVMContext(msg, header, bc, author)
	// Create a new environment which holds all relevant information
	// about the transaction and calling mechanisms.
	vmenv := vm.NewEVM(context, statedb, config, cfg)
	// Apply the transaction to the current state (included in the env)
	_, gas, failed, err := ApplyMessage(vmenv, msg, gp)
	if err != nil {
		return nil, 0, err,0,0,0
	}
	if msg.Type() != types.Binary {
		if err = applyDposMessage(dposContext, msg); err != nil {
			return nil, 0, err,0,0,0
		}
	}
	t3 := time.Now()

	// Update the state with pending changes
	var root []byte
	if config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(config.IsEIP158(header.Number)).Bytes()
	}
	*usedGas += gas

	// Create a new receipt for the transaction, storing the intermediate root and gas used by the tx
	// based on the eip phase, we're passing whether the root touch-delete accounts.
	receipt := types.NewReceipt(root, failed, *usedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = gas
	// if the transaction created a contract, store the creation address in the receipt.
	if msg.To() == nil {
		receipt.ContractAddress = crypto.CreateAddress(vmenv.Context.Origin, tx.Nonce())
	}
	// Set the receipt logs and create a bloom for filtering
	receipt.Logs = statedb.GetLogs(tx.Hash())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	t4 := time.Now()
	return receipt, gas, err,int64(t2.Sub(t1)),int64(t3.Sub(t2)),int64(t4.Sub(t3))
}

func applyDposMessage(dposContext *types.DposContext, msg types.Message) error {
	switch msg.Type() {
	case types.LoginCandidate:
		dposContext.BecomeCandidate(msg.From())
	case types.LogoutCandidate:
		dposContext.KickoutCandidate(msg.From())
	case types.Delegate:
		dposContext.Delegate(msg.From(), *(msg.To()))
	case types.UnDelegate:
		dposContext.UnDelegate(msg.From(), *(msg.To()))
	default:
		return types.ErrInvalidType
	}
	return nil
}

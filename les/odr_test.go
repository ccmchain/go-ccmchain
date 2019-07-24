// Copyright 2016 The go-ccmchain Authors
// This file is part of the go-ccmchain library.
//
// The go-ccmchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ccmchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ccmchain library. If not, see <http://www.gnu.org/licenses/>.

package les

import (
	"bytes"
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ccmchain/go-ccmchain/common"
	"github.com/ccmchain/go-ccmchain/common/math"
	"github.com/ccmchain/go-ccmchain/core"
	"github.com/ccmchain/go-ccmchain/core/rawdb"
	"github.com/ccmchain/go-ccmchain/core/state"
	"github.com/ccmchain/go-ccmchain/core/types"
	"github.com/ccmchain/go-ccmchain/core/vm"
	"github.com/ccmchain/go-ccmchain/ccmdb"
	"github.com/ccmchain/go-ccmchain/light"
	"github.com/ccmchain/go-ccmchain/params"
	"github.com/ccmchain/go-ccmchain/rlp"
)

type odrTestFn func(ctx context.Context, db ccmdb.Database, config *params.ChainConfig, bc *core.BlockChain, lc *light.LightChain, bhash common.Hash) []byte

func TestOdrGetBlockLes2(t *testing.T) { testOdr(t, 2, 1, true, odrGetBlock) }

func odrGetBlock(ctx context.Context, db ccmdb.Database, config *params.ChainConfig, bc *core.BlockChain, lc *light.LightChain, bhash common.Hash) []byte {
	var block *types.Block
	if bc != nil {
		block = bc.GetBlockByHash(bhash)
	} else {
		block, _ = lc.GetBlockByHash(ctx, bhash)
	}
	if block == nil {
		return nil
	}
	rlp, _ := rlp.EncodeToBytes(block)
	return rlp
}

func TestOdrGetReceiptsLes2(t *testing.T) { testOdr(t, 2, 1, true, odrGetReceipts) }

func odrGetReceipts(ctx context.Context, db ccmdb.Database, config *params.ChainConfig, bc *core.BlockChain, lc *light.LightChain, bhash common.Hash) []byte {
	var receipts types.Receipts
	if bc != nil {
		if number := rawdb.ReadHeaderNumber(db, bhash); number != nil {
			receipts = rawdb.ReadReceipts(db, bhash, *number, config)
		}
	} else {
		if number := rawdb.ReadHeaderNumber(db, bhash); number != nil {
			receipts, _ = light.GetBlockReceipts(ctx, lc.Odr(), bhash, *number)
		}
	}
	if receipts == nil {
		return nil
	}
	rlp, _ := rlp.EncodeToBytes(receipts)
	return rlp
}

func TestOdrAccountsLes2(t *testing.T) { testOdr(t, 2, 1, true, odrAccounts) }

func odrAccounts(ctx context.Context, db ccmdb.Database, config *params.ChainConfig, bc *core.BlockChain, lc *light.LightChain, bhash common.Hash) []byte {
	dummyAddr := common.HexToAddress("1234567812345678123456781234567812345678")
	acc := []common.Address{bankAddr, userAddr1, userAddr2, dummyAddr}

	var (
		res []byte
		st  *state.StateDB
		err error
	)
	for _, addr := range acc {
		if bc != nil {
			header := bc.GetHeaderByHash(bhash)
			st, err = state.New(header.Root, state.NewDatabase(db))
		} else {
			header := lc.GetHeaderByHash(bhash)
			st = light.NewState(ctx, header, lc.Odr())
		}
		if err == nil {
			bal := st.GetBalance(addr)
			rlp, _ := rlp.EncodeToBytes(bal)
			res = append(res, rlp...)
		}
	}
	return res
}

func TestOdrContractCallLes2(t *testing.T) { testOdr(t, 2, 2, true, odrContractCall) }

type callmsg struct {
	types.Message
}

func (callmsg) CheckNonce() bool { return false }

func odrContractCall(ctx context.Context, db ccmdb.Database, config *params.ChainConfig, bc *core.BlockChain, lc *light.LightChain, bhash common.Hash) []byte {
	data := common.Hex2Bytes("60CD26850000000000000000000000000000000000000000000000000000000000000000")

	var res []byte
	for i := 0; i < 3; i++ {
		data[35] = byte(i)
		if bc != nil {
			header := bc.GetHeaderByHash(bhash)
			statedb, err := state.New(header.Root, state.NewDatabase(db))

			if err == nil {
				from := statedb.GetOrNewStateObject(bankAddr)
				from.SetBalance(math.MaxBig256)

				msg := callmsg{types.NewMessage(from.Address(), &testContractAddr, 0, new(big.Int), 100000, new(big.Int), data, false)}

				context := core.NewEVMContext(msg, header, bc, nil)
				vmenv := vm.NewEVM(context, statedb, config, vm.Config{})

				//vmenv := core.NewEnv(statedb, config, bc, msg, header, vm.Config{})
				gp := new(core.GasPool).AddGas(math.MaxUint64)
				ret, _, _, _ := core.ApplyMessage(vmenv, msg, gp)
				res = append(res, ret...)
			}
		} else {
			header := lc.GetHeaderByHash(bhash)
			state := light.NewState(ctx, header, lc.Odr())
			state.SetBalance(bankAddr, math.MaxBig256)
			msg := callmsg{types.NewMessage(bankAddr, &testContractAddr, 0, new(big.Int), 100000, new(big.Int), data, false)}
			context := core.NewEVMContext(msg, header, lc, nil)
			vmenv := vm.NewEVM(context, state, config, vm.Config{})
			gp := new(core.GasPool).AddGas(math.MaxUint64)
			ret, _, _, _ := core.ApplyMessage(vmenv, msg, gp)
			if state.Error() == nil {
				res = append(res, ret...)
			}
		}
	}
	return res
}

func TestOdrTxStatusLes2(t *testing.T) { testOdr(t, 2, 1, false, odrTxStatus) }

func odrTxStatus(ctx context.Context, db ccmdb.Database, config *params.ChainConfig, bc *core.BlockChain, lc *light.LightChain, bhash common.Hash) []byte {
	var txs types.Transactions
	if bc != nil {
		block := bc.GetBlockByHash(bhash)
		txs = block.Transactions()
	} else {
		if block, _ := lc.GetBlockByHash(ctx, bhash); block != nil {
			btxs := block.Transactions()
			txs = make(types.Transactions, len(btxs))
			for i, tx := range btxs {
				var err error
				txs[i], _, _, _, err = light.GetTransaction(ctx, lc.Odr(), tx.Hash())
				if err != nil {
					return nil
				}
			}
		}
	}
	rlp, _ := rlp.EncodeToBytes(txs)
	return rlp
}

// testOdr tests odr requests whose validation guaranteed by block headers.
func testOdr(t *testing.T, protocol int, expFail uint64, checkCached bool, fn odrTestFn) {
	// Assemble the test environment
	server, client, tearDown := newClientServerEnv(t, 4, protocol, nil, true)
	defer tearDown()
	client.pm.synchronise(client.rPeer)

	test := func(expFail uint64) {
		// Mark this as a helper to put the failures at the correct lines
		t.Helper()

		for i := uint64(0); i <= server.pm.blockchain.CurrentHeader().Number.Uint64(); i++ {
			bhash := rawdb.ReadCanonicalHash(server.db, i)
			b1 := fn(light.NoOdr, server.db, server.pm.chainConfig, server.pm.blockchain.(*core.BlockChain), nil, bhash)

			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			b2 := fn(ctx, client.db, client.pm.chainConfig, nil, client.pm.blockchain.(*light.LightChain), bhash)

			eq := bytes.Equal(b1, b2)
			exp := i < expFail
			if exp && !eq {
				t.Fatalf("odr mismatch: have %x, want %x", b2, b1)
			}
			if !exp && eq {
				t.Fatalf("unexpected odr match")
			}
		}
	}
	// temporarily remove peer to test odr fails
	// expect retrievals to fail (except genesis block) without a les peer
	client.peers.Unregister(client.rPeer.id)
	time.Sleep(time.Millisecond * 10) // ensure that all peerSetNotify callbacks are executed
	test(expFail)

	// expect all retrievals to pass
	client.peers.Register(client.rPeer)
	time.Sleep(time.Millisecond * 10) // ensure that all peerSetNotify callbacks are executed
	client.peers.lock.Lock()
	client.rPeer.hasBlock = func(common.Hash, uint64, bool) bool { return true }
	client.peers.lock.Unlock()
	test(5)
	if checkCached {
		// still expect all retrievals to pass, now data should be cached locally
		client.peers.Unregister(client.rPeer.id)
		time.Sleep(time.Millisecond * 10) // ensure that all peerSetNotify callbacks are executed
		test(5)
	}
}

// Copyright 2019 The go-ccmchain Authors
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
	"math/big"
	"testing"

	"net"

	"github.com/ccmchain/go-ccmchain/common"
	"github.com/ccmchain/go-ccmchain/core/types"
	"github.com/ccmchain/go-ccmchain/crypto"
	"github.com/ccmchain/go-ccmchain/p2p"
	"github.com/ccmchain/go-ccmchain/p2p/enode"
)

func TestFetcherULCPeerSelector(t *testing.T) {
	id1 := newNodeID(t).ID()
	id2 := newNodeID(t).ID()
	id3 := newNodeID(t).ID()
	id4 := newNodeID(t).ID()

	ftn1 := &fetcherTreeNode{
		hash: common.HexToHash("1"),
		td:   big.NewInt(1),
	}
	ftn2 := &fetcherTreeNode{
		hash:   common.HexToHash("2"),
		td:     big.NewInt(2),
		parent: ftn1,
	}
	ftn3 := &fetcherTreeNode{
		hash:   common.HexToHash("3"),
		td:     big.NewInt(3),
		parent: ftn2,
	}
	lf := lightFetcher{
		pm: &ProtocolManager{
			ulc: &ulc{
				keys: map[string]bool{
					id1.String(): true,
					id2.String(): true,
					id3.String(): true,
					id4.String(): true,
				},
				fraction: 70,
			},
		},
		maxConfirmedTd: ftn1.td,

		peers: map[*peer]*fetcherPeerInfo{
			{
				id:      "peer1",
				Peer:    p2p.NewPeer(id1, "peer1", []p2p.Cap{}),
				trusted: true,
			}: {
				nodeByHash: map[common.Hash]*fetcherTreeNode{
					ftn1.hash: ftn1,
					ftn2.hash: ftn2,
				},
			},
			{
				Peer:    p2p.NewPeer(id2, "peer2", []p2p.Cap{}),
				id:      "peer2",
				trusted: true,
			}: {
				nodeByHash: map[common.Hash]*fetcherTreeNode{
					ftn1.hash: ftn1,
					ftn2.hash: ftn2,
				},
			},
			{
				id:      "peer3",
				Peer:    p2p.NewPeer(id3, "peer3", []p2p.Cap{}),
				trusted: true,
			}: {
				nodeByHash: map[common.Hash]*fetcherTreeNode{
					ftn1.hash: ftn1,
					ftn2.hash: ftn2,
					ftn3.hash: ftn3,
				},
			},
			{
				id:      "peer4",
				Peer:    p2p.NewPeer(id4, "peer4", []p2p.Cap{}),
				trusted: true,
			}: {
				nodeByHash: map[common.Hash]*fetcherTreeNode{
					ftn1.hash: ftn1,
				},
			},
		},
		chain: &lightChainStub{
			tds: map[common.Hash]*big.Int{},
			headers: map[common.Hash]*types.Header{
				ftn1.hash: {},
				ftn2.hash: {},
				ftn3.hash: {},
			},
		},
	}
	bestHash, bestAmount, bestTD, sync := lf.findBestRequest()

	if bestTD == nil {
		t.Fatal("Empty result")
	}

	if bestTD.Cmp(ftn2.td) != 0 {
		t.Fatal("bad td", bestTD)
	}
	if bestHash != ftn2.hash {
		t.Fatal("bad hash", bestTD)
	}

	_, _ = bestAmount, sync
}

type lightChainStub struct {
	BlockChain
	tds                         map[common.Hash]*big.Int
	headers                     map[common.Hash]*types.Header
	insertHeaderChainAssertFunc func(chain []*types.Header, checkFreq int) (int, error)
}

func (l *lightChainStub) GetHeader(hash common.Hash, number uint64) *types.Header {
	if h, ok := l.headers[hash]; ok {
		return h
	}

	return nil
}

func (l *lightChainStub) LockChain()   {}
func (l *lightChainStub) UnlockChain() {}

func (l *lightChainStub) GetTd(hash common.Hash, number uint64) *big.Int {
	if td, ok := l.tds[hash]; ok {
		return td
	}
	return nil
}

func (l *lightChainStub) InsertHeaderChain(chain []*types.Header, checkFreq int) (int, error) {
	return l.insertHeaderChainAssertFunc(chain, checkFreq)
}

func newNodeID(t *testing.T) *enode.Node {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal("generate key err:", err)
	}
	return enode.NewV4(&key.PublicKey, net.IP{}, 35000, 35000)
}
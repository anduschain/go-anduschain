// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"github.com/anduschain/go-anduschain/common"
)

// NewTxsEvent is posted when a batch of transactions enter the transaction pool.
type NewTxsEvent struct{ Txs Transactions }

// NewJoinTxsEvent is posted when a batch of join transactions enter the join transaction pool.
type NewJoinTxsEvent struct{ Txs Transactions }

// PendingLogsEvent is posted pre mining and notifies of pending logs.
type PendingLogsEvent struct {
	Logs []*Log
}

// NewMinedBlockEvent is posted when a block has been imported.
type NewMinedBlockEvent struct{ Block *Block }

// RemovedLogsEvent is posted when a reorg happens
type RemovedLogsEvent struct{ Logs []*Log }

type ChainEvent struct {
	Block *Block
	Hash  common.Hash
	Logs  []*Log
}

type ChainSideEvent struct {
	Block *Block
}

type ChainHeadEvent struct{ Block *Block }

// for deb consensus
type NewLeagueBlockEvent struct {
	Block   *Block
	Address common.Address
	Sign    []byte
}

type FairnodeStatusEvent struct {
	Status  FnStatus
	Payload interface{}
}

type ClientClose struct{}

type BlockVoter struct {
	Voter    common.Address
	VoteSign []byte
}

type VoteBlock struct {
	Block  Block
	Voters []BlockVoter
}

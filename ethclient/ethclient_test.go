// Copyright 2016 The go-ethereum Authors
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

package ethclient

import (
	"bytes"
	"context"
	ethereum "github.com/anduschain/go-anduschain"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/rpc"
	"math/big"
	"testing"
)

// Verify that Client implements the ethereum interfaces.
var (
	_ = ethereum.ChainReader(&Client{})
	_ = ethereum.TransactionReader(&Client{})
	_ = ethereum.ChainStateReader(&Client{})
	_ = ethereum.ChainSyncReader(&Client{})
	_ = ethereum.ContractCaller(&Client{})
	_ = ethereum.GasEstimator(&Client{})
	_ = ethereum.GasPricer(&Client{})
	_ = ethereum.LogFilterer(&Client{})
	_ = ethereum.PendingStateReader(&Client{})
	// _ = ethereum.PendingStateEventer(&Client{})
	_ = ethereum.PendingContractCaller(&Client{})
)

func connectRpc() (*rpc.Client, *Client, error) {
	conn := "http://localhost:8545"
	con, err := rpc.Dial(conn)
	if err != nil {
		return nil, nil, err
	}
	return con, NewClient(con), nil
}

func TestClient_BlockByNumber(t *testing.T) {
	rc, ec, err := connectRpc()
	if err != nil {
		t.Error(err)
	}
	defer rc.Close()

	block, err := ec.BlockByNumber(context.Background(), big.NewInt(10))
	if err != nil {
		t.Error(err)
	}

	js, err := block.Header().MarshalJSON()
	if err != nil {
		t.Error(err)
	}

	t.Log("Header MarshalJSON", string(js))

	var b bytes.Buffer
	err = block.EncodeRLP(&b)
	if err != nil {
		t.Error(err)
	}
	t.Log("rlp string : ", common.Bytes2Hex(b.Bytes()))
}

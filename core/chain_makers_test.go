// Copyright 2015 The go-ethereum Authors
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

package core

import (
	"fmt"
	"github.com/anduschain/go-anduschain/consensus/deb"
	"github.com/anduschain/go-anduschain/core/types"
	"math/big"

	"github.com/anduschain/go-anduschain/core/vm"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/ethdb"
	"github.com/anduschain/go-anduschain/params"
)

func ExampleGenerateChain() {
	var (
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		key2, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
		key3, _ = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		addr2   = crypto.PubkeyToAddress(key2.PublicKey)
		addr3   = crypto.PubkeyToAddress(key3.PublicKey)
		db      = ethdb.NewMemDatabase()
	)

	gspec := DefaultGenesisForTesting()
	gspec.Alloc = GenesisAlloc{
		addr1: {Balance: big.NewInt(1000000)},
		addr2: {Balance: big.NewInt(1000000)},
		addr3: {Balance: big.NewInt(100)},
	}

	genesis := gspec.MustCommit(db)

	// This call generates a chain of 5 blocks. The function runs for
	// each block and adds different features to gen based on the
	// block index.
	signer := types.NewEIP155Signer(gspec.Config.ChainID)
	chain, _ := GenerateChain(gspec.Config, genesis, deb.NewFaker(), db, 5, func(i int, gen *BlockGen) {
		switch i {
		case 0:
			gen.SetCoinbase(addr2)
			// In block 1, addr1 sends addr2 some ether.
			tx, _ := types.SignTx(types.NewTransaction(gen.TxNonce(addr1), addr2, big.NewInt(10000), params.TxGas, nil, nil), signer, key1)
			jtx, _ := types.SignTx(types.NewJoinTransaction(gen.TxNonce(addr2), []byte("otprn")), signer, key2)
			gen.AddTx(tx)
			gen.AddTx(jtx)
		case 1:
			// In block 2, addr1 sends some more ether to addr2.
			// addr2 passes it on to addr3.
			gen.SetCoinbase(addr3)
			tx1, _ := types.SignTx(types.NewTransaction(gen.TxNonce(addr1), addr2, big.NewInt(1000), params.TxGas, nil, nil), signer, key1)
			tx2, _ := types.SignTx(types.NewTransaction(gen.TxNonce(addr2), addr3, big.NewInt(1000), params.TxGas, nil, nil), signer, key2)

			jtx, _ := types.SignTx(types.NewJoinTransaction(gen.TxNonce(addr3), []byte("otprn")), signer, key3)
			jtx2, _ := types.SignTx(types.NewJoinTransaction(gen.TxNonce(addr2), []byte("otprn")), signer, key2)
			gen.AddTx(tx1)
			gen.AddTx(tx2)
			gen.AddTx(jtx)
			gen.AddTx(jtx2) // joinnonce == 1
		case 2:
			// Block 3 is empty but was mined by addr3.
			//gen.SetCoinbase(addr3)
			//gen.SetExtra([]byte("yeehaw"))
		case 3:
			// Block 4 includes blocks 2 and 3 as uncle headers (with modified extra data).
			//b2 := gen.PrevBlock(1).Header()
			//b2.Extra = []byte("foo")
			//gen.AddUncle(b2)
			//b3 := gen.PrevBlock(2).Header()
			//b3.Extra = []byte("foo")
			//gen.AddUncle(b3)
		}
	})

	// Import the chain. This runs all block validation rules.
	blockchain, _ := NewBlockChain(db, nil, gspec.Config, deb.NewFaker(), vm.Config{})
	defer blockchain.Stop()

	if i, err := blockchain.InsertChain(chain); err != nil {
		fmt.Printf("insert error (block %d): %v\n", chain[i].NumberU64(), err)
		return
	}

	state, _ := blockchain.State()
	fmt.Printf("last block: #%d\n", blockchain.CurrentBlock().Number())
	fmt.Println("balance of addr1:", state.GetBalance(addr1))
	fmt.Println("balance of addr2:", state.GetBalance(addr2))
	fmt.Println("balance of addr3:", state.GetBalance(addr3))

	fmt.Println("joinNonce of addr1:", state.GetJoinNonce(addr1))
	fmt.Println("joinNonce of addr2:", state.GetJoinNonce(addr2))
	fmt.Println("joinNonce of addr3:", state.GetJoinNonce(addr3))

	// Output:
	// last block: #5
	// balance of addr1: 989000
	// balance of addr2: 1010000
	// balance of addr3: 1100
	// joinNonce of addr1: 0
	// joinNonce of addr2: 1
	// joinNonce of addr3: 0
}

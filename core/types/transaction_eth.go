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
	"math/big"
)

type txEthdata struct {
	AccountNonce uint64          `json:"nonce"    gencodec:"required"`
	Price        *big.Int        `json:"gasPrice" gencodec:"required"`
	GasLimit     uint64          `json:"gas"      gencodec:"required"`
	Recipient    *common.Address `json:"to"       rlp:"nil"` // nil means contract creation
	Amount       *big.Int        `json:"value"    gencodec:"required"`
	Payload      []byte          `json:"input"    gencodec:"required"`

	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`

	// This is only used when marshaling to JSON.
	Hash *common.Hash `json:"hash" rlp:"-"`
}

func (t txEthdata) txType() uint64 {
	return EthTx
}

func (t txEthdata) copy() TxData {
	cpy := &txEthdata{
		AccountNonce: t.AccountNonce,
		Recipient:    copyAddressPtr(t.Recipient),
		Payload:      common.CopyBytes(t.Payload),
		GasLimit:     t.GasLimit,
		// These are initialized below.
		Amount: new(big.Int),
		Price:  new(big.Int),
		V:      new(big.Int),
		R:      new(big.Int),
		S:      new(big.Int),
	}
	if t.Amount != nil {
		cpy.Amount.Set(t.Amount)
	}
	if t.Price != nil {
		cpy.Price.Set(t.Price)
	}
	if t.V != nil {
		cpy.V.Set(t.V)
	}
	if t.R != nil {
		cpy.R.Set(t.R)
	}
	if t.S != nil {
		cpy.S.Set(t.S)
	}
	return cpy
}

func (t txEthdata) chainID() *big.Int {
	return deriveChainId(t.V)
}

func (t txEthdata) data() []byte {
	return t.Payload
}

func (t txEthdata) gas() uint64 {
	return t.GasLimit
}

func (t txEthdata) gasPrice() *big.Int {
	return t.Price
}

func (t txEthdata) gasTipCap() *big.Int {
	return t.Price
}

func (t txEthdata) gasFeeCap() *big.Int {
	return t.Price
}

func (t txEthdata) value() *big.Int {
	return t.Amount
}

func (t txEthdata) nonce() uint64 {
	return t.AccountNonce
}

func (t txEthdata) to() *common.Address {
	return t.Recipient
}

func (t txEthdata) rawSignatureValues() (v, r, s *big.Int) {
	return t.V, t.R, t.S
}

func (t txEthdata) setSignatureValues(chainID, v, r, s *big.Int) {
	t.V, t.R, t.S = v, r, s
}

func (tx txEthdata) TxData() TxData {
	cpy := &LegacyTx{
		Type:         EthTx,
		AccountNonce: tx.AccountNonce,
		Recipient:    copyAddressPtr(tx.Recipient),
		Payload:      common.CopyBytes(tx.Payload),
		GasLimit:     tx.GasLimit,
		// These are initialized below.
		Amount: new(big.Int),
		Price:  new(big.Int),
		V:      new(big.Int),
		R:      new(big.Int),
		S:      new(big.Int),
	}
	if tx.Amount != nil {
		cpy.Amount.Set(tx.Amount)
	}
	if tx.Price != nil {
		cpy.Price.Set(tx.Price)
	}
	if tx.V != nil {
		cpy.V.Set(tx.V)
	}
	if tx.R != nil {
		cpy.R.Set(tx.R)
	}
	if tx.S != nil {
		cpy.S.Set(tx.S)
	}
	return cpy
}

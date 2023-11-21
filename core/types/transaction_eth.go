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
	"bytes"
	"io"
	"math/big"
	"sync/atomic"

	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/common/hexutil"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/rlp"
)

//go:generate gencodec -dir . -type txEthdata -field-override txEthdataMarshaling -out eth_tx_json.go

type TransactionEth struct {
	data txEthdata
	// caches
	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

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

type txEthdataMarshaling struct {
	AccountNonce hexutil.Uint64
	Price        *hexutil.Big
	GasLimit     hexutil.Uint64
	Amount       *hexutil.Big
	Payload      hexutil.Bytes
	V            *hexutil.Big
	R            *hexutil.Big
	S            *hexutil.Big
}

// ChainId returns which chain id this transaction was signed for (if at all)
func (tx *TransactionEth) ChainId() *big.Int {
	return deriveChainId(tx.data.V)
}

// Protected returns whether the transaction is protected from replay protection.
func (tx *TransactionEth) Protected() bool {
	return isProtectedV(tx.data.V)
}

// EncodeRLP implements rlp.Encoder
func (tx *TransactionEth) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &tx.data)
}

// DecodeRLP implements rlp.Decoder
func (tx *TransactionEth) DecodeRLP(s *rlp.Stream) error {
	_, size, _ := s.Kind()
	err := s.Decode(&tx.data)
	if err == nil {
		tx.size.Store(common.StorageSize(rlp.ListSize(size)))
	}

	return err
}

// MarshalJSON encodes the web3 RPC transaction format.
func (tx *TransactionEth) MarshalJSON() ([]byte, error) {
	hash := tx.Hash()
	data := tx.data
	data.Hash = &hash
	return data.MarshalJSON()
}

// UnmarshalJSON decodes the web3 RPC transaction format.
func (tx *TransactionEth) UnmarshalJSON(input []byte) error {
	var dec txEthdata
	if err := dec.UnmarshalJSON(input); err != nil {
		return err
	}
	var V byte
	if isProtectedV(dec.V) {
		chainID := deriveChainId(dec.V).Uint64()
		V = byte(dec.V.Uint64() - 35 - 2*chainID)
	} else {
		V = byte(dec.V.Uint64() - 27)
	}
	if !crypto.ValidateSignatureValues(V, dec.R, dec.S, false) {
		return ErrInvalidSig
	}
	*tx = TransactionEth{data: dec}
	return nil
}

func (tx *TransactionEth) Data() []byte       { return common.CopyBytes(tx.data.Payload) }
func (tx *TransactionEth) Gas() uint64        { return tx.data.GasLimit }
func (tx *TransactionEth) GasPrice() *big.Int { return new(big.Int).Set(tx.data.Price) }
func (tx *TransactionEth) Value() *big.Int    { return new(big.Int).Set(tx.data.Amount) }
func (tx *TransactionEth) Nonce() uint64      { return tx.data.AccountNonce }
func (tx *TransactionEth) CheckNonce() bool   { return true }

// To returns the recipient address of the transaction.
// It returns nil if the transaction is a contract creation.
func (tx *TransactionEth) To() *common.Address {
	if tx.data.Recipient == nil {
		return nil
	}
	to := *tx.data.Recipient
	return &to
}

func (tx *TransactionEth) Payload() []byte {
	return tx.data.Payload
}

func (tx *TransactionEth) Vrs() (*big.Int, *big.Int, *big.Int) {
	return tx.data.V, tx.data.R, tx.data.S
}

// Hash hashes the RLP encoding of tx.
// It uniquely identifies the transaction.
func (tx *TransactionEth) Hash() common.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := rlpHash(tx)
	tx.hash.Store(v)
	return v
}

// Size returns the true RLP encoded storage size of the transaction, either by
// encoding and returning it, or returning a previsouly cached value.
func (tx *TransactionEth) Size() common.StorageSize {
	if size := tx.size.Load(); size != nil {
		return size.(common.StorageSize)
	}
	c := writeCounter(0)
	rlp.Encode(&c, &tx.data)
	tx.size.Store(common.StorageSize(c))
	return common.StorageSize(c)
}

// Cost returns amount + gasprice * gaslimit.
func (tx *TransactionEth) Cost() *big.Int {
	total := new(big.Int).Mul(tx.data.Price, new(big.Int).SetUint64(tx.data.GasLimit))
	total.Add(total, tx.data.Amount)
	return total
}

func (tx *TransactionEth) RawSignatureValues() (*big.Int, *big.Int, *big.Int) {
	return tx.data.V, tx.data.R, tx.data.S
}

func (tx *TransactionEth) Transaction() Transaction {
	var rtn Transaction

	rtn.data = tx.data.TxData()

	return rtn
}

// Transactions implements DerivableList for transactions.
type EthTransactions []*TransactionEth

// Len returns the length of s.
func (s EthTransactions) Len() int { return len(s) }

// EncodeIndex encodes the i'th transaction to w. Note that this does not check for errors
// because we assume that *Transaction will only ever contain valid txs that were either
// constructed by decoding or via public API in this package.
func (s EthTransactions) EncodeIndex(i int, w *bytes.Buffer) {
	tx := s[i]
	rlp.Encode(w, tx.data)
}

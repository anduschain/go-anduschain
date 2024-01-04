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
	"encoding/binary"
	"encoding/json"
	"errors"
	"github.com/anduschain/go-anduschain/params"
	"io"
	"math/big"
	"sync/atomic"

	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/common/hexutil"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/rlp"
)

var (
	ErrInvalidSig = errors.New("invalid transaction v, r, s values")
)

type Transaction struct {
	data TxData
	// caches
	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

type TxData interface {
	txType() uint64 // returns the type ID
	copy() TxData   // creates a deep copy and initializes all fields

	chainID() *big.Int
	data() []byte
	gas() uint64
	gasPrice() *big.Int
	gasTipCap() *big.Int
	gasFeeCap() *big.Int
	value() *big.Int
	nonce() uint64
	to() *common.Address

	rawSignatureValues() (v, r, s *big.Int)
	setSignatureValues(chainID, v, r, s *big.Int)
}

type LegacyTx struct {
	Type         uint64          `json:"type"    gencodec:"required"`
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

func (tx *LegacyTx) copy() TxData {
	cpy := &LegacyTx{
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

func (tx *LegacyTx) txType() uint64 {
	return tx.Type
}

func (tx *LegacyTx) chainID() *big.Int {
	return deriveChainId(tx.V)
}

func (tx *LegacyTx) data() []byte {
	return tx.Payload
}

func (tx *LegacyTx) gas() uint64 {
	return tx.GasLimit
}

func (tx *LegacyTx) gasPrice() *big.Int {
	return tx.Price
}

func (tx *LegacyTx) gasTipCap() *big.Int {
	return tx.Price
}

func (tx *LegacyTx) gasFeeCap() *big.Int {
	return tx.Price
}

func (tx *LegacyTx) value() *big.Int {
	return tx.Amount
}

func (tx *LegacyTx) nonce() uint64 {
	return tx.AccountNonce
}

func (tx *LegacyTx) to() *common.Address {
	return tx.Recipient
}

func (tx *LegacyTx) rawSignatureValues() (v, r, s *big.Int) {
	return tx.V, tx.R, tx.S
}

func (tx *LegacyTx) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.V, tx.R, tx.S = v, r, s
}

// NewTx creates a new transaction.
func NewTx(inner TxData) *Transaction {
	tx := new(Transaction)
	tx.setDecoded(inner.copy(), 0)

	return tx
}

func NewTransaction(nonce uint64, to common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	return newTransaction(nonce, &to, amount, gasLimit, gasPrice, data)
}

func NewContractCreation(nonce uint64, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	return newTransaction(nonce, nil, amount, gasLimit, gasPrice, data)
}

func ConvertEthTransaction(nonce uint64, to *common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, R, S, V *big.Int) *Transaction {
	if len(data) > 0 {
		data = common.CopyBytes(data)
	}
	d := &LegacyTx{
		Type:         EthTx,
		AccountNonce: nonce,
		Recipient:    to,
		Payload:      data,
		Amount:       new(big.Int),
		GasLimit:     gasLimit,
		Price:        new(big.Int),
		V:            V,
		R:            R,
		S:            S,
	}
	if amount != nil {
		d.Amount.Set(amount)
	}
	if gasPrice != nil {
		d.Price.Set(gasPrice)
	}

	return &Transaction{data: d}
}

func NewJoinTransaction(nonce, joinNonce uint64, otprn []byte, coinase common.Address) *Transaction {
	if len(otprn) > 0 {
		otprn = common.CopyBytes(otprn)
	} else {
		return nil
	}

	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, joinNonce)
	hash := rlpHash([]interface{}{joinNonce, otprn, coinase})

	var payload []byte
	payload = append(payload, otprn...)
	payload = append(payload, b...)
	payload = append(payload, hash.Bytes()...)

	d := &LegacyTx{
		Type:         JoinTx,
		AccountNonce: nonce,
		Recipient:    &params.JtxAddress,
		Payload:      payload,
		Amount:       new(big.Int),
		GasLimit:     0,
		Price:        new(big.Int),
		V:            new(big.Int),
		R:            new(big.Int),
		S:            new(big.Int),
	}

	return &Transaction{data: d}
}

func newTransaction(nonce uint64, to *common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	if len(data) > 0 {
		data = common.CopyBytes(data)
	}
	d := &LegacyTx{
		Type:         GeneralTx,
		AccountNonce: nonce,
		Recipient:    to,
		Payload:      data,
		Amount:       new(big.Int),
		GasLimit:     gasLimit,
		Price:        new(big.Int),
		V:            new(big.Int),
		R:            new(big.Int),
		S:            new(big.Int),
	}
	if amount != nil {
		d.Amount.Set(amount)
	}
	if gasPrice != nil {
		d.Price.Set(gasPrice)
	}

	return &Transaction{data: d}
}

func (tx *Transaction) TransactionId() uint64 {
	return tx.data.txType()
}

// Type returns the transaction type.
func (tx *Transaction) Type() uint64 {
	data, ok := tx.data.(*LegacyTx)
	if ok {
		return data.txType()
	} else {
		return EthTx
	}

}

// for join transaction
func (tx *Transaction) JoinNonce() (uint64, error) {
	if tx.data.txType() == JoinTx {
		payload := tx.data.data()
		return binary.LittleEndian.Uint64(payload[len(payload)-40 : len(payload)-32]), nil
	}
	return 0, errors.New("not join transaction")
}

// for join transaction
func (tx *Transaction) Otprn() ([]byte, error) {
	if tx.data.txType() == JoinTx {
		payload := tx.data.data()
		return payload[:len(payload)-40], nil
	}
	return nil, errors.New("not join transaction")
}

// for join transaction
func (tx *Transaction) PayloadHash() ([]byte, error) {
	if tx.data.txType() == JoinTx {
		payload := tx.data.data()
		return payload[len(payload)-32:], nil
	}
	return nil, errors.New("not join transaction")
}

func (tx *Transaction) Payload() []byte {
	return tx.data.data()
}

// ChainId returns which chain id this transaction was signed for (if at all)
func (tx *Transaction) ChainId() *big.Int {
	return tx.data.chainID()
}

// Protected returns whether the transaction is protected from replay protection.
func (tx *Transaction) Protected() bool {
	v, _, _ := tx.data.rawSignatureValues()
	return isProtectedV(v)
}

func isProtectedV(V *big.Int) bool {
	if V.BitLen() <= 8 {
		v := V.Uint64()
		return v != 27 && v != 28
	}
	// anything not 27 or 28 is considered protected
	return true
}

// EncodeRLP implements rlp.Encoder
func (tx *Transaction) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &tx.data)
}

// encodeTyped writes the canonical encoding of a typed transaction to w.
func (tx *Transaction) encodeTyped(w *bytes.Buffer) error {
	return rlp.Encode(w, tx.data)
}

// MarshalBinary returns the canonical encoding of the transaction.
// For legacy transactions, it returns the RLP encoding. For EIP-2718 typed
// transactions, it returns the type and payload.
func (tx *Transaction) MarshalBinary() ([]byte, error) {
	return rlp.EncodeToBytes(tx.data)
}

// DecodeRLP implements rlp.Decoder
func (tx *Transaction) DecodeRLP(s *rlp.Stream) error {
	_, size, _ := s.Kind()
	var inner LegacyTx
	err := s.Decode(&inner)

	if err == nil {
		tx.setDecoded(&inner, int(rlp.ListSize(size)))
	} else {
		var inner txEthdata
		err := s.Decode(&inner)

		if err == nil {
			tx.setDecoded(&inner, int(rlp.ListSize(size)))
		}
	}

	return err
}

type ethJSON struct {
	AccountNonce hexutil.Uint64  `json:"nonce"    gencodec:"required"`
	Price        *hexutil.Big    `json:"gasPrice" gencodec:"required"`
	GasLimit     hexutil.Uint64  `json:"gas"      gencodec:"required"`
	Recipient    *common.Address `json:"to"       rlp:"nil"`
	Amount       *hexutil.Big    `json:"value"    gencodec:"required"`
	Payload      hexutil.Bytes   `json:"input"    gencodec:"required"`
	V            *hexutil.Big    `json:"v" gencodec:"required"`
	R            *hexutil.Big    `json:"r" gencodec:"required"`
	S            *hexutil.Big    `json:"s" gencodec:"required"`
	Hash         *common.Hash    `json:"hash" rlp:"-"`
}

type txJSON struct {
	Type         hexutil.Uint64  `json:"type"    gencodec:"required"`
	AccountNonce hexutil.Uint64  `json:"nonce"    gencodec:"required"`
	Price        *hexutil.Big    `json:"gasPrice" gencodec:"required"`
	GasLimit     hexutil.Uint64  `json:"gas"      gencodec:"required"`
	Recipient    *common.Address `json:"to"       rlp:"nil"`
	Amount       *hexutil.Big    `json:"value"    gencodec:"required"`
	Payload      hexutil.Bytes   `json:"input"    gencodec:"required"`
	V            *hexutil.Big    `json:"v" gencodec:"required"`
	R            *hexutil.Big    `json:"r" gencodec:"required"`
	S            *hexutil.Big    `json:"s" gencodec:"required"`
	Hash         *common.Hash    `json:"hash" rlp:"-"`
}

// MarshalJSON encodes the web3 RPC transaction format.
func (tx *Transaction) MarshalJSON() ([]byte, error) {
	var enc txJSON
	hash := tx.Hash()
	enc.Hash = &hash

	t := tx.data
	if tx.Type() == EthTx {
		enc.Type = hexutil.Uint64(EthTx)
	} else {
		enc.Type = hexutil.Uint64(t.txType())
	}
	enc.AccountNonce = hexutil.Uint64(t.nonce())
	enc.Price = (*hexutil.Big)(t.gasPrice())
	enc.GasLimit = hexutil.Uint64(t.gas())
	enc.Recipient = t.to()
	enc.Amount = (*hexutil.Big)(t.value())
	enc.Payload = t.data()
	v, r, s := t.rawSignatureValues()
	enc.V = (*hexutil.Big)(v)
	enc.R = (*hexutil.Big)(r)
	enc.S = (*hexutil.Big)(s)

	return json.Marshal(&enc)
}

// UnmarshalJSON decodes the web3 RPC transaction format.
func (tx *Transaction) UnmarshalJSON(input []byte) error {
	var dec txJSON
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	//if dec.Type == nil {
	//	return errors.New("missing required field 'type' for TxData")
	//}
	var inner TxData
	if dec.Type == hexutil.Uint64(EthTx) {
		var itx txEthdata
		inner = &itx
		//if dec.AccountNonce == nil {
		//	return errors.New("missing required field 'nonce' for TxData")
		//}
		itx.AccountNonce = uint64(dec.AccountNonce)
		if dec.Price == nil {
			return errors.New("missing required field 'gasPrice' for TxData")
		}
		itx.Price = (*big.Int)(dec.Price)
		//if dec.GasLimit == nil {
		//	return errors.New("missing required field 'gas' for TxData")
		//}
		itx.GasLimit = uint64(dec.GasLimit)
		//if dec.Recipient != nil {
		//	t.Recipient = dec.Recipient
		//}
		itx.Recipient = dec.Recipient
		if dec.Amount == nil {
			return errors.New("missing required field 'value' for TxData")
		}
		itx.Amount = (*big.Int)(dec.Amount)
		if dec.Payload == nil {
			return errors.New("missing required field 'input' for TxData")
		}
		itx.Payload = dec.Payload
		if dec.V == nil {
			return errors.New("missing required field 'v' for TxData")
		}
		itx.V = (*big.Int)(dec.V)
		if dec.R == nil {
			return errors.New("missing required field 'r' for TxData")
		}
		itx.R = (*big.Int)(dec.R)
		if dec.S == nil {
			return errors.New("missing required field 's' for TxData")
		}
		itx.S = (*big.Int)(dec.S)
		if dec.Hash != nil {
			itx.Hash = dec.Hash
		}

		var V byte
		if isProtectedV((*big.Int)(dec.V)) {
			chainID := deriveChainId((*big.Int)(dec.V)).Uint64()
			V = byte((*big.Int)(dec.V).Uint64() - 35 - 2*chainID)
		} else {
			V = byte((*big.Int)(dec.V).Uint64() - 27)
		}
		if !crypto.ValidateSignatureValues(V, (*big.Int)(dec.R), (*big.Int)(dec.S), false) {
			return ErrInvalidSig
		}
		*tx = Transaction{data: inner}
	} else {
		var itx LegacyTx
		inner = &itx
		itx.Type = uint64(dec.Type)
		//if dec.AccountNonce == nil {
		//	return errors.New("missing required field 'nonce' for TxData")
		//}
		itx.AccountNonce = uint64(dec.AccountNonce)
		if dec.Price == nil {
			return errors.New("missing required field 'gasPrice' for TxData")
		}
		itx.Price = (*big.Int)(dec.Price)
		//if dec.GasLimit == nil {
		//	return errors.New("missing required field 'gas' for TxData")
		//}
		itx.GasLimit = uint64(dec.GasLimit)
		//if dec.Recipient != nil {
		//	t.Recipient = dec.Recipient
		//}
		itx.Recipient = dec.Recipient
		if dec.Amount == nil {
			return errors.New("missing required field 'value' for TxData")
		}
		itx.Amount = (*big.Int)(dec.Amount)
		if dec.Payload == nil {
			return errors.New("missing required field 'input' for TxData")
		}
		itx.Payload = dec.Payload
		if dec.V == nil {
			return errors.New("missing required field 'v' for TxData")
		}
		itx.V = (*big.Int)(dec.V)
		if dec.R == nil {
			return errors.New("missing required field 'r' for TxData")
		}
		itx.R = (*big.Int)(dec.R)
		if dec.S == nil {
			return errors.New("missing required field 's' for TxData")
		}
		itx.S = (*big.Int)(dec.S)
		if dec.Hash != nil {
			itx.Hash = dec.Hash
		}

		var V byte
		if isProtectedV((*big.Int)(dec.V)) {
			chainID := deriveChainId((*big.Int)(dec.V)).Uint64()
			V = byte((*big.Int)(dec.V).Uint64() - 35 - 2*chainID)
		} else {
			V = byte((*big.Int)(dec.V).Uint64() - 27)
		}
		if !crypto.ValidateSignatureValues(V, (*big.Int)(dec.R), (*big.Int)(dec.S), false) {
			return ErrInvalidSig
		}
		*tx = Transaction{data: inner}
	}

	return nil
}

// UnmarshalBinary decodes the canonical encoding of transactions.
// It supports legacy RLP transactions and Ethereum typed transactions.
func (tx *Transaction) UnmarshalBinary(b []byte) error {
	var inner TxData
	var data LegacyTx
	err := rlp.DecodeBytes(b, &data)
	if err != nil {
		var data2 txEthdata
		err := rlp.DecodeBytes(b, &data2)
		if err != nil {
			return err
		}
		data.Type = EthTx
		data.AccountNonce = data2.AccountNonce
		data.Recipient = copyAddressPtr(data2.Recipient)
		data.Payload = common.CopyBytes(data2.Payload)
		data.GasLimit = data2.GasLimit
		data.Amount = data2.Amount
		data.Price = data2.Price
		data.V = data2.V
		data.R = data2.R
		data.S = data2.S
	}
	inner = &data
	tx.setDecoded(inner, len(b))

	return nil
}

// setDecoded sets the inner transaction and size after decoding.
func (tx *Transaction) setDecoded(inner TxData, size int) {
	tx.data = inner
	if size > 0 {
		tx.size.Store(common.StorageSize(size))
	}
}
func (tx *Transaction) Data() []byte       { return common.CopyBytes(tx.data.data()) }
func (tx *Transaction) Gas() uint64        { return tx.data.gas() }
func (tx *Transaction) GasPrice() *big.Int { return new(big.Int).Set(tx.data.gasPrice()) }
func (tx *Transaction) Value() *big.Int    { return new(big.Int).Set(tx.data.value()) }
func (tx *Transaction) Nonce() uint64      { return tx.data.nonce() }
func (tx *Transaction) CheckNonce() bool   { return true }

// To returns the recipient address of the transaction.
// It returns nil if the transaction is a contract creation.
func (tx *Transaction) To() *common.Address {
	if tx.data.to() == nil {
		return nil
	}
	to := *tx.data.to()
	return &to
}

// Hash hashes the RLP encoding of tx.
// It uniquely identifies the transaction.
func (tx *Transaction) Hash() common.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := rlpHash(tx)
	tx.hash.Store(v)
	return v
}

// Size returns the true RLP encoded storage size of the transaction, either by
// encoding and returning it, or returning a previsouly cached value.
func (tx *Transaction) Size() common.StorageSize {
	if size := tx.size.Load(); size != nil {
		return size.(common.StorageSize)
	}
	c := writeCounter(0)
	rlp.Encode(&c, &tx.data)
	tx.size.Store(common.StorageSize(c))
	return common.StorageSize(c)
}

// AsMessage returns the transaction as a core.Message.
//
// AsMessage requires a signer to derive the sender.
//
// XXX Rename message to something less arbitrary?
func (tx *Transaction) AsMessage(s Signer) (Message, error) {
	msg := Message{
		nonce:      tx.data.nonce(),
		gasLimit:   tx.data.gas(),
		gasPrice:   new(big.Int).Set(tx.data.gasPrice()),
		to:         tx.data.to(),
		amount:     tx.data.value(),
		data:       tx.data.data(),
		checkNonce: true,
	}

	var err error
	msg.from, err = tx.Sender(s)
	return msg, err
}

// WithSignature returns a new transaction with the given signature.
// This signature needs to be formatted as described in the yellow paper (v+27).
func (tx *Transaction) WithSignature(signer Signer, sig []byte) (*Transaction, error) {
	r, s, v, err := signer.SignatureValues(tx, sig)
	if err != nil {
		return nil, err
	}
	cpy := &Transaction{data: tx.data}
	cpy.data.setSignatureValues(signer.ChainID(), v, r, s)
	return cpy, nil
}

// Cost returns amount + gasprice * gaslimit.
func (tx *Transaction) Cost() *big.Int {
	total := new(big.Int).Mul(tx.data.gasPrice(), new(big.Int).SetUint64(tx.data.gas()))
	total.Add(total, tx.data.value())
	return total
}

func (tx *Transaction) RawSignatureValues() (*big.Int, *big.Int, *big.Int) {
	v, r, s := tx.data.rawSignatureValues()
	return v, r, s
}

func (tx *Transaction) Sender(signer Signer) (common.Address, error) {
	if sc := tx.from.Load(); sc != nil {
		sigCache := sc.(sigCache)
		// If the signer used to derive from in a previous
		// call is not the same as used current, invalidate
		// the cache.
		if sigCache.signer.Equal(signer) {
			return sigCache.from, nil
		}
	}
	addr, err := signer.Sender(tx)
	if err != nil {
		return common.Address{}, err
	}
	tx.from.Store(sigCache{signer: signer, from: addr})
	return addr, nil
}

// Message is a fully derived transaction and implements core.Message
//
// NOTE: In a future PR this will be removed.
type Message struct {
	to         *common.Address
	from       common.Address
	nonce      uint64
	amount     *big.Int
	gasLimit   uint64
	gasPrice   *big.Int
	data       []byte
	checkNonce bool
}

func (m Message) IsL1MessageTx() bool {
	//TODO implement me
	panic("implement me")
}

func NewMessage(from common.Address, to *common.Address, nonce uint64, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, checkNonce bool) Message {
	return Message{
		from:       from,
		to:         to,
		nonce:      nonce,
		amount:     amount,
		gasLimit:   gasLimit,
		gasPrice:   gasPrice,
		data:       data,
		checkNonce: checkNonce,
	}
}

func (m Message) From() common.Address { return m.from }
func (m Message) To() *common.Address  { return m.to }
func (m Message) GasPrice() *big.Int   { return m.gasPrice }
func (m Message) Value() *big.Int      { return m.amount }
func (m Message) Gas() uint64          { return m.gasLimit }
func (m Message) Nonce() uint64        { return m.nonce }
func (m Message) Data() []byte         { return m.data }
func (m Message) CheckNonce() bool     { return m.checkNonce }

// copyAddressPtr copies an address.
func copyAddressPtr(a *common.Address) *common.Address {
	if a == nil {
		return nil
	}
	cpy := *a
	return &cpy
}

// IsL1MessageTx returns true if the transaction is an L1 cross-domain tx.
func (tx *Transaction) IsL1MessageTx() bool {
	return tx.Type() == L1MessageTxType
}

// AsL1MessageTx casts the tx into an L1 cross-domain tx.
func (tx *Transaction) AsL1MessageTx() *L1MessageTx {
	if !tx.IsL1MessageTx() {
		return nil
	}
	return tx.data.(*L1MessageTx)
}

// L1MessageQueueIndex returns the L1 queue index if `tx` is of type `L1MessageTx`.
// It returns 0 otherwise.
func (tx *Transaction) L1MessageQueueIndex() uint64 {
	if !tx.IsL1MessageTx() {
		return 0
	}
	return tx.AsL1MessageTx().QueueIndex
}

package transaction

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/common/hexutil"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/rlp"
	"io"
	"math/big"
	"sync/atomic"
)

//go:generate gencodec -type genTxdata -field-override genTxdataMarshaling -out gen_tx_json.go

// GenTransaction is a fully derived transaction and implements Transactioner
type GenTransaction struct {
	data genTxdata
	// caches
	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

type genTxdata struct {
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

type genTxdataMarshaling struct {
	AccountNonce hexutil.Uint64
	Price        *hexutil.Big
	GasLimit     hexutil.Uint64
	Amount       *hexutil.Big
	Payload      hexutil.Bytes
	V            *hexutil.Big
	R            *hexutil.Big
	S            *hexutil.Big
}

func NewGenTransaction(nonce uint64, to common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *GenTransaction {
	return newGenTransaction(nonce, &to, amount, gasLimit, gasPrice, data)
}

func NewContractCreation(nonce uint64, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *GenTransaction {
	return newGenTransaction(nonce, nil, amount, gasLimit, gasPrice, data)
}

func newGenTransaction(nonce uint64, to *common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *GenTransaction {
	if len(data) > 0 {
		data = common.CopyBytes(data)
	}
	d := genTxdata{
		AccountNonce: nonce,
		Recipient:    to,
		Payload:      data,
		Amount:       new(big.Int),
		GasLimit:     gasLimit,
		Price:        new(big.Int),
	}

	d.V, d.R, d.S = new(big.Int), new(big.Int), new(big.Int)

	if amount != nil {
		d.Amount.Set(amount)
	}
	if gasPrice != nil {
		d.Price.Set(gasPrice)
	}

	return &GenTransaction{data: d}
}

// TransactionId returns transaction id
func (gtx *GenTransaction) TransactionId() string {
	return "general"
}

func (gtx *GenTransaction) Sender(signer Signer) (common.Address, error) {
	if sc := gtx.from.Load(); sc != nil {
		sigCache := sc.(sigCache)
		// If the signer used to derive from in a previous
		// call is not the same as used current, invalidate
		// the cache.
		if signer.Equal(signer) {
			return sigCache.from, nil
		}
	}
	addr, err := signer.Sender(gtx)
	if err != nil {
		return common.Address{}, err
	}
	gtx.from.Store(sigCache{signer: signer, from: addr})
	return addr, nil
}

func (gtx *GenTransaction) Signature() Signature {
	return Signature{
		V: gtx.data.V,
		R: gtx.data.R,
		S: gtx.data.S,
	}
}

// ChainId returns which chain id this transaction was signed for (if at all)
func (gtx *GenTransaction) ChainId() *big.Int {
	return DeriveChainId(gtx.data.V)
}

// Protected returns whether the transaction is protected from replay protection.
func (gtx *GenTransaction) Protected() bool {
	return isProtectedV(gtx.data.V)
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
func (gtx *GenTransaction) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &gtx.data)
}

// DecodeRLP implements rlp.Decoder
func (gtx *GenTransaction) DecodeRLP(s *rlp.Stream) error {
	_, size, _ := s.Kind()
	err := s.Decode(&gtx.data)
	if err == nil {
		gtx.size.Store(common.StorageSize(rlp.ListSize(size)))
	}

	return err
}

// MarshalJSON encodes the web3 RPC transaction format.
func (gtx *GenTransaction) MarshalJSON() ([]byte, error) {
	hash := gtx.Hash()
	data := gtx.data
	data.Hash = &hash
	return data.MarshalJSON()
}

// UnmarshalJSON decodes the web3 RPC transaction format.
func (gtx *GenTransaction) UnmarshalJSON(input []byte) error {
	var dec genTxdata
	if err := dec.UnmarshalJSON(input); err != nil {
		return err
	}
	var V byte
	if isProtectedV(dec.V) {
		chainID := DeriveChainId(dec.V).Uint64()
		V = byte(dec.V.Uint64() - 35 - 2*chainID)
	} else {
		V = byte(dec.V.Uint64() - 27)
	}
	if !crypto.ValidateSignatureValues(V, dec.R, dec.S, false) {
		return ErrInvalidSig
	}
	*gtx = GenTransaction{data: dec}
	return nil
}

func (gtx *GenTransaction) Data() []byte       { return common.CopyBytes(gtx.data.Payload) }
func (gtx *GenTransaction) Gas() uint64        { return gtx.data.GasLimit }
func (gtx *GenTransaction) GasPrice() *big.Int { return new(big.Int).Set(gtx.data.Price) }
func (gtx *GenTransaction) Value() *big.Int    { return new(big.Int).Set(gtx.data.Amount) }
func (gtx *GenTransaction) Nonce() uint64      { return gtx.data.AccountNonce }
func (gtx *GenTransaction) CheckNonce() bool   { return true }

func (gtx *GenTransaction) Price() *big.Int { return new(big.Int).Set(gtx.data.Price) }

// To returns the recipient address of the transaction.
// It returns nil if the transaction is a contract creation.
func (gtx *GenTransaction) To() *common.Address {
	if gtx.data.Recipient == nil {
		return nil
	}
	to := *gtx.data.Recipient
	return &to
}

// Hash hashes the RLP encoding of tx.
// It uniquely identifies the transaction.
func (gtx *GenTransaction) Hash() common.Hash {
	if hash := gtx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := RlpHash(gtx)
	gtx.hash.Store(v)
	return v
}

func (gtx *GenTransaction) SigHash(values ...interface{}) common.Hash {
	data := []interface{}{
		gtx.data.AccountNonce,
		gtx.data.Price,
		gtx.data.GasLimit,
		gtx.data.Recipient,
		gtx.data.Amount,
		gtx.data.Payload,
	}

	data = append(data, values...)

	return RlpHash(data)
}

// Size returns the true RLP encoded storage size of the transaction, either by
// encoding and returning it, or returning a previsouly cached value.
func (gtx *GenTransaction) Size() common.StorageSize {
	if size := gtx.size.Load(); size != nil {
		return size.(common.StorageSize)
	}
	c := WriteCounter(0)
	rlp.Encode(&c, &gtx.data)
	gtx.size.Store(common.StorageSize(c))
	return common.StorageSize(c)
}

// AsMessage returns the transaction as a core.Message.
//
// AsMessage requires a signer to derive the sender.
//
// XXX Rename message to something less arbitrary?
func (gtx *GenTransaction) AsMessage(s Signer) (Message, error) {
	msg := Message{
		nonce:      gtx.data.AccountNonce,
		gasLimit:   gtx.data.GasLimit,
		gasPrice:   new(big.Int).Set(gtx.data.Price),
		to:         gtx.data.Recipient,
		amount:     gtx.data.Amount,
		data:       gtx.data.Payload,
		checkNonce: true,
	}

	var err error
	msg.from, err = gtx.Sender(s)
	return msg, err
}

// WithSignature returns a new transaction with the given signature.
// This signature needs to be formatted as described in the yellow paper (v+27).
func (gtx *GenTransaction) WithSignature(signer Signer, sig []byte) (Transactioner, error) {
	r, s, v, err := signer.SignatureValues(gtx, sig)
	if err != nil {
		return nil, err
	}
	cpy := &GenTransaction{data: gtx.data}
	cpy.data.R, cpy.data.S, cpy.data.V = r, s, v
	return cpy, nil
}

// Cost returns amount + gasprice * gaslimit.
func (gtx *GenTransaction) Cost() *big.Int {
	total := new(big.Int).Mul(gtx.data.Price, new(big.Int).SetUint64(gtx.data.GasLimit))
	total.Add(total, gtx.data.Amount)
	return total
}

func (gtx *GenTransaction) RawSignatureValues() (*big.Int, *big.Int, *big.Int) {
	return gtx.data.V, gtx.data.R, gtx.data.S
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

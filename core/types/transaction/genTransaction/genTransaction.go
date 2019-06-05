package genTx

import (
	"container/heap"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/common/hexutil"
	tx "github.com/anduschain/go-anduschain/core/types/transaction"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/rlp"
	"io"
	"math/big"
	"sync/atomic"
)

//go:generate gencodec -type genTxdata -field-override genTxdataMarshaling -out gen_tx_json.go

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

func NewTransaction(nonce uint64, to common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *GenTransaction {
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

func (gtx *GenTransaction) Signature() tx.Signature {
	return tx.Signature{
		V: gtx.data.V,
		R: gtx.data.R,
		S: gtx.data.S,
	}
}

// ChainId returns which chain id this transaction was signed for (if at all)
func (gtx *GenTransaction) ChainId() *big.Int {
	return tx.DeriveChainId(gtx.data.V)
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
		chainID := tx.DeriveChainId(dec.V).Uint64()
		V = byte(dec.V.Uint64() - 35 - 2*chainID)
	} else {
		V = byte(dec.V.Uint64() - 27)
	}
	if !crypto.ValidateSignatureValues(V, dec.R, dec.S, false) {
		return tx.ErrInvalidSig
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

func (gtx *GenTransaction) From() atomic.Value { return gtx.from }
func (gtx *GenTransaction) Price() *big.Int    { return new(big.Int).Set(gtx.data.Price) }

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
	v := tx.RlpHash(gtx)
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

	return tx.RlpHash(data)
}

// Size returns the true RLP encoded storage size of the transaction, either by
// encoding and returning it, or returning a previsouly cached value.
func (gtx *GenTransaction) Size() common.StorageSize {
	if size := gtx.size.Load(); size != nil {
		return size.(common.StorageSize)
	}
	c := tx.WriteCounter(0)
	rlp.Encode(&c, &gtx.data)
	gtx.size.Store(common.StorageSize(c))
	return common.StorageSize(c)
}

// AsMessage returns the transaction as a core.Message.
//
// AsMessage requires a signer to derive the sender.
//
// XXX Rename message to something less arbitrary?
func (gtx *GenTransaction) AsMessage(s tx.Signer) (Message, error) {
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
	msg.from, err = tx.Sender(s, gtx)
	return msg, err
}

// WithSignature returns a new transaction with the given signature.
// This signature needs to be formatted as described in the yellow paper (v+27).
func (gtx *GenTransaction) WithSignature(signer tx.Signer, sig []byte) (tx.Transaction, error) {
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

// TxDifference returns a new set which is the difference between a and b.
func TxDifference(a, b tx.Transactions) tx.Transactions {
	keep := make(tx.Transactions, 0, len(a))

	remove := make(map[common.Hash]struct{})
	for _, tx := range b {
		remove[tx.Hash()] = struct{}{}
	}

	for _, tx := range a {
		if _, ok := remove[tx.Hash()]; !ok {
			keep = append(keep, tx)
		}
	}

	return keep
}

// TxByNonce implements the sort interface to allow sorting a list of transactions
// by their nonces. This is usually only useful for sorting transactions from a
// single account, otherwise a nonce comparison doesn't make much sense.
type TxByNonce tx.Transactions

func (s TxByNonce) Len() int           { return len(s) }
func (s TxByNonce) Less(i, j int) bool { return s[i].Nonce() < s[j].Nonce() }
func (s TxByNonce) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// TxByPrice implements both the sort and the heap interface, making it useful
// for all at once sorting as well as individually adding and removing elements.
type TxByPrice tx.Transactions

func (s TxByPrice) Len() int           { return len(s) }
func (s TxByPrice) Less(i, j int) bool { return s[i].Price().Cmp(s[j].Price()) > 0 }
func (s TxByPrice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func (s *TxByPrice) Push(x interface{}) {
	*s = append(*s, x.(*GenTransaction))
}

func (s *TxByPrice) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0 : n-1]
	return x
}

// TransactionsByPriceAndNonce represents a set of GenTransactions that can return
// transactions in a profit-maximizing sorted order, while supporting removing
// entire batches of transactions for non-executable accounts.
type TransactionsByPriceAndNonce struct {
	txs    map[common.Address]tx.Transactions // Per account nonce-sorted list of transactions
	heads  TxByPrice                          // Next transaction for each unique account (price heap)
	signer tx.Signer                          // Signer for the set of transactions
}

// NewTransactionsByPriceAndNonce creates a transaction set that can retrieve
// price sorted transactions in a nonce-honouring way.
//
// Note, the input map is reowned so the caller should not interact any more with
// if after providing it to the constructor.
func NewTransactionsByPriceAndNonce(signer tx.Signer, txs map[common.Address]tx.Transactions) *TransactionsByPriceAndNonce {
	// Initialize a price based heap with the head transactions

	heads := make(TxByPrice, 0, len(txs))
	for from, accTxs := range txs {
		heads = append(heads, accTxs[0])
		// Ensure the sender address is from the signer
		acc, _ := tx.Sender(signer, accTxs[0])
		txs[acc] = accTxs[1:]
		if from != acc {
			delete(txs, from)
		}
	}
	heap.Init(&heads)

	// Assemble and return the transaction set
	return &TransactionsByPriceAndNonce{
		txs:    txs,
		heads:  heads,
		signer: signer,
	}
}

// Peek returns the next transaction by price.
func (t *TransactionsByPriceAndNonce) Peek() tx.Transaction {
	if len(t.heads) == 0 {
		return nil
	}
	return t.heads[0]
}

// Shift replaces the current best head with the next one from the same account.
func (t *TransactionsByPriceAndNonce) Shift() {
	acc, _ := tx.Sender(t.signer, t.heads[0])
	if txs, ok := t.txs[acc]; ok && len(txs) > 0 {
		t.heads[0], t.txs[acc] = txs[0], txs[1:]
		heap.Fix(&t.heads, 0)
	} else {
		heap.Pop(&t.heads)
	}
}

// Pop removes the best transaction, *not* replacing it with the next one from
// the same account. This should be used when a transaction cannot be executed
// and hence all subsequent ones should be discarded from the same account.
func (t *TransactionsByPriceAndNonce) Pop() {
	heap.Pop(&t.heads)
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

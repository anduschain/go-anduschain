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

//go:generate gencodec -type joinTxdata -field-override joinTxdataMarshaling -out join_tx_json.go

// JoinTranasction is a fully derived transaction and implements Transactioner
type JoinTranasction struct {
	data joinTxdata
	// caches
	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

type joinTxdata struct {
	JoinNonce uint64   `json:"nonce"    gencodec:"required"`
	Fee       *big.Int `json:"fee"    gencodec:"required"`
	Otprn     []byte   `json:"otprn"    gencodec:"required"`

	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`

	// This is only used when marshaling to JSON.
	Hash *common.Hash `json:"hash" rlp:"-"`
}

type joinTxdataMarshaling struct {
	JoinNonce hexutil.Uint64
	Fee       *hexutil.Big
	Otprn     hexutil.Bytes
	V         *hexutil.Big
	R         *hexutil.Big
	S         *hexutil.Big
}

func NewJoinTransaction(joinNonce uint64, fee *big.Int, otprn []byte) *JoinTranasction {
	if len(otprn) > 0 {
		otprn = common.CopyBytes(otprn)
	} else {
		return nil
	}

	d := joinTxdata{
		JoinNonce: joinNonce,
		Otprn:     otprn,
		Fee:       fee,
	}

	d.V, d.R, d.S = new(big.Int), new(big.Int), new(big.Int)

	return &JoinTranasction{data: d}
}

func (tx *JoinTranasction) TransactionId() string {
	return "join"
}

func (tx *JoinTranasction) ChainId() *big.Int {
	return DeriveChainId(tx.data.V)
}
func (tx *JoinTranasction) Protected() bool {
	return isProtectedV(tx.data.V)
}
func (tx *JoinTranasction) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &tx.data)
}
func (tx *JoinTranasction) DecodeRLP(s *rlp.Stream) error {
	_, size, _ := s.Kind()
	err := s.Decode(&tx.data)
	if err == nil {
		tx.size.Store(common.StorageSize(rlp.ListSize(size)))
	}
	return err
}
func (tx *JoinTranasction) MarshalJSON() ([]byte, error) {
	hash := tx.Hash()
	data := tx.data
	data.Hash = &hash
	return data.MarshalJSON()
}
func (tx *JoinTranasction) UnmarshalJSON(input []byte) error {
	var dec joinTxdata
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
	*tx = JoinTranasction{data: dec}
	return nil

}
func (tx *JoinTranasction) Data() []byte        { return common.CopyBytes(tx.data.Otprn) }
func (tx *JoinTranasction) Value() *big.Int     { return tx.data.Fee }
func (tx *JoinTranasction) Nonce() uint64       { return tx.data.JoinNonce }
func (tx *JoinTranasction) CheckNonce() bool    { return true }
func (tx *JoinTranasction) Price() *big.Int     { return nil }
func (tx *JoinTranasction) Cost() *big.Int      { return nil }
func (tx *JoinTranasction) Gas() uint64         { return 0 }
func (tx *JoinTranasction) To() *common.Address { return nil }
func (tx *JoinTranasction) Hash() common.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := RlpHash(tx)
	tx.hash.Store(v)
	return v
}
func (tx *JoinTranasction) Size() common.StorageSize {
	if size := tx.size.Load(); size != nil {
		return size.(common.StorageSize)
	}
	c := WriteCounter(0)
	rlp.Encode(&c, &tx.data)
	tx.size.Store(common.StorageSize(c))
	return common.StorageSize(c)
}

func (tx *JoinTranasction) RawSignatureValues() (*big.Int, *big.Int, *big.Int) {
	return tx.data.V, tx.data.R, tx.data.S
}

func (tx *JoinTranasction) WithSignature(signer Signer, sig []byte) (Transactioner, error) {
	r, s, v, err := signer.SignatureValues(tx, sig)
	if err != nil {
		return nil, err
	}
	cpy := &JoinTranasction{data: tx.data}
	cpy.data.R, cpy.data.S, cpy.data.V = r, s, v
	return cpy, nil
}
func (tx *JoinTranasction) SigHash(values ...interface{}) common.Hash {
	data := []interface{}{
		tx.data.JoinNonce,
		tx.data.Fee,
		tx.data.Otprn,
	}

	data = append(data, values...)
	return RlpHash(data)
}

func (tx *JoinTranasction) Signature() Signature {
	return Signature{
		V: tx.data.V,
		R: tx.data.R,
		S: tx.data.S,
	}
}

func (tx *JoinTranasction) Sender(signer Signer) (common.Address, error) {
	if sc := tx.from.Load(); sc != nil {
		sigCache := sc.(sigCache)
		// If the signer used to derive from in a previous
		// call is not the same as used current, invalidate
		// the cache.
		if signer.Equal(signer) {
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

func (tx *JoinTranasction) AsMessage(s Signer) (Message, error) {
	msg := Message{
		nonce:      tx.data.JoinNonce,
		gasLimit:   0,
		gasPrice:   nil,
		to:         nil,
		amount:     tx.data.Fee,
		data:       tx.data.Otprn,
		checkNonce: true,
	}

	var err error
	msg.from, err = tx.Sender(s)
	return msg, err
}

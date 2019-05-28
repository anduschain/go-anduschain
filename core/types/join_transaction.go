package types

import (
	"bytes"
	"crypto/ecdsa"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/log"
	"github.com/anduschain/go-anduschain/rlp"
	"io"
	"math/big"
	"time"
)

type JoinTransaction struct{ *Transaction }

func NewJoinTransaction(nonce uint64, to common.Address, amount *big.Int, joinTxData *JoinTxData) *JoinTransaction {
	return newJoinTransaction(nonce, &to, amount, joinTxData)
}

func newJoinTransaction(nonce uint64, to *common.Address, amount *big.Int, joinTxData *JoinTxData) *JoinTransaction {
	gasLimit := uint64(0)
	gasPrice := big.NewInt(0)
	jtx := JoinTransaction{newTransaction(nonce, to, amount, gasLimit, gasPrice, EncodeJoinTxData(joinTxData))}
	return &jtx
}

func (jtx *JoinTransaction) Hash() common.Hash {
	if hash := jtx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := rlpHash(jtx)
	jtx.hash.Store(v)
	return v
}

type JoinTxData struct {
	JoinNonce    uint64
	Otprn        *Otprn
	FairNodeSig  []byte
	TimeStamp    time.Time
	NextBlockNum uint64
}

func NewJoinTxData(nonce uint64, otprn *Otprn, fairsig []byte, nBlockNum uint64) *JoinTxData {
	return &JoinTxData{nonce, otprn, fairsig, time.Now().UTC(), nBlockNum}
}

// EncodeRLP implements rlp.Encoder
func (jtd *JoinTxData) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &jtd)
}

// DecodeRLP implements rlp.Decoder
func (jtd *JoinTxData) DecodeRLP(s *rlp.Stream) error {
	err := s.Decode(&jtd)
	if err != nil {
		return err
	}
	return nil
}

// Transactions is a Transaction slice type for basic sorting.
type JoinTransactions []*JoinTransaction

// Len returns the length of s.
func (jtxs JoinTransactions) Len() int { return len(jtxs) }

// Swap swaps the i'th and the j'th element in s.
func (jtxs JoinTransactions) Swap(i, j int) { jtxs[i], jtxs[j] = jtxs[j], jtxs[i] }

// GetRlp implements Rlpable and returns the i'th element of s in rlp.
func (jtxs JoinTransactions) GetRlp(i int) []byte {
	enc, _ := rlp.EncodeToBytes(jtxs[i])
	return enc
}

type EncodedJoinTxData []byte

func EncodeJoinTxData(jtd *JoinTxData) EncodedJoinTxData {
	var b bytes.Buffer
	err := jtd.EncodeRLP(&b)
	if err != nil {
		log.Error("EncodeJoinTxData", "error", err)
	}
	return b.Bytes()
}

func DecodeJoinTxData(ejtd []byte) *JoinTxData {
	var jtd JoinTxData
	stream := rlp.NewStream(bytes.NewReader(ejtd), 0)
	if err := jtd.DecodeRLP(stream); err != nil {
		log.Error("DecodeJoinTxData", "error", err)
	}

	return &jtd
}

// SignJoinTx signs the join transaction using the given signer and private key
func SignJoinTx(jtx *JoinTransaction, s Signer, prv *ecdsa.PrivateKey) (*JoinTransaction, error) {
	tx := jtx.Transaction
	h := s.Hash(tx)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}

	signedTx, err := tx.WithSignature(s, sig)
	if err != nil {
		return nil, err
	}
	return &JoinTransaction{signedTx}, nil
}

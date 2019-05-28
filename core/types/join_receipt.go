package types

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/rlp"
	"io"
)

type JoinReceipt struct {
	*Receipt
}

// NewReceipt creates a barebone transaction receipt, copying the init fields.
func NewJoinReceipt(root []byte, failed bool, cumulativeGasUsed uint64) *JoinReceipt {
	r := &Receipt{PostState: common.CopyBytes(root), CumulativeGasUsed: cumulativeGasUsed}
	if failed {
		r.Status = ReceiptStatusFailed
	} else {
		r.Status = ReceiptStatusSuccessful
	}
	return &JoinReceipt{r}
}

// Receipts is a wrapper around a Receipt array to implement DerivableList.
type JoinReceipts []*JoinReceipt

// Len returns the number of receipts in this list.
func (r JoinReceipts) Len() int { return len(r) }

// GetRlp returns the RLP encoding of one receipt from the list.
func (r JoinReceipts) GetRlp(i int) []byte {
	bytes, err := rlp.EncodeToBytes(r[i])
	if err != nil {
		panic(err)
	}
	return bytes
}

// ReceiptForStorage is a wrapper around a Receipt that flattens and parses the
// entire content of a receipt, as opposed to only the consensus fields originally.
type JoinReceiptForStorage JoinReceipt

// EncodeRLP implements rlp.Encoder, and flattens all content fields of a receipt
// into an RLP stream.
func (r *JoinReceiptForStorage) EncodeRLP(w io.Writer) error {
	enc := &receiptStorageRLP{
		PostStateOrStatus: (*JoinReceipt)(r).statusEncoding(),
		CumulativeGasUsed: r.CumulativeGasUsed,
		Bloom:             r.Bloom,
		TxHash:            r.TxHash,
		ContractAddress:   r.ContractAddress,
		Logs:              make([]*LogForStorage, len(r.Logs)),
		GasUsed:           r.GasUsed,
	}
	for i, log := range r.Logs {
		enc.Logs[i] = (*LogForStorage)(log)
	}
	return rlp.Encode(w, enc)
}

// DecodeRLP implements rlp.Decoder, and loads both consensus and implementation
// fields of a receipt from an RLP stream.
func (r *JoinReceiptForStorage) DecodeRLP(s *rlp.Stream) error {
	var dec receiptStorageRLP
	if err := s.Decode(&dec); err != nil {
		return err
	}
	if err := (*JoinReceipt)(r).setStatus(dec.PostStateOrStatus); err != nil {
		return err
	}
	// Assign the consensus fields
	r.CumulativeGasUsed, r.Bloom = dec.CumulativeGasUsed, dec.Bloom
	r.Logs = make([]*Log, len(dec.Logs))
	for i, log := range dec.Logs {
		r.Logs[i] = (*Log)(log)
	}
	// Assign the implementation fields
	r.TxHash, r.ContractAddress, r.GasUsed = dec.TxHash, dec.ContractAddress, dec.GasUsed
	return nil
}

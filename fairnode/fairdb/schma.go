package fairdb

import (
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/fairdb/fntype"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"time"
)

func TransBlock(block *types.Block) fntype.Block {
	var tsx []fntype.Transaction
	var voters []fntype.Voter

	for _, tx := range block.Transactions() {
		tsx = append(tsx, fntype.Transaction{
			Hash:      tx.Hash().String(),
			BlockHash: block.Hash().String(),
			Data: fntype.TxData{
				Type:         tx.TransactionId(),
				AccountNonce: tx.Nonce(),
				Price:        tx.GasPrice().String(),
				GasLimit:     tx.Gas(),
				Recipient:    tx.To().String(),
				Amount:       tx.Value().String(),
				Payload:      tx.Data(),
			},
		})
	}

	for _, vote := range block.Voters() {
		voters = append(voters, fntype.Voter{
			Header:   vote.Header,
			Voter:    vote.Voter.String(),
			VoteSign: vote.VoteSign,
		})
	}

	blockNum, _ := bson.ParseDecimal128(block.Number().String())
	return fntype.Block{
		Hash: block.Hash().String(),
		Header: fntype.Header{
			ParentHash:   block.ParentHash().String(),
			Coinbase:     block.Coinbase().String(),
			Root:         block.Root().String(),
			TxHash:       block.TxHash().String(),
			VoteHash:     block.VoterHash().String(),
			ReceiptHash:  block.ReceiptHash().String(),
			Bloom:        block.Bloom().Bytes(),
			Difficulty:   block.Difficulty().String(),
			Number:       blockNum,
			GasLimit:     block.GasLimit(),
			GasUsed:      block.GasUsed(),
			Time:         block.Time().String(),
			Extra:        block.Extra(),
			Nonce:        block.Nonce(),
			Otprn:        block.Otprn(),
			FairnodeSign: block.FairnodeSign(),
		},
		Body: fntype.Body{
			Transactions: tsx,
			Voters:       voters,
		},
	}
}

func TransOtprn(otprn types.Otprn) (fntype.Otprn, error) {

	raw, err := otprn.EncodeOtprn()
	if err != nil {
		return fntype.Otprn{}, err
	}

	return fntype.Otprn{
		Rand:   otprn.RandToByte(),
		FnAddr: otprn.FnAddr.String(),
		Cminer: otprn.Cminer,
		Data: fntype.ChainConfig{
			BlockNumber: otprn.Data.BlockNumber,
			JoinTxPrice: otprn.Data.JoinTxPrice,
			FnFee:       otprn.Data.FnFee,
			Mminer:      otprn.Data.Mminer,
			Epoch:       otprn.Data.Epoch,
			NodeVersion: otprn.Data.NodeVersion,
			Sign:        otprn.Data.Sign,
		},
		Sign:      otprn.Sign,
		Timestamp: time.Now().Unix(),
		Raw:       raw,
	}, nil
}

func RecvOtprn(q *mgo.Query) (*types.Otprn, error) {
	otp := new(fntype.Otprn)
	err := q.One(otp)
	if err != nil {
		return nil, err
	}
	return types.DecodeOtprn(otp.Raw)
}

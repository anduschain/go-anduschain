package fairdb

import (
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/fairdb/fntype"
	"gopkg.in/mgo.v2"
	"math/big"
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
			Number:       block.NumberU64(),
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

func TransTransaction(block *types.Block, chainID *big.Int, txC *mgo.Collection) {
	singer := types.NewEIP155Signer(chainID)
	for _, tx := range block.Transactions() {
		from, err := tx.Sender(singer)
		if err != nil {
			logger.Error("Save Transaction, Sender", "database", "mongo", "msg", err)
			continue
		}

		err = txC.Insert(fntype.STransaction{
			Hash:      tx.Hash().String(),
			BlockHash: block.Hash().String(),
			From:      from.String(),
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

		if err != nil {
			if !mgo.IsDup(err) {
				logger.Error("Save Transaction, Insert", "database", "mongo", "msg", err)
				continue
			}
		}
	}
}

func TransOtprn(otprn types.Otprn) (fntype.Otprn, error) {

	raw, err := otprn.EncodeOtprn()
	if err != nil {
		return fntype.Otprn{}, err
	}

	return fntype.Otprn{
		Hash:   otprn.HashOtprn().String(),
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

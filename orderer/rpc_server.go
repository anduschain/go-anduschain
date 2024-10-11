package orderer

import (
	"context"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/orderer/ordererdb"
	proto "github.com/anduschain/go-anduschain/protos/common"
	"google.golang.org/protobuf/types/known/emptypb"
	"sync"
)

type ordererServer interface {
	Database() ordererdb.OrdererDB
}

// orderer rpc method implemented
type rpcServer struct {
	mu sync.RWMutex
	fn ordererServer
	db ordererdb.OrdererDB
}

func (r rpcServer) HeartBeat(ctx context.Context, beat *proto.HeartBeat) (*emptypb.Empty, error) {
	logger.Info("HeartBeat", "msg", beat)
	return &emptypb.Empty{}, nil
}

func (r rpcServer) Transactions(ctx context.Context, txlist *proto.TransactionList) (*emptypb.Empty, error) {
	_ = txlist.ChainID
	_ = txlist.CurrentHeaderNumber
	_ = txlist.Address
	_ = txlist.Sign
	//err := r.db.InsertTransactionsToTxPool(txlist.Transactions)
	//logger.Info("InsertTransactionsToTxPool", "err", err)
	for idx, aTx := range txlist.Transactions {
		tx := types.Transaction{}
		err := tx.UnmarshalBinary(aTx.Transaction)
		if err != nil {
			logger.Info("Transaction Unmarshal Error", "err", err)
		}
		sender, _ := tx.Sender(types.EIP155Signer{})
		logger.Info("GOT TX", "idx", idx, "hash", tx.Hash(), "from", sender, "nonce", tx.Nonce())
	}
	return &emptypb.Empty{}, nil
}

func newServer(fn ordererServer) *rpcServer {
	return &rpcServer{
		fn: fn,
		db: fn.Database(),
	}
}

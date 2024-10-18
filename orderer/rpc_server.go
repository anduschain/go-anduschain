package orderer

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/log"
	"github.com/anduschain/go-anduschain/orderer/ordererdb"
	proto "github.com/anduschain/go-anduschain/protos/common"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"sync"
)

var (
	emptyByte []byte
)

type ordererServer interface {
	Database() ordererdb.OrdererDB
	GetPrivateKey() *ecdsa.PrivateKey
	GetAddress() common.Address
}

// orderer rpc method implemented
type rpcServer struct {
	mu sync.RWMutex
	os ordererServer
	db ordererdb.OrdererDB
}

func (r rpcServer) ProcessController(participate *proto.Participate, g grpc.ServerStreamingServer[proto.TransactionList]) error {
	//TODO implement me
	panic("implement me")
}

func errorEmpty(key string) error {
	return errors.New(fmt.Sprintf("%s value is empty", key))
}

func (r rpcServer) HeartBeat(ctx context.Context, beat *proto.HeartBeat) (*emptypb.Empty, error) {
	logger.Info("HeartBeat", "msg", beat)
	return &emptypb.Empty{}, nil
}

func (r rpcServer) Transactions(ctx context.Context, txlist *proto.TransactionList) (*emptypb.Empty, error) {
	// ToDo: CSW Check Request
	_ = txlist.ChainID
	_ = txlist.CurrentHeaderNumber
	_ = txlist.Address
	_ = txlist.Sign
	//err := r.db.InsertTransactionsToTxPool(txlist.Transactions)
	//logger.Info("InsertTransactionsToTxPool", "err", err)
	for _, aTx := range txlist.Transactions {
		tx := types.Transaction{}
		err := tx.UnmarshalBinary(aTx.Transaction)
		if err != nil {
			logger.Info("Transaction Unmarshal Error", "err", err)
		}
		sender, _ := tx.Sender(types.EIP155Signer{})

		err = r.db.InsertTransactionToTxPool(sender.Hex(), tx.Nonce(), tx.Hash().Hex(), aTx.Transaction)
		if err != nil {
			logger.Info("Transaction Insert Error", "err", err)
		}
	}
	return &emptypb.Empty{}, nil
}

func (r *rpcServer) RequestOtprn(ctx context.Context, nodeInfo *proto.ReqOtprn) (*proto.ResOtprn, error) {
	if nodeInfo.GetEnode() == "" {
		return nil, errorEmpty("enode")
	}

	if nodeInfo.GetMinerAddress() == "" {
		return nil, errorEmpty("miner's address")
	}

	if bytes.Compare(nodeInfo.GetSign(), emptyByte) == 0 {
		return nil, errorEmpty("sign")
	}
	log.Info("================================ GET REQUEST OTPRN=====================================")
	// ToDo: CSW => Check Node
	otprn := types.NewOtprn(uint64(0), r.os.GetAddress(), types.ChainConfig{}) // 체인 관련 설정값 읽고, OTPRN 생성
	err := otprn.SignOtprn(r.os.GetPrivateKey())                               // OTPRN 서명
	if err != nil {
		return nil, err
	}

	bOtprn, err := otprn.EncodeOtprn()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Otprn EncodeOtprn failed msg=%s", err.Error()))
	}

	return &proto.ResOtprn{
		Otprn:  bOtprn,
		Result: proto.Status_SUCCESS,
	}, nil
}

func newServer(os ordererServer) *rpcServer {
	return &rpcServer{
		os: os,
		db: os.Database(),
	}
}

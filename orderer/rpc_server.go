package orderer

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/verify"
	"github.com/anduschain/go-anduschain/orderer/ordererdb"
	proto "github.com/anduschain/go-anduschain/protos/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"math/big"
	"sync"
	"time"
)

var (
	emptyByte []byte
)

type ordererServer interface {
	SignHash(hash []byte) ([]byte, error)
	Database() ordererdb.OrdererDB
	GetPrivateKey() *ecdsa.PrivateKey
	GetAddress() common.Address
	GetChainID() *big.Int
}

// orderer rpc method implemented
type rpcServer struct {
	mu sync.RWMutex
	os ordererServer
	db ordererdb.OrdererDB
}

func (r rpcServer) ProcessController(participate *proto.Participate, stream grpc.ServerStreamingServer[proto.TransactionList]) error {
	if participate.GetEnode() == "" {
		return errorEmpty("enode")
	}

	if participate.GetMinerAddress() == "" {
		return errorEmpty("miner's address")
	}

	if bytes.Compare(participate.GetOtprnHash(), emptyByte) == 0 {
		return errorEmpty("otprn hash")
	}

	if bytes.Compare(participate.GetSign(), emptyByte) == 0 {
		return errorEmpty("sign")
	}

	hash := rlpHash([]interface{}{
		participate.GetEnode(),
		participate.GetMinerAddress(),
		participate.GetOtprnHash(),
	})

	addr := common.HexToAddress(participate.GetMinerAddress())
	err := verify.ValidationSignHash(participate.GetSign(), hash, addr)
	if err != nil {
		return errors.New(fmt.Sprintf("sign validation failed msg=%s", err.Error()))
	}

	rHash := common.Hash{}
	makeMsg := func(txs []proto.Transaction) *proto.TransactionList {
		var msg proto.TransactionList
		msg.ChainID = r.os.GetChainID().Uint64()
		msg.Address = r.os.GetAddress().String()
		msg.Sign = nil
		msg.Transactions = []*proto.Transaction{}
		for _, tx := range txs {
			aTx := types.Transaction{}
			err := aTx.UnmarshalBinary(tx.Transaction)
			if err == nil {
				hash := aTx.Hash()
				thash, err := XorHashes(rHash.String(), hash.String())
				if err == nil {
					rHash = common.HexToHash(thash)
				}
			}
			tTx := proto.Transaction{}
			tTx = tx
			msg.Transactions = append(msg.Transactions, &tTx)
		}
		msg.Sign, _ = r.os.SignHash(rHash.Bytes())

		return &msg
	}

	for {
		txlist, err := r.db.GetTransactionListFromTxPool()
		if err == nil {
			m := makeMsg(txlist) // make message

			if err := stream.Send(m); err != nil {
				grpcErr, ok := status.FromError(err)
				if ok {
					if grpcErr.Code() == codes.Unavailable {
						return nil
					} else {
						logger.Error("ProcessController gRPC Error", "code", grpcErr.Code(), "msg", grpcErr.Message())
					}
				} else {
					logger.Error("ProcessController send status message", "msg", err)
				}

				return err
			}
		}

		time.Sleep(2 * time.Second)
	}
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

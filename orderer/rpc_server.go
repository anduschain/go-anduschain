package orderer

import (
	"context"
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
	//TODO implement me
	panic("implement me")
}

func newServer(fn ordererServer) *rpcServer {
	return &rpcServer{
		fn: fn,
		db: fn.Database(),
	}
}

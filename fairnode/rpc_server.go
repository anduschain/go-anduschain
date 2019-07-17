package fairnode

import (
	"context"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/fairdb"
	proto "github.com/anduschain/go-anduschain/protos/common"
	"github.com/anduschain/go-anduschain/protos/fairnode"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"math/big"
	"time"
)

func errorEmpty(key string) error {
	return errors.New(fmt.Sprintf("%s value is empty", key))
}

// fairnode rpc method implemented
type rpcServer struct {
	db fairdb.FairnodeDB
}

func newServer(db fairdb.FairnodeDB) *rpcServer {
	return &rpcServer{
		db: db,
	}
}

// Heart Beat : notify to fairnode, I'm alive.
func (rs *rpcServer) HeartBeat(ctx context.Context, nodeInfo *proto.HeartBeat) (*empty.Empty, error) {
	if nodeInfo.GetEnode() == "" {
		return nil, errorEmpty("enode")
	}

	if nodeInfo.GetChainID() == "" {
		return nil, errorEmpty("chainID")
	}

	if nodeInfo.GetMinerAddress() == "" {
		return nil, errorEmpty("miner address")
	}

	if nodeInfo.GetNodeVersion() == "" {
		return nil, errorEmpty("node version")
	}

	if nodeInfo.GetHead() == "" {
		return nil, errorEmpty("head")
	}

	if block := rs.db.CurrentBlock(); block != nil {
		if block.Hash().String() != nodeInfo.GetHead() {
			return nil, errors.New("head is mismatch")
		}
	}

	rs.db.SaveActiveNode(types.HeartBeat{
		Enode:        nodeInfo.GetEnode(),
		NodeVersion:  nodeInfo.GetNodeVersion(),
		ChainID:      nodeInfo.GetChainID(),
		MinerAddress: nodeInfo.GetMinerAddress(),
		Head:         common.HexToHash(nodeInfo.GetHead()),
		Time:         big.NewInt(time.Now().Unix()),
	})

	return &empty.Empty{}, nil
}

func (rs *rpcServer) RequestOtprn(ctx context.Context, nodeInfo *proto.ReqOtprn) (*proto.ResOtprn, error) {
	// otprn 전달
	return nil, nil
}

func (rs *rpcServer) ProcessController(nodeInfo *proto.Participate, stream fairnode.FairnodeService_ProcessControllerServer) error {
	logger.Info("ProcessController", "message", nodeInfo.Enode)
	return nil
}

func (rs *rpcServer) Vote(ctx context.Context, vote *proto.Vote) (*empty.Empty, error) {
	return nil, nil
}

func (rs *rpcServer) RequestVoteResult(ctx context.Context, res *proto.ReqVoteResult) (*proto.ResVoteResult, error) {
	return nil, nil
}

func (rs *rpcServer) SealConfirm(reqSeal *proto.ReqConfirmSeal, stream fairnode.FairnodeService_SealConfirmServer) error {
	return nil
}

func (rs *rpcServer) SendBlock(ctx context.Context, block *proto.ReqBlock) (*empty.Empty, error) {
	return nil, nil
}

func (rs *rpcServer) RequestFairnodeSign(ctx context.Context, reqInfo *proto.ReqFairnodeSign) (*proto.ResFairnodeSign, error) {
	return nil, nil
}

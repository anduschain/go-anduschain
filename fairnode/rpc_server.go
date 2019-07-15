package fairnode

import (
	"context"
	"github.com/anduschain/go-anduschain/protos/common"
	"github.com/anduschain/go-anduschain/protos/fairnode"
	"github.com/golang/protobuf/ptypes/empty"
)

// fairnode rpc method implemented
type rpcServer struct{}

func newServer() *rpcServer {
	logger.Info("==========================")
	return &rpcServer{}
}

func (rs *rpcServer) HeartBeat(ctx context.Context, nodeInfo *common.HeartBeat) (*empty.Empty, error) {
	// heart beat data 받아서 저장
	return nil, nil
}

func (rs *rpcServer) RequestOtprn(ctx context.Context, nodeInfo *common.ReqOtprn) (*common.ResOtprn, error) {
	// otprn 전달
	return nil, nil
}

func (rs *rpcServer) ProcessController(nodeInfo *common.Participate, stream fairnode.FairnodeService_ProcessControllerServer) error {
	return nil
}

func (rs *rpcServer) Vote(ctx context.Context, vote *common.Vote) (*empty.Empty, error) {
	return nil, nil
}

func (rs *rpcServer) RequestVoteResult(ctx context.Context, res *common.ReqVoteResult) (*common.ResVoteResult, error) {
	return nil, nil
}

func (rs *rpcServer) SealConfirm(reqSeal *common.ReqConfirmSeal, stream fairnode.FairnodeService_SealConfirmServer) error {
	return nil
}

func (rs *rpcServer) SendBlock(ctx context.Context, block *common.ReqBlock) (*empty.Empty, error) {
	return nil, nil
}

func (rs *rpcServer) RequestFairnodeSign(ctx context.Context, reqInfo *common.ReqFairnodeSign) (*common.ResFairnodeSign, error) {
	return nil, nil
}

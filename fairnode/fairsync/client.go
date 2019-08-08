package fairsync

import (
	"context"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/log"
	"github.com/anduschain/go-anduschain/protos/fairnode"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

type fnBackClient interface {
	SyncErrChannel() chan error
	SyncMessageChannel() chan []Leagues
}

type rpcSyncClinet struct {
	ctx        context.Context
	rpc        fairnode.FairnodeSyncServiceClient
	syncErrch  chan error
	grpcClinet *grpc.ClientConn
	backend    fnBackClient
}

func newRpcSyncClinet(endPoint string, backend fnBackClient) *rpcSyncClinet {
	grpcConn, err := grpc.Dial(endPoint, grpc.WithInsecure())
	if err != nil {
		logger.Error("did not connect", "msg", err)
		backend.SyncErrChannel() <- err
	}

	return &rpcSyncClinet{
		ctx:        context.Background(),
		grpcClinet: grpcConn,
		rpc:        fairnode.NewFairnodeSyncServiceClient(grpcConn),
		syncErrch:  backend.SyncErrChannel(),
		backend:    backend,
	}
}

func (rc *rpcSyncClinet) Start() {
	go rc.LeaderMessageStream()
}

func (rc *rpcSyncClinet) Stop() {
	rc.grpcClinet.Close()
}

func (rc *rpcSyncClinet) LeaderMessageStream() {
	defer logger.Warn("Leager Message Stream was dead")
	stream, err := rc.rpc.SyncController(rc.ctx, &empty.Empty{})
	if err != nil {
		log.Error("ProcessController", "msg", err)
		rc.syncErrch <- err
		return
	}

	defer stream.CloseSend()

	for {
		in, err := stream.Recv()
		if err != nil {
			logger.Error("Leager Message Stream", "msg", err)
			rc.syncErrch <- err
			return
		}

		var leagues []Leagues
		for _, msg := range in.GetMsg() {
			leagues = append(leagues, Leagues{
				OtprnHash: common.BytesToHash(msg.GetOtprnHash()),
				Status:    types.ProtoToStatus(msg.GetCode()),
			})
		}

		rc.backend.SyncMessageChannel() <- leagues
	}
}

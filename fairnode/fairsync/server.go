package fairsync

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	proto "github.com/anduschain/go-anduschain/protos/common"
	"github.com/anduschain/go-anduschain/protos/fairnode"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"net"
	"time"
)

type fnBackSrv interface {
	Address() string
	SyncErrChannel() chan error
	FairnodeLeagues() map[common.Hash]*Leagues
}

type rpcSyncServer struct {
	tcpListener net.Listener
	gRpcServer  *grpc.Server
	errCh       chan error
	syncErrch   chan error
	backend     fnBackSrv
}

func newRpcSyncServer(backend fnBackSrv) *rpcSyncServer {
	// for fairnode
	lisSync, err := net.Listen("tcp", backend.Address())
	if err != nil {
		logger.Error("Failed to listen", "msg", err)
		backend.SyncErrChannel() <- err
	}

	return &rpcSyncServer{
		tcpListener: lisSync,
		gRpcServer:  grpc.NewServer(),
		errCh:       make(chan error),
		backend:     backend,
		syncErrch:   backend.SyncErrChannel(),
	}
}

func (rs *rpcSyncServer) Start() {
	go rs.syncLoop()
}

func (rs *rpcSyncServer) Stop() {
	rs.gRpcServer.Stop()
	rs.tcpListener.Close()
	defer logger.Warn("Stoped NewRpcSyncServer")
}

func (rs *rpcSyncServer) syncLoop() {
	fairnode.RegisterFairnodeSyncServiceServer(rs.gRpcServer, rs)
	if err := rs.gRpcServer.Serve(rs.tcpListener); err != nil {
		logger.Error("Sync Server Failed to serve", "msg", err)
		rs.errCh <- err
		return
	}

	defer logger.Warn("Sync loop was dead")
}

func (rs *rpcSyncServer) SyncController(empty *empty.Empty, stream fairnode.FairnodeSyncService_SyncControllerServer) error {

	makeMsg := func() *proto.FairnodeMessage {
		msg := new(proto.FairnodeMessage)
		for otprnHash, league := range rs.backend.FairnodeLeagues() {
			msg.Msg = append(msg.Msg, &proto.SyncMessage{
				Code:      types.StatusToProto(league.Status),
				OtprnHash: otprnHash.Bytes(),
			})
		}
		return msg
	}

	for {
		if err := stream.Send(makeMsg()); err != nil {
			logger.Error("Sync Controller send status message", "msg", err)
			return err
		}

		time.Sleep(1 * time.Second)
	}
}

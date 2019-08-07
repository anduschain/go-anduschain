package fairsync

import (
	"fmt"
	"github.com/anduschain/go-anduschain/protos/fairnode"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"net"
)

type rpcSyncServer struct {
	tcpListener net.Listener
	gRpcServer  *grpc.Server
	errCh       chan error
}

func NewRpcSyncServer(port string) (*rpcSyncServer, error) {

	// for fairnode
	lisSync, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		logger.Error("Failed to listen", "msg", err)
		return nil, err
	}

	return &rpcSyncServer{
		tcpListener: lisSync,
		gRpcServer:  grpc.NewServer(),
		errCh:       make(chan error),
	}, nil
}

func (rs *rpcSyncServer) Start() {
	go rs.syncLoop()

	select {
	case err := <-rs.errCh:
		logger.Error("Started NewRpcSyncServer", "msg", err)
		return
	default:
		logger.Info("Started NewRpcSyncServer")
		return
	}
}

func (rs *rpcSyncServer) Stop() {
	rs.gRpcServer.Stop()
	rs.tcpListener.Close()
	defer logger.Warn("Stoped NewRpcSyncServer")
}

func (rs *rpcSyncServer) syncLoop() {
	fairnode.RegisterFairnodeSyncServiceServer(rs.gRpcServer, rs)
	if err := rs.gRpcServer.Serve(rs.tcpListener); err != nil {
		logger.Error("failed to serve: %v", err)
		rs.errCh <- err
	}

	defer logger.Warn("Sync loop was dead")
}

func (rs *rpcSyncServer) SyncController(empty *empty.Empty, stream fairnode.FairnodeSyncService_SyncControllerServer) error {
	return nil
}

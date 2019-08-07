package fairsync

import (
	"github.com/anduschain/go-anduschain/protos/fairnode"
)

type rpcSyncClinet struct {
	rpc fairnode.FairnodeSyncServiceClient
}

func newRpcSyncClinet() *rpcSyncClinet {
	//dc.grpcConn, err = grpc.Dial(dc.fnEndpoint, grpc.WithInsecure())
	//if err != nil {
	//	log.Error("did not connect", "msg", err)
	//	return err
	//}
	//
	//dc.rpc = fairnode.NewFairnodeServiceClient(dc.grpcConn)
	return &rpcSyncClinet{}
}

func (rc *rpcSyncClinet) leaderMessageStream() {

}

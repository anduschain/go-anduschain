package fairnode

import (
	"github.com/anduschain/go-anduschain/protos/fairnode"
	"google.golang.org/grpc"
	"log"
	"net"
)

const (
	port = ":50051"
)

type Fairnode struct {
	tcpListener net.Listener
	gRpcServer  *grpc.Server
}

func NewFairnode() *Fairnode {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	return &Fairnode{
		tcpListener: lis,
		gRpcServer:  grpc.NewServer(),
	}
}

func (fn *Fairnode) Start() {
	fairnode.RegisterFairnodeServiceServer(fn.gRpcServer, newServer())
	if err := fn.gRpcServer.Serve(fn.tcpListener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func (fn *Fairnode) Stop() {
	fn.tcpListener.Close()
	fn.gRpcServer.Stop()
}

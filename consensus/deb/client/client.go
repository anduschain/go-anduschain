package client

import (
	"crypto/ecdsa"
	"github.com/anduschain/go-anduschain/core/types"
	logger "github.com/anduschain/go-anduschain/log"
	"github.com/anduschain/go-anduschain/protos/fairnode"
	"google.golang.org/grpc"
	"sync"
)

var (
	log = logger.New("deb", "deb client")
)

type DebClient struct {
	mu           sync.Mutex
	otprn        *types.Otprn
	minerPrivKey ecdsa.PrivateKey
	rpc          fairnode.FairnodeServiceClient
	fnEndpoint   string
	grpcConn     *grpc.ClientConn
}

func NewDebClient(network types.Network) (*DebClient, error) {
	return &DebClient{
		fnEndpoint: DefaultConfig.FairnodeEndpoint(network),
	}, nil
}

func (dc *DebClient) Start() error {
	var err error
	dc.grpcConn, err = grpc.Dial(dc.fnEndpoint, grpc.WithInsecure())
	if err != nil {
		log.Error("did not connect: %v", err)
		return err
	}

	dc.rpc = fairnode.NewFairnodeServiceClient(dc.grpcConn)
	return nil
}

func (dc *DebClient) Stop() {
	dc.grpcConn.Close()
	log.Debug("grpc connection was closed")
}

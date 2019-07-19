package fairnode

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/fairnode/fairdb"
	"github.com/anduschain/go-anduschain/protos/fairnode"
	"google.golang.org/grpc"
	log "gopkg.in/inconshreveable/log15.v2"
	"net"
	"time"
)

// crypto.HexToECDSA("09bfa4fac90f9daade1722027f6350518c0c2a69728793f8753b2d166ada1a9c") - for test private key
// 0x10Ca4B84feF9Fce8910cb58aCf77255a1A8b61fD - for test addresss
const (
	CLEAN_OLD_NODE_TERM = 3 // per min
)

type fnType uint64

const (
	FN_LEADER fnType = iota
	FN_FOLLOWER
)

var (
	logger = log.New("fairnode", "main")
)

type Fairnode struct {
	tcpListener net.Listener
	gRpcServer  *grpc.Server
	privKey     *ecdsa.PrivateKey
	db          fairdb.FairnodeDB
	errCh       chan error
	role        fnType
}

func NewFairnode() (*Fairnode, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", DefaultConfig.Port))
	if err != nil {
		logger.Error("failed to listen", "msg", err)
		return nil, err
	}

	pKey, err := GetPriveKey(DefaultConfig.KeyPath, DefaultConfig.KeyPass)
	if err != nil {
		return nil, err
	}

	return &Fairnode{
		tcpListener: lis,
		gRpcServer:  grpc.NewServer(),
		errCh:       make(chan error),
		privKey:     pKey,
	}, nil
}

func (fn *Fairnode) Start() error {
	if DefaultConfig.Fake {
		// fake mode memory db
		fn.db = fairdb.NewMemDatabase()
	}

	if err := fn.db.Start(); err != nil {
		logger.Error("fail to db start", "msg", err)
		return err
	}

	fn.db.InitActiveNode() // fairnode init Active node reset

	go fn.severLoop()
	go fn.cleanOldNode()

	if fn.role == FN_LEADER {
		logger.Info("I'm Leader in faionode group")
		go fn.makeOtprn()
	}

	select {
	case err := <-fn.errCh:
		return err
	default:
		logger.Info("Started fairnode")
		return nil
	}
}

func (fn *Fairnode) severLoop() {
	fairnode.RegisterFairnodeServiceServer(fn.gRpcServer, newServer(fn.db))
	if err := fn.gRpcServer.Serve(fn.tcpListener); err != nil {
		logger.Error("failed to serve: %v", err)
		fn.errCh <- err
	}

	defer logger.Warn("server loop was dead")
}

func (fn *Fairnode) Stop() {
	fn.db.Stop()
	fn.gRpcServer.Stop()
	fn.tcpListener.Close()
	defer logger.Warn("Stoped fairnode")
}

func (fn *Fairnode) GetAddress() common.Address {
	return crypto.PubkeyToAddress(fn.privKey.PublicKey)
}

func (fn *Fairnode) GetPublicKey() ecdsa.PublicKey {
	return fn.privKey.PublicKey
}

// 3분에 한번씩 3분간 heartbeat를 보내지 않은 노드 삭제
func (fn *Fairnode) cleanOldNode() {
	defer logger.Warn("clean old node was dead")
	t := time.NewTicker(CLEAN_OLD_NODE_TERM * time.Minute)
	for {
		select {
		case <-t.C:
			now := time.Now().Unix()
			// 3분이상 들어오지 않은 node clean
			nodes := fn.db.GetActiveNode()
			count := 0
			for _, node := range nodes {
				t := node.Time.Int64()
				if (now - t) > (CLEAN_OLD_NODE_TERM * 60) {
					fn.db.RemoveActiveNode(node.Enode)
					count++
				}
			}
			logger.Info("clean old node", "count", count)
		}
	}
}

func (fn *Fairnode) makeOtprn() {
	// 체인 관련 설정값 읽어옴
	//fn.db.Get
	// OTPRN생성후, db에 저장
	//otprn := types.NewOtprn()
}

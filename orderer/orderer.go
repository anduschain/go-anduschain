package orderer

import (
	"fmt"
	"github.com/anduschain/go-anduschain/orderer/ordererdb"
	"github.com/anduschain/go-anduschain/protos/orderer"
	"google.golang.org/grpc"
	log "gopkg.in/inconshreveable/log15.v2"
	"net"
	"sync"
)

var (
	logger log.Logger
)

type Orderer struct {
	mu          sync.Mutex
	tcpListener net.Listener
	gRpcServer  *grpc.Server
	db          ordererdb.OrdererDB
	errCh       chan error
	//roleCh      chan fs.FnType
	//
	//currentLeague *common.Hash
	//pendingLeague *common.Hash

	//fnSyncer    *fs.FnSyncer
	//syncRecvCh  chan []fs.Leagues
	//syncErrorCh chan struct{}

}

func NewOrderer() (*Orderer, error) {
	if DefaultConfig.Debug {
		log.Root().SetHandler(log.StdoutHandler)
	} else {
		handler := log.MultiHandler(
			log.Must.FileHandler("./orderer.log", log.TerminalFormat()), // orderer.log로 저장
		)
		log.Root().SetHandler(handler)
	}

	logger = log.New("orderer", "main")

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", DefaultConfig.Port))
	if err != nil {
		logger.Error("Failed to listen", "msg", err)
		return nil, err
	}

	fn := Orderer{
		tcpListener: lis,
		gRpcServer:  grpc.NewServer(),
		errCh:       make(chan error),
	}

	// orderer syncer
	// ToDo: CSW
	//fn.fnSyncer = fs.NewFnSyncer(&fn, DefaultConfig.SubPort)

	return &fn, nil
}

func (fn *Orderer) Start() error {
	var err error
	fn.db, err = ordererdb.NewMongoDatabase(DefaultConfig)
	if err != nil {
		return err
	}

	if err := fn.db.Start(); err != nil {
		logger.Error("Fail to db start", "msg", err)
		return err
	}

	go fn.severLoop()

	select {
	case err := <-fn.errCh:
		return err
	default:
		logger.Info("Started fairnode")
		return nil
	}
}

func (fn *Orderer) severLoop() {
	orderer.RegisterOrdererServiceServer(fn.gRpcServer, newServer(fn))
	if err := fn.gRpcServer.Serve(fn.tcpListener); err != nil {
		logger.Error("failed to serve: %v", err)
		fn.errCh <- err
	}

	defer logger.Warn("server loop was dead")
}

func (fn *Orderer) Stop() {
	//fn.fnSyncer.Stop()
	fn.db.Stop()
	fn.gRpcServer.Stop()
	fn.tcpListener.Close()
	defer logger.Warn("Stoped orderer")
}

func (fn *Orderer) Database() ordererdb.OrdererDB {
	return fn.db
}
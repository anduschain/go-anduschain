package fairnode

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/fairnode/fairdb"
	"github.com/anduschain/go-anduschain/protos/fairnode"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	log "gopkg.in/inconshreveable/log15.v2"
	"math/big"
	"net"
	"time"
)

// crypto.HexToECDSA("09bfa4fac90f9daade1722027f6350518c0c2a69728793f8753b2d166ada1a9c") - for test private key
// 0x10Ca4B84feF9Fce8910cb58aCf77255a1A8b61fD - for test addresss
const (
	CLEAN_OLD_NODE_TERM    = 3 // per min
	CHECK_ACTIVE_NODE_TERM = 3 // per sec
	MIN_LEAGUE_NUM         = 2
)

type fnType uint64

const (
	FN_LEADER fnType = iota
	FN_FOLLOWER
)

type league struct {
	Otprn     *types.Otprn
	Status    types.FnStatus
	Current   *big.Int // current block number
	BlockHash *common.Hash
	Votehash  *common.Hash
}

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

	currentLeague common.Hash
	leagues       map[common.Hash]*league
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
		tcpListener:   lis,
		gRpcServer:    grpc.NewServer(),
		errCh:         make(chan error),
		privKey:       pKey,
		currentLeague: common.Hash{},
		leagues:       make(map[common.Hash]*league),
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

	if config := fn.db.GetChainConfig(); config == nil {
		return errors.New("chain config is nil, please run addChainConfig")
	}

	fn.db.InitActiveNode() // fairnode init Active node reset

	go fn.severLoop()
	go fn.cleanOldNode()
	go fn.roleCheck()
	go fn.statusLoop()

	select {
	case err := <-fn.errCh:
		return err
	default:
		logger.Info("Started fairnode")
		return nil
	}
}

func (fn *Fairnode) severLoop() {
	fairnode.RegisterFairnodeServiceServer(fn.gRpcServer, newServer(fn))
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

func (fn *Fairnode) SignHash(hash []byte) ([]byte, error) {
	return crypto.Sign(hash, fn.privKey)
}

func (fn *Fairnode) Database() fairdb.FairnodeDB {
	return fn.db
}

func (fn *Fairnode) LeagueSet() map[common.Hash]*league {
	return fn.leagues
}

// role에 따른 작업 구분
func (fn *Fairnode) roleCheck() {
	defer logger.Warn("role check was dead")
	if fn.role == FN_LEADER {
		logger.Info("I'm Leader in fairnode group")
		go fn.makeOtprn()
		go fn.processManageLoop()
	}
}

func (fn *Fairnode) statusLoop() {
	defer logger.Warn("status loop was dead")
	t := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-t.C:
			for id, league := range fn.leagues {
				logger.Debug("status", "otprn", id.String(), "code", league.Status)
			}
		}
	}

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

func (fn *Fairnode) processManageLoop() {
	t := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-t.C:
			if l, ok := fn.leagues[fn.currentLeague]; ok {
				status := l.Status
				switch status {
				case types.PENDING:
					// now league connection count check
					nodes := fn.db.GetLeagueList(fn.currentLeague)
					if len(nodes) >= MIN_LEAGUE_NUM {
						l.Status = types.MAKE_LEAGUE
					}
				case types.MAKE_LEAGUE:
					time.Sleep(5 * time.Second)
					l.Status = types.MAKE_JOIN_TX
				case types.MAKE_JOIN_TX:
					time.Sleep(5 * time.Second)
					// 생성할 블록 번호 조회
					if block := fn.db.CurrentBlock(); block != nil {
						l.Current = block.Number()
					}
					l.Status = types.MAKE_BLOCK
				case types.MAKE_BLOCK:
					time.Sleep(3 * time.Second)
					l.Status = types.LEAGUE_BROADCASTING
				case types.LEAGUE_BROADCASTING:
					time.Sleep(7 * time.Second)
					l.Status = types.VOTE_START
				case types.VOTE_START:
					time.Sleep(5 * time.Second)
					l.Status = types.VOTE_COMPLETE
				case types.REQ_FAIRNODE_SIGN:
					time.Sleep(5 * time.Second)
					l.Status = types.FINALIZE
				case types.FINALIZE:
					time.Sleep(5 * time.Second)
					l.Current = fn.db.CurrentBlock().Number()
					l.Status = types.MAKE_JOIN_TX
				default:
					logger.Debug("process Manage Loop", "staus", status.String())
				}
			}
		}
	}
}

func (fn *Fairnode) makeOtprn() {
	defer logger.Warn("make otprn was dead")
	t := time.NewTicker(CHECK_ACTIVE_NODE_TERM * time.Second)

	newOtprn := func() error {
		if nodes := fn.db.GetActiveNode(); len(nodes) >= MIN_LEAGUE_NUM {
			// 체인 관련 설정값 읽어옴
			config := fn.db.GetChainConfig()
			// OTPRN생성
			otprn := types.NewOtprn(uint64(len(nodes)), fn.GetAddress(), *config)
			// otprn 서명
			err := otprn.SignOtprn(fn.privKey)
			if err != nil {
				return err
			}

			if _, ok := fn.leagues[otprn.HashOtprn()]; !ok {
				// make new league
				fn.leagues[otprn.HashOtprn()] = &league{Otprn: otprn, Status: types.PENDING, Current: big.NewInt(0)} // TODO(hakuna) : currnet block status
				fn.db.SaveOtprn(*otprn)

				fn.currentLeague = otprn.HashOtprn() // TODO(hakuna) : currnet <-> pending ... 처리

				logger.Info("make otprn for league", "otprn", otprn.HashOtprn().String())
				return nil
			} else {
				return errors.New(fmt.Sprintf("league was exist otprn=%s", otprn.HashOtprn().String()))
			}
		} else {
			return errors.New(fmt.Sprintf("not enough active node minimum=%d count=%d", MIN_LEAGUE_NUM, len(nodes)))
		}
	}

	if err := newOtprn(); err != nil {
		logger.Error("new otprn error", "msg", err)
	}

	// league status를 체크해서 주기마다 otprn 생성
	for {
		select {
		case <-t.C:
			if league, ok := fn.leagues[fn.currentLeague]; ok {
				if league.Status == types.MAKE_PENDING_LEAGUE {
					if err := newOtprn(); err != nil {
						logger.Error("new otprn error", "msg", err)
					}
				}
			} else {
				if err := newOtprn(); err != nil {
					logger.Error("new otprn error", "msg", err)
				}
			}
		}
	}

}

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
	"sync"
	"time"
)

// crypto.HexToECDSA("09bfa4fac90f9daade1722027f6350518c0c2a69728793f8753b2d166ada1a9c") - for test private key
// 0x10Ca4B84feF9Fce8910cb58aCf77255a1A8b61fD - for test addresss
const (
	CLEAN_OLD_NODE_TERM    = 3 // per min
	CHECK_ACTIVE_NODE_TERM = 3 // per sec
	MIN_LEAGUE_NUM         = 3
)

type fnType uint64

const (
	FN_LEADER fnType = iota
	FN_FOLLOWER
)

type league struct {
	Mu        sync.Mutex
	Otprn     *types.Otprn
	Status    types.FnStatus
	Current   *big.Int // current block number
	BlockHash *common.Hash
	Votehash  *common.Hash
	Voted     []bool // vote state
}

var (
	logger log.Logger
)

func init() {
	if !DefaultConfig.Debug {
		handler := log.MultiHandler(
			log.Must.FileHandler("./fairnode.log", log.TerminalFormat()), // fairnode.log로 저장
		)
		log.Root().SetHandler(handler)
	} else {
		log.Root().SetHandler(log.StdoutHandler)
	}

	logger = log.New("fairnode", "main")
}

type Fairnode struct {
	mu          sync.Mutex
	tcpListener net.Listener
	gRpcServer  *grpc.Server
	privKey     *ecdsa.PrivateKey
	db          fairdb.FairnodeDB
	errCh       chan error
	role        fnType

	currentLeague       common.Hash
	pendingLeague       common.Hash
	leagues             map[common.Hash]*league
	makePendingLeagueCh chan struct{}
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
		tcpListener:         lis,
		gRpcServer:          grpc.NewServer(),
		errCh:               make(chan error),
		privKey:             pKey,
		currentLeague:       common.Hash{},
		pendingLeague:       common.Hash{},
		leagues:             make(map[common.Hash]*league),
		makePendingLeagueCh: make(chan struct{}),
	}, nil
}

func (fn *Fairnode) Start() error {
	var err error
	if DefaultConfig.Fake {
		fn.db = fairdb.NewMemDatabase() // fake mode memory db
	} else {
		fn.db, err = fairdb.NewMongoDatabase(DefaultConfig) // fake mode memory db
		if err != nil {
			return err
		}
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
				logger.Debug("status", "otprn", reduceStr(id.String()), "code", league.Status)
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
	t := time.NewTicker(500 * time.Millisecond)
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
					// 생성할 블록 번호 조회
					if block := fn.db.CurrentBlock(); block != nil {
						l.Current = block.Number()
					}
					time.Sleep(5 * time.Second)
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
				case types.VOTE_COMPLETE:
					if len(l.Voted) >= MIN_LEAGUE_NUM {
						logger.Error("anyone was not vote, league change and term")
						l.Mu.Lock()
						l.Status = types.REJECT
						l.Mu.Unlock()
					}
				case types.SEND_BLOCK_WAIT:
					time.Sleep(5 * time.Second)
					if l.BlockHash == nil {
						logger.Error("Send block wait, timeout")
						l.Mu.Lock()
						l.Status = types.REJECT
						l.Mu.Unlock()
					}
				case types.REQ_FAIRNODE_SIGN:
					time.Sleep(5 * time.Second)
					l.Status = types.FINALIZE
				case types.FINALIZE:
					l.Current = fn.db.CurrentBlock().Number()
					fn.makePendingLeagueCh <- struct{}{} // signal for checking league otprn
					time.Sleep(5 * time.Second)
					if l.Status == types.REJECT {
						continue
					}
					l.Status = types.MAKE_JOIN_TX
					l.Voted = l.Voted[:0] // reset vote state
					l.Votehash = nil
					l.BlockHash = nil
				case types.REJECT:
					l.Mu.Lock()
					delete(fn.leagues, fn.currentLeague) // league delete
					if pl, ok := fn.leagues[fn.pendingLeague]; ok {
						pl.Current = fn.db.CurrentBlock().Number()
						fn.currentLeague = fn.pendingLeague // change pending to current
					}
					l.Mu.Unlock()
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

	newOtprn := func(isCur bool) error {
		if nodes := fn.db.GetActiveNode(); len(nodes) >= MIN_LEAGUE_NUM {
			config := fn.db.GetChainConfig()                                      // 체인 관련 설정값 읽어옴
			otprn := types.NewOtprn(uint64(len(nodes)), fn.GetAddress(), *config) // OTPRN 생성
			err := otprn.SignOtprn(fn.privKey)                                    // OTPRN 서명
			if err != nil {
				return err
			}
			if _, ok := fn.leagues[otprn.HashOtprn()]; !ok {
				// make new league
				fn.mu.Lock()
				fn.leagues[otprn.HashOtprn()] = &league{Otprn: otprn, Status: types.PENDING, Current: big.NewInt(0)}
				fn.db.SaveOtprn(*otprn)

				if isCur {
					fn.currentLeague = otprn.HashOtprn()
				} else {
					fn.pendingLeague = otprn.HashOtprn()
				}

				fn.mu.Unlock()
				logger.Info("make otprn for league", "otprn", otprn.HashOtprn().String())
				return nil
			} else {
				return errors.New(fmt.Sprintf("league was exist otprn=%s", otprn.HashOtprn().String()))
			}
		} else {
			return errors.New(fmt.Sprintf("not enough active node minimum=%d count=%d", MIN_LEAGUE_NUM, len(nodes)))
		}
	}

	if err := newOtprn(true); err != nil {
		logger.Error("new otprn error", "msg", err)
	}

	// league status를 체크해서 주기마다 otprn 생성
	for {
		select {
		case <-t.C:
			if _, ok := fn.leagues[fn.currentLeague]; !ok {
				if err := newOtprn(true); err != nil {
					logger.Error("new otprn error", "msg", err)
				}
			}
		case <-fn.makePendingLeagueCh:
			// channel for making pending league
			if league, ok := fn.leagues[fn.currentLeague]; ok {
				epoch := new(big.Int).SetUint64(league.Otprn.Data.Epoch) // epoch, league change term
				if epoch.Uint64() == 0 {
					continue
				}

				if league.Current.Uint64() == 0 {
					continue
				}

				// stop league and new league start
				isPending := new(big.Int).Mod(league.Current, big.NewInt(int64(epoch.Int64()/2)))
				isChange := new(big.Int).Mod(league.Current, epoch)
				if isPending.Uint64() == 0 {
					if isChange.Uint64() == 0 {
						logger.Warn("currnet league will be rejected", "epoch", epoch.String(), "current", league.Current.String())
						league.Mu.Lock()
						league.Status = types.REJECT
						league.Mu.Unlock()
					} else {
						// make otprn and pending league
						if err := newOtprn(false); err != nil {
							logger.Error("new otprn error", "msg", err)
						}
						logger.Info("make pending league", "epoch", epoch.String(), "current", league.Current.String())
					}
				}
			}
		}
	}

}

package fairnode

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/fairnode/fairdb"
	fs "github.com/anduschain/go-anduschain/fairnode/fairsync"
	"github.com/anduschain/go-anduschain/fairnode/verify"
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
	MIN_LEAGUE_NUM         = 3 // minimum node count
)

type league struct {
	Mu        sync.Mutex
	Otprn     *types.Otprn
	Status    types.FnStatus
	Current   *big.Int // current block number
	BlockHash *common.Hash
	Votehash  *common.Hash
}

var (
	logger log.Logger
)

type Fairnode struct {
	mu          sync.Mutex
	tcpListener net.Listener
	gRpcServer  *grpc.Server
	privKey     *ecdsa.PrivateKey
	db          fairdb.FairnodeDB
	errCh       chan error
	roleCh      chan fs.FnType

	currentLeague       *common.Hash
	pendingLeague       *common.Hash
	leagues             map[common.Hash]*league
	makePendingLeagueCh chan struct{}

	fnSyncer    *fs.FnSyncer
	syncRecvCh  chan []fs.Leagues
	syncErrorCh chan struct{}

	lastBlock *types.Block

	curRole fs.FnType
}

func NewFairnode() (*Fairnode, error) {
	if DefaultConfig.Debug {
		log.Root().SetHandler(log.StdoutHandler)
	} else {
		handler := log.MultiHandler(
			log.Must.FileHandler("./fairnode.log", log.TerminalFormat()), // fairnode.log로 저장
		)
		log.Root().SetHandler(handler)
	}

	logger = log.New("fairnode", "main")

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", DefaultConfig.Port))
	if err != nil {
		logger.Error("Failed to listen", "msg", err)
		return nil, err
	}

	pKey, err := GetPriveKey(DefaultConfig.KeyPath, DefaultConfig.KeyPass)
	if err != nil {
		return nil, err
	}

	fn := Fairnode{
		tcpListener: lis,
		gRpcServer:  grpc.NewServer(),
		errCh:       make(chan error),
		roleCh:      make(chan fs.FnType),

		privKey:             pKey,
		leagues:             make(map[common.Hash]*league),
		makePendingLeagueCh: make(chan struct{}),
		syncRecvCh:          make(chan []fs.Leagues),
		syncErrorCh:         make(chan struct{}),
	}

	// fairnode syncer
	fn.fnSyncer = fs.NewFnSyncer(&fn, DefaultConfig.SubPort)

	return &fn, nil
}

func (fn *Fairnode) Leagues() map[common.Hash]*fs.Leagues {
	res := make(map[common.Hash]*fs.Leagues)
	for key, league := range fn.leagues {
		res[key] = &fs.Leagues{
			OtprnHash: league.Otprn.HashOtprn(),
			Status:    league.Status,
		}
	}
	return res
}

func (fn *Fairnode) RoleCheckChannel() chan fs.FnType {
	return fn.roleCh
}

func (fn *Fairnode) SyncMessageChannel() chan []fs.Leagues {
	return fn.syncRecvCh
}

func (fn *Fairnode) IsLeader() bool {
	return fn.curRole == fs.FN_LEADER
}

func (fn *Fairnode) Start() error {
	var err error
	if DefaultConfig.Memorydb {
		fn.db = fairdb.NewMemDatabase() // fake mode memory db
	} else {
		fn.db, err = fairdb.NewMongoDatabase(&DefaultConfig) // fake mode memory db
		if err != nil {
			return err
		}
	}

	if err := fn.db.Start(); err != nil {
		logger.Error("Fail to db start", "msg", err)
		return err
	}

	if config := fn.db.GetChainConfig(); config == nil {
		return errors.New("Chain config is nil, please run addChainConfig")
	}

	if db, ok := fn.db.(*fairdb.MongoDatabase); ok {
		fn.fnSyncer.Start(db)
	} else {
		fn.fnSyncer.Start(nil)
	}

	// get last block
	if block := fn.db.CurrentBlock(); block != nil {
		fn.lastBlock = block
		logger.Info("Get Current Block", "hash", fn.lastBlock.Hash().String(), "number", fn.lastBlock.Number().String())
	}

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
	fn.fnSyncer.Stop()
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

// role checking loop
func (fn *Fairnode) roleCheck() {
	defer logger.Warn("Role check was dead")
	for {
		select {
		case role := <-fn.roleCh:
			if fn.curRole == role {
				continue
			}
			fn.curRole = role
			switch fn.curRole {
			case fs.FN_LEADER:
				logger.Info("I'm Leader in fairnode group")
				fn.db.InitActiveNode() // fairnode init Active node reset
				go fn.makeOtprn()
				go fn.processManageLoop()
			case fs.FN_FOLLOWER:
				logger.Info("I'm Follower in fairnode group")
				go fn.processManageLoopFollower()
			case fs.PENDING:
				fn.syncErrorCh <- struct{}{}

			}
		}
	}
}

func (fn *Fairnode) statusLoop() {
	defer logger.Warn("Status loop was dead")
	t := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-t.C:
			if len(fn.leagues) == 0 {
				continue
			}
			for id, league := range fn.leagues {
				logger.Debug("Status", "otprn", reduceStr(id.String()), "code", league.Status, "len", len(fn.leagues), "current", league.Current.String())
			}
		}
	}

}

// 3분에 한번씩 3분간 heartbeat를 보내지 않은 노드 삭제
func (fn *Fairnode) cleanOldNode() {
	defer logger.Warn("Clean old node was dead")
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
			logger.Info("Clean old node", "count", count)
		}
	}
}

// when fairnode Folllower
func (fn *Fairnode) processManageLoopFollower() {
	defer logger.Warn("Process Manage Loop Follower was Dead")
	var dHash common.Hash // delete otprn
	for {
		select {
		case leagues := <-fn.syncRecvCh:
			for _, l := range leagues {
				// 리그 생성
				if _, ok := fn.leagues[l.OtprnHash]; !ok {
					otprn := fn.db.GetOtprn(l.OtprnHash)
					if otprn == nil {
						logger.Error("Process Manage Loop Follower, Get Otprn is nil", "hash", l.OtprnHash)
						continue
					}
					if dHash == l.OtprnHash {
						continue
					}
					fn.addLeague(otprn)
				} else {
					league := fn.leagues[l.OtprnHash]
					if league.Status != l.Status {
						league.Status = l.Status
					}
					switch league.Status {
					case types.MAKE_JOIN_TX:
						if fn.lastBlock != nil {
							league.Current = fn.lastBlock.Number()
						}
					case types.VOTE_COMPLETE:
						voteKey := fairdb.MakeVoteKey(l.OtprnHash, new(big.Int).Add(league.Current, big.NewInt(1)))
						voters := fn.db.GetVoters(voteKey)
						if len(voters) < (MIN_LEAGUE_NUM - 1) {
							continue
						}
						finalBlockHash := verify.ValidationFinalBlockHash(voters) // block hash
						voteHash := types.Voters(voters).Hash()                   // voter hash
						league.BlockHash = &finalBlockHash
						league.Votehash = &voteHash
					case types.FINALIZE:
						if block := fn.db.CurrentBlock(); block != nil {
							fn.lastBlock = block
							league.Current = block.Number()
							league.Votehash = nil
							league.BlockHash = nil
						} else {
							continue
						}
					case types.REJECT:
						if _, ok := fn.leagues[l.OtprnHash]; ok {
							logger.Warn("League Reject", "hash", l.OtprnHash)
							delete(fn.leagues, l.OtprnHash)
							dHash = l.OtprnHash
						}
					}
				}
			}
		case <-fn.syncErrorCh:
			for _, league := range fn.leagues {
				league.Status = types.REJECT
			}
			time.Sleep(500 * time.Millisecond)
			for _, league := range fn.leagues {
				logger.Warn("Leader Node Error, League Reject", "hash", league.Otprn.HashOtprn().String())
				delete(fn.leagues, league.Otprn.HashOtprn())
			}
			logger.Warn("Sync error channel called")
			return
		}
	}
}

// when fairnode leader
func (fn *Fairnode) processManageLoop() {
	t := time.NewTicker(500 * time.Millisecond)
	var status types.FnStatus
	for {
		select {
		case <-t.C:
			if fn.currentLeague == nil {
				continue
			}
			if l, ok := fn.leagues[*fn.currentLeague]; ok {
				if status != l.Status {
					status = l.Status
				}
				switch status {
				case types.PENDING:
					// now league connection count check
					nodes := fn.db.GetLeagueList(*fn.currentLeague)
					if len(nodes) >= MIN_LEAGUE_NUM {
						l.Status = types.MAKE_LEAGUE
					}
				case types.MAKE_LEAGUE:
					time.Sleep(3 * time.Second)
					l.Status = types.MAKE_JOIN_TX
				case types.MAKE_JOIN_TX:
					if fn.lastBlock != nil {
						l.Current = fn.lastBlock.Number()
					}
					time.Sleep(3 * time.Second)
					l.Status = types.MAKE_BLOCK
				case types.MAKE_BLOCK:
					time.Sleep(3 * time.Second)
					l.Status = types.LEAGUE_BROADCASTING
				case types.LEAGUE_BROADCASTING:
					time.Sleep(5 * time.Second)
					l.Status = types.VOTE_START
				case types.VOTE_START:
					time.Sleep(3 * time.Second)
					l.Status = types.VOTE_COMPLETE
				case types.VOTE_COMPLETE:
					voteKey := fairdb.MakeVoteKey(l.Otprn.HashOtprn(), new(big.Int).Add(l.Current, big.NewInt(1)))
					voters := fn.db.GetVoters(voteKey)
					if len(voters) < (MIN_LEAGUE_NUM - 1) {
						logger.Error("Anyone was not vote, league change and term", "VoteCount", len(voters))
						l.Status = types.REJECT
					} else {
						finalBlockHash := verify.ValidationFinalBlockHash(voters) // block hash
						voteHash := types.Voters(voters).Hash()                   // voter hash
						l.BlockHash = &finalBlockHash
						l.Votehash = &voteHash
						time.Sleep(1 * time.Second)
						l.Status = types.SEND_BLOCK
					}
				case types.SEND_BLOCK:
					hash := *l.BlockHash
					if l.BlockHash == nil {
						logger.Error("Send block wait, timeout")
						l.Status = types.REJECT
					} else {
						if block := fn.db.GetBlock(hash); block != nil {
							logger.Info("Send Block Complate", "hash", block.Hash())
							l.Status = types.REQ_FAIRNODE_SIGN
						} else {
							logger.Warn("Wait Send Block", "hash", hash.String())
							time.Sleep(200 * time.Millisecond)
							continue
						}
					}
				case types.REQ_FAIRNODE_SIGN:
					time.Sleep(3 * time.Second)
					l.Status = types.FINALIZE
				case types.FINALIZE:
					if block := fn.db.CurrentBlock(); block != nil {
						fn.lastBlock = block
						l.Current = block.Number()
						l.Votehash = nil
						l.BlockHash = nil
						time.Sleep(3 * time.Second)
						fn.makePendingLeagueCh <- struct{}{} // signal for checking league otprn
					} else {
						hash := *l.BlockHash
						fn.db.RemoveBlock(hash)
						logger.Warn("Remove Block, Because Current Info is nil", "hash", hash)
						l.Status = types.REJECT
					}
				case types.REJECT:
					time.Sleep(2 * time.Second)
					cur := *fn.currentLeague
					logger.Warn("League Reject and Delete League", "hash", cur.String())
					delete(fn.leagues, cur) // league delete
					if fn.pendingLeague != nil {
						if pl, ok := fn.leagues[*fn.pendingLeague]; ok {
							pl.Current = fn.db.CurrentInfo().Number
							fn.currentLeague = fn.pendingLeague // change pending to current
							fn.pendingLeague = nil              // empty pending league key
						}
					} else {
						fn.currentLeague = nil
					}
					if block := fn.db.CurrentBlock(); block != nil {
						fn.lastBlock = block
					}
				default:
					logger.Debug("Process Manage Loop", "Status", status.String())
				}
			}
		}
	}
}

func (fn *Fairnode) addLeague(otprn *types.Otprn) {
	fn.mu.Lock()
	defer fn.mu.Unlock()
	l := league{Otprn: otprn, Status: types.PENDING}
	if fn.lastBlock != nil {
		l.Current = fn.lastBlock.Number()
	} else {
		l.Current = big.NewInt(0)
	}
	fn.leagues[otprn.HashOtprn()] = &l
}

func (fn *Fairnode) makeOtprn() {
	defer logger.Warn("Make OTPRN was dead")
	t := time.NewTicker(CHECK_ACTIVE_NODE_TERM * time.Second)
	newOtprn := func(isCur bool) error {
		if nodes := fn.db.GetActiveNode(); len(nodes) >= MIN_LEAGUE_NUM {
			config := fn.db.GetChainConfig()                                      // 체인 관련 설정값 읽어옴
			otprn := types.NewOtprn(uint64(len(nodes)), fn.GetAddress(), *config) // OTPRN 생성
			err := otprn.SignOtprn(fn.privKey)                                    // OTPRN 서명
			if err != nil {
				return err
			}
			otpHash := otprn.HashOtprn()
			if isCur {
				if fn.currentLeague == nil {
					fn.currentLeague = &otpHash
					fn.addLeague(otprn)
					fn.db.SaveOtprn(*otprn)
				} else {
					return errors.New(fmt.Sprintf("Current League was exist otprn=%s", otpHash.String()))
				}
			} else {
				if fn.pendingLeague == nil {
					fn.pendingLeague = &otpHash
					fn.addLeague(otprn)
					fn.db.SaveOtprn(*otprn)
				} else {
					return errors.New(fmt.Sprintf("Pending League was exist otprn=%s", otpHash.String()))
				}
			}
			logger.Info("Made League otprn", "hash", otpHash.String())
			return nil
		} else {
			return errors.New(fmt.Sprintf("Not enough active node minimum=%d count=%d", MIN_LEAGUE_NUM, len(nodes)))
		}
	}

	if err := newOtprn(true); err != nil {
		logger.Error("Make otprn error", "msg", err)
	}

	// league status를 체크해서 주기마다 otprn 생성
	for {
		select {
		case <-t.C:
			if fn.currentLeague != nil {
				continue
			}
			if err := newOtprn(true); err != nil {
				logger.Error("Make otprn error", "msg", err)
			}
		case <-fn.makePendingLeagueCh:
			if fn.currentLeague == nil {
				continue
			}
			// channel for making pending league
			if league, ok := fn.leagues[*fn.currentLeague]; ok {
				epoch := new(big.Int).SetUint64(league.Otprn.Data.Epoch) // epoch, league change term
				if epoch.Uint64() == 0 {
					league.Status = types.MAKE_JOIN_TX
					continue
				}

				if league.Current.Uint64() == 0 {
					league.Status = types.MAKE_JOIN_TX
					continue
				}

				// stop league and new league start
				isPending := new(big.Int).Mod(league.Current, big.NewInt(int64(epoch.Int64()/2)))
				isChange := new(big.Int).Mod(league.Current, epoch)
				if isPending.Uint64() == 0 {
					if isChange.Uint64() == 0 {
						logger.Warn("Currnet league will be rejected", "epoch", epoch.String(), "current", league.Current.String())
						league.Status = types.REJECT
					} else {
						league.Status = types.MAKE_JOIN_TX
						// make otprn and pending league
						if err := newOtprn(false); err != nil {
							logger.Error("Make otprn error", "msg", err)
							continue
						}
						logger.Info("Make pending league", "epoch", epoch.String(), "current", league.Current.String())
					}
				} else {
					league.Status = types.MAKE_JOIN_TX
				}
			}
		}
	}

}

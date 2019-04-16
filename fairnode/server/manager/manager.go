package manager

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/fairutil/queue"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/fairnode/server/backend"
	"github.com/anduschain/go-anduschain/fairnode/server/db"
	"github.com/anduschain/go-anduschain/fairnode/server/fairtcp"
	"github.com/anduschain/go-anduschain/fairnode/server/fairudp"
	"github.com/anduschain/go-anduschain/fairnode/server/manager/pool"
	"github.com/anduschain/go-anduschain/fairnode/transport"
	log "gopkg.in/inconshreveable/log15.v2"
	"math/big"
	"sync"
)

type ServiceFunc interface {
	Start() error
	Stop() error
}

type FairManager struct {
	Services      map[string]ServiceFunc
	srvKey        *backend.SeverKey
	leaguePool    *pool.LeaguePool
	votePool      *pool.VotePool
	LastBlockNum  *big.Int
	db            *db.FairNodeDB
	Signer        types.Signer
	exit          chan struct{}
	Epoch         *big.Int
	ManageOtprnCh chan struct{}
	StopLeagueCh  chan struct{}
	mux           sync.Mutex

	UsingOtprn *otprn.Otprn // 사용중인 otprn
	OtprnQueue *queue.Queue // fairnode에서 받은 otprn 저장 queue

	reSendOtprn chan common.Hash
	makeJoinTx  chan struct{}
	logger      log.Logger
}

func New() (*FairManager, error) {
	if !backend.DefaultConfig.Debug {
		handler := log.MultiHandler(
			log.Must.FileHandler("./fairnode.json", log.JsonFormat()),
		)
		log.Root().SetHandler(handler)
	}

	fm := &FairManager{
		Epoch:         big.NewInt(backend.DefaultConfig.Epoch),
		Services:      make(map[string]ServiceFunc),
		Signer:        types.NewEIP155Signer(big.NewInt(backend.DefaultConfig.ChainID)),
		exit:          make(chan struct{}),
		ManageOtprnCh: make(chan struct{}),
		StopLeagueCh:  make(chan struct{}),
		UsingOtprn:    nil,
		OtprnQueue:    queue.NewQueue(1),
		reSendOtprn:   make(chan common.Hash),
		makeJoinTx:    make(chan struct{}),
		logger:        log.New("fairnode", "manager"),
	}

	mongoDB, err := db.New(backend.DefaultConfig.DBhost, backend.DefaultConfig.DBport, backend.DefaultConfig.DBpass, backend.DefaultConfig.DBuser, fm.Signer)
	if err != nil {
		return nil, err
	}

	fm.leaguePool = pool.New(mongoDB)
	fm.votePool = pool.NewVotePool(mongoDB)
	fm.db = mongoDB

	ft, err := fairtcp.New(mongoDB, fm)
	if err != nil {
		return nil, err
	}

	fu, err := fairudp.New(mongoDB, fm, ft)
	if err != nil {
		return nil, err
	}

	fm.Services["mongoDB"] = mongoDB
	fm.Services["LeaguePool"] = fm.leaguePool
	fm.Services["VotePool"] = fm.votePool
	fm.Services["fairudp"] = fu
	fm.Services["fairtcp"] = ft

	return fm, nil
}

func (fm *FairManager) Start(srvKey *backend.SeverKey) error {
	fm.mux.Lock()
	defer fm.mux.Unlock()
	fm.srvKey = srvKey

	for name, sev := range fm.Services {
		fm.logger.Info("서비스 시작됨", "service", name)
		if err := sev.Start(); err != nil {
			return err
		}
	}

	go fm.RequestWinningBlock(fm.exit)

	return nil
}

func (fm *FairManager) Stop() error {
	fm.mux.Lock()
	defer fm.mux.Unlock()
	for name, sev := range fm.Services {
		fm.logger.Info("서비스 종료됨", "service", name)
		if err := sev.Stop(); err != nil {
			return err
		}
	}

	fm.exit <- struct{}{}

	return nil
}

func (fm *FairManager) SetService(name string, srv ServiceFunc) {

	for n := range fm.Services {
		if n == name {
			return
		}
	}

	fm.Services[name] = srv
}

func (fm *FairManager) StoreOtprn(otprn *otprn.Otprn) {
	fm.mux.Lock()
	defer fm.mux.Unlock()
	fm.OtprnQueue.Push(otprn)
}

// 순차적으로 만든 otprn return
func (fm *FairManager) GetStoredOtprn() *otprn.Otprn {
	fm.mux.Lock()
	defer fm.mux.Unlock()

	item := fm.OtprnQueue.Pop()
	if item != nil {
		otprn := item.(*otprn.Otprn)
		fm.UsingOtprn = otprn
		return otprn
	}

	return nil
}
func (fm *FairManager) DeleteStoreOtprn() {
	fm.mux.Lock()
	defer fm.mux.Unlock()
	fm.OtprnQueue.Pop()
	fm.UsingOtprn = nil
}
func (fm *FairManager) GetReSendOtprn() chan common.Hash {
	return fm.reSendOtprn
}
func (fm *FairManager) GetUsingOtprn() *otprn.Otprn     { return fm.UsingOtprn }
func (fm *FairManager) GetStopLeagueCh() chan struct{}  { return fm.StopLeagueCh }
func (fm *FairManager) GetEpoch() *big.Int              { return fm.Epoch }
func (fm *FairManager) SetEpoch(epoch int64)            { fm.Epoch = big.NewInt(epoch) }
func (fm *FairManager) GetServerKey() *backend.SeverKey { return fm.srvKey }
func (fm *FairManager) GetLeaguePool() *pool.LeaguePool { return fm.leaguePool }
func (fm *FairManager) GetVotePool() *pool.VotePool     { return fm.votePool }
func (fm *FairManager) GetLastBlockNum() *big.Int {
	fm.mux.Lock()
	defer fm.mux.Unlock()
	fm.LastBlockNum = fm.db.GetCurrentBlock()
	return fm.LastBlockNum
}
func (fm *FairManager) GetSinger() types.Signer          { return fm.Signer }
func (fm *FairManager) GetManagerOtprnCh() chan struct{} { return fm.ManageOtprnCh }
func (fm *FairManager) RequestWinningBlock(exit chan struct{}) {
	for {
		select {
		case req := <-fm.votePool.RequestBlockCh:
			otprnHash := common.Hash(req.OtprnHash)
			if node := fm.leaguePool.GetNode(otprnHash, req.Addr); node != nil {
				msg, err := transport.MakeTsMsg(transport.RequestWinningBlock, req.BlockHash)
				if err != nil {
					fm.logger.Error("RequestWinningBlock", "error", err)
					continue
				}

				err = node.Conn.WriteMsg(msg)
				if err != nil {
					fm.logger.Error("RequestWinningBlock SendMessage", "error", err)
					continue
				}
			}
		case <-exit:
			return
		}
	}
}

func (fm *FairManager) GetMakeJoinTxCh() chan struct{} { return fm.makeJoinTx }

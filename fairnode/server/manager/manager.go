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
	"log"
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

	LeagueOtprnHash common.Hash
	Otprn           map[common.Hash]*otprn.Otprn

	UsingOtprn *otprn.Otprn // 사용중인 otprn
	OtprnQueue *queue.Queue // fairnode에서 받은 otprn 저장 queue

}

func New() (*FairManager, error) {
	fm := &FairManager{
		Epoch:         big.NewInt(backend.DefaultConfig.Epoch),
		Otprn:         make(map[common.Hash]*otprn.Otprn),
		Services:      make(map[string]ServiceFunc),
		Signer:        types.NewEIP155Signer(big.NewInt(backend.DefaultConfig.ChainID)),
		exit:          make(chan struct{}),
		ManageOtprnCh: make(chan struct{}),
		StopLeagueCh:  make(chan struct{}),
		UsingOtprn:    nil,
		OtprnQueue:    queue.NewQueue(1),
	}

	mongoDB, err := db.New(backend.DefaultConfig.DBhost, backend.DefaultConfig.DBport, backend.DefaultConfig.DBpass, backend.DefaultConfig.DBuser, fm.Signer)
	if err != nil {
		return nil, err
	}

	fm.leaguePool = pool.New(mongoDB)
	fm.votePool = pool.NewVotePool(mongoDB)

	ft, err := fairtcp.New(mongoDB, fm)
	if err != nil {
		return nil, err
	}

	fu, err := fairudp.New(mongoDB, fm, ft)
	if err != nil {
		return nil, err
	}

	fm.db = mongoDB

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
		log.Printf("Info[andus] : %s 서비스 시작됨", name)

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
		log.Printf("Info[andus] : %s 서비스 종료됨", name)

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

	otprn := fm.OtprnQueue.Pop().(*otprn.Otprn)
	if otprn != nil {
		fm.UsingOtprn = otprn
		return otprn
	}

	return nil
}
func (fm *FairManager) GetUsingOtprn() *otprn.Otprn     { return fm.UsingOtprn }
func (fm *FairManager) GetStopLeagueCh() chan struct{}  { return fm.StopLeagueCh }
func (fm *FairManager) GetEpoch() *big.Int              { return fm.Epoch }
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
					log.Println("Info[andus] : RequestWinningBlock", err)
					continue
				}

				err = node.Conn.WriteMsg(msg)
				if err != nil {
					log.Println("Info[andus] : RequestWinningBlock SendMessage", err)
					continue
				}
			}
		case <-exit:
			return
		}
	}
}

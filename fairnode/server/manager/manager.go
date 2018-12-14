package manager

import (
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/fairnode/server/backend"
	"github.com/anduschain/go-anduschain/fairnode/server/db"
	"github.com/anduschain/go-anduschain/fairnode/server/fairtcp"
	"github.com/anduschain/go-anduschain/fairnode/server/fairudp"
	"github.com/anduschain/go-anduschain/fairnode/server/manager/pool"
	"log"
)

type ServiceFunc interface {
	Start() error
	Stop() error
}

type FairManager struct {
	LeagueRunningOK bool
	Otprn           otprn.Otprn
	Services        map[string]ServiceFunc
	srvKey          *backend.SeverKey
	leaguePool      *pool.LeaguePool
	votePool        *pool.VotePool
	LastBlockNum    uint64
}

func New() (*FairManager, error) {
	fm := &FairManager{
		Services: make(map[string]ServiceFunc),
	}

	mongoDB, err := db.New(backend.DefaultConfig.DBhost, backend.DefaultConfig.DBport, backend.DefaultConfig.DBpass, backend.DefaultConfig.DBuser)
	if err != nil {
		return nil, err
	}

	fm.leaguePool = pool.New(mongoDB)
	fm.votePool = pool.NewVotePool(mongoDB)

	fu, err := fairudp.New(mongoDB, fm)
	if err != nil {
		return nil, err
	}

	ft, err := fairtcp.New(mongoDB, fm)
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
	fm.srvKey = srvKey

	for name, sev := range fm.Services {
		log.Printf("Info[andus] : %s 서비스 시작됨", name)

		if err := sev.Start(); err != nil {
			return err
		}
	}
	return nil
}

func (fm *FairManager) Stop() error {
	for name, sev := range fm.Services {
		log.Printf("Info[andus] : %s 서비스 종료됨", name)

		if err := sev.Stop(); err != nil {
			return err
		}
	}
	return nil
}

func (fm *FairManager) SetService(name string, srv ServiceFunc) {

	for n, _ := range fm.Services {
		if n == name {
			return
		}
	}

	fm.Services[name] = srv
}

func (fm *FairManager) SetOtprn(otp otprn.Otprn)        { fm.Otprn = otp }
func (fm *FairManager) GetOtprn() otprn.Otprn           { return fm.Otprn }
func (fm *FairManager) GetLeagueRunning() bool          { return fm.LeagueRunningOK }
func (fm *FairManager) SetLeagueRunning(status bool)    { fm.LeagueRunningOK = status }
func (fm *FairManager) GetServerKey() *backend.SeverKey { return fm.srvKey }
func (fm *FairManager) GetLeaguePool() *pool.LeaguePool { return fm.leaguePool }
func (fm *FairManager) GetVotePool() *pool.VotePool     { return fm.votePool }
func (fm *FairManager) SetLastBlockNum(num uint64)      { fm.LastBlockNum = num }
func (fm *FairManager) GetLastBlockNum() uint64         { return fm.LastBlockNum }

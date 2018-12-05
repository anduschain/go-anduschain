package manager

import (
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/fairnode/server/backend"
	"github.com/anduschain/go-anduschain/fairnode/server/db"
	"github.com/anduschain/go-anduschain/fairnode/server/fairudp"
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
	leaguePool      *LeaguePool
}

func New() (*FairManager, error) {
	fm := &FairManager{
		Services: make(map[string]ServiceFunc),
	}

	mongoDB, err := db.New(backend.DefaultConfig.DBhost, backend.DefaultConfig.DBport, backend.DefaultConfig.DBpass, backend.DefaultConfig.DBuser)
	if err != nil {
		return nil, err
	}

	fu, err := fairudp.New(mongoDB, fm)
	if err != nil {
		return nil, err
	}

	fm.leaguePool = NewLeaguePool(mongoDB)

	fm.Services["mongoDB"] = mongoDB
	fm.Services["fairudp"] = fu
	fm.Services["LeaguePool"] = fm.leaguePool

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

func (fm *FairManager) SetOtprn(otp otprn.Otprn) {
	fm.Otprn = otp
}

func (fm *FairManager) GetLeagueRunning() bool {
	return fm.LeagueRunningOK
}

func (fm *FairManager) SetLeagueRunning(status bool) {
	fm.LeagueRunningOK = status
}

func (fm *FairManager) GetServerKey() *backend.SeverKey {
	return fm.srvKey
}

func (fm *FairManager) GetLeaguePool() *LeaguePool {
	return fm.leaguePool
}

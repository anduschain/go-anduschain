package fairtcp

import (
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/fairnode/server/backend"
	"github.com/anduschain/go-anduschain/fairnode/server/config"
	"github.com/anduschain/go-anduschain/fairnode/server/db"
	"github.com/anduschain/go-anduschain/fairnode/server/manager/pool"
	"github.com/anduschain/go-anduschain/fairnode/transport"
	"github.com/anduschain/go-anduschain/p2p/nat"
	log "gopkg.in/inconshreveable/log15.v2"
	"net"
	"time"
)

var (
	errNat          = errors.New("NAT 설정에 문제가 있습니다")
	errTcpListen    = errors.New("TCP Lister 설정에 문제가 있습니다")
	closeConnection = errors.New("close")
)

const (
	chkTCPConn = 10 // TCP로 연결되어있는 노드가 하나도 없을때 확인하는 횟수
)

type FairTcp struct {
	LAddrTCP     *net.TCPAddr
	natm         nat.Interface
	listener     *net.TCPListener
	Db           *db.FairNodeDB
	manager      backend.Manager
	services     map[string]backend.Goroutine
	sendLeagueCh chan struct{}
	leaguePool   *pool.LeaguePool
	logger       log.Logger
}

func New(db *db.FairNodeDB, fm backend.Manager) (*FairTcp, error) {
	addr := fmt.Sprintf(":%s", config.DefaultConfig.Port)

	LAddrTCP, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}

	natm, err := nat.Parse(config.DefaultConfig.NAT)
	if err != nil {
		return nil, errNat
	}

	ft := &FairTcp{
		LAddrTCP:     LAddrTCP,
		natm:         natm,
		Db:           db,
		manager:      fm,
		services:     make(map[string]backend.Goroutine),
		sendLeagueCh: make(chan struct{}),
		leaguePool:   fm.GetLeaguePool(),
		logger:       log.New("fairnode", "tcp"),
	}

	ft.services["accepter"] = backend.Goroutine{ft.accepter, make(chan struct{}, 1)}

	return ft, nil
}

func (ft *FairTcp) Start() error {

	var err error

	ft.listener, err = net.ListenTCP("tcp", ft.LAddrTCP)
	if err != nil {
		return errTcpListen
	}

	if ft.natm != nil {
		laddr := ft.listener.Addr().(*net.TCPAddr)
		// Map the TCP listening port if NAT is configured.
		if !laddr.IP.IsLoopback() {
			go func() {
				nat.Map(ft.natm, nil, "tcp", laddr.Port, laddr.Port, "andus fairnode discovery")
			}()
		}
	}

	for name, srv := range ft.services {
		ft.logger.Info("TCP 서비스 실행됨", "service", name)
		go srv.Fn(srv.Exit)
	}

	return nil

}

func (ft *FairTcp) Stop() error {
	err := ft.listener.Close()
	if err != nil {
		return err
	}

	time.Sleep(1 * time.Second)

	for _, srv := range ft.services {
		srv.Exit <- struct{}{}
	}

	return nil
}

func (ft *FairTcp) accepter(exit chan struct{}) {
	defer ft.logger.Debug("tcp accepter kill")

	notify := make(chan error)
	accept := make(chan net.Conn)

	go func() {
		for {
			conn, err := ft.listener.Accept()
			if err != nil {
				notify <- err
				return
			}
			accept <- conn
		}
	}()

Exit:
	for {
		select {
		case <-exit:
			break Exit
		case err := <-notify:
			ft.logger.Error("accepter", "error", err)
		case conn := <-accept:
			ft.logger.Info("tcp 접속 함")
			go func() {
				rwtsp := transport.New(conn)
				otprn := ft.manager.GetUsingOtprn()
				if otprn == nil {
					return
				}

				for {
					if err := ft.handelMsg(rwtsp, otprn.HashOtprn()); err != nil {
						ft.logger.Error("handelMsg", "error", err)
						rwtsp.Close()
						return
					}
				}
			}()
		}
	}
}

func (ft *FairTcp) StartLeague(leagueChange bool) {
	if ft.manager.GetLastBlockNum().Uint64() == 0 || leagueChange {
		otprn := ft.manager.GetStoredOtprn()
		if otprn == nil {
			ft.manager.GetReSendOtprn() <- common.Hash{}
			return
		}

		go func() {
			t := time.NewTicker(1 * time.Second)
			ticker := 0
			for {
				select {
				case <-t.C:
					leaguepool := ft.manager.GetLeaguePool()
					if _, num, _ := leaguepool.GetLeagueList(pool.OtprnHash(otprn.HashOtprn())); num == 0 {
						ticker++
					} else {
						go ft.sendLeague(otprn.HashOtprn())
						go ft.leagueControlle(otprn.HashOtprn())
						return
					}

					if ticker >= chkTCPConn {
						ft.logger.Debug("OTPRN 재발행")
						ft.manager.GetReSendOtprn() <- otprn.HashOtprn()
						return
					}
				}
			}
		}()
	}
}

func (ft *FairTcp) StopLeague(otprnHash common.Hash) {
	ft.logger.Debug("League Stop / 재시작")
	ft.sendTcpAll(otprnHash, transport.FinishLeague, otprnHash)
	ft.StartLeague(true)
}

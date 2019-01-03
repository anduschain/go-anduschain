package fairtcp

import (
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/fairnode/server/backend"
	"github.com/anduschain/go-anduschain/fairnode/server/db"
	"github.com/anduschain/go-anduschain/fairnode/server/manager/pool"
	"github.com/anduschain/go-anduschain/fairnode/transport"
	"github.com/anduschain/go-anduschain/p2p/nat"
	"log"
	"net"
	"time"
)

var (
	errNat          = errors.New("NAT 설정에 문제가 있습니다")
	errTcpListen    = errors.New("TCP Lister 설정에 문제가 있습니다")
	closeConnection = errors.New("close")
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
}

func New(db *db.FairNodeDB, fm backend.Manager) (*FairTcp, error) {
	addr := fmt.Sprintf(":%s", backend.DefaultConfig.Port)

	LAddrTCP, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}

	natm, err := nat.Parse(backend.DefaultConfig.NAT)
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
		log.Printf("Info[andus] : TCP 서비스 %s 실행됨", name)
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
	defer log.Println("Info[andus] : tcp accepter kill")

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
		case <-time.After(time.Second * 1):
			//fmt.Println("TCP timeout, still alive")
		case err := <-notify:
			log.Println("Error[andus] : ", err)
		case conn := <-accept:
			log.Println("Info[andus] : tcp 접속 함")
			go func() {
				rwtsp := transport.New(conn)
				for {
					if err := ft.handelMsg(rwtsp); err != nil {
						log.Println("Error [andus] : handelMsg 에러", err)
						rwtsp.Close()
						return
					}
				}
			}()
		}
	}
}

func (ft *FairTcp) StartLeague() {
	go ft.sendLeague(ft.manager.GetOtprn().HashOtprn().String())
}

func (ft *FairTcp) StopLeague() {
	ft.sendTcpAll(transport.FinishLeague, "리그가 종료 되었습니다")
	ft.manager.SetLeagueRunning(false)
}

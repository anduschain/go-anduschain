package fairtcp

import (
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/fairnode/server/backend"
	"github.com/anduschain/go-anduschain/p2p/nat"
	"net"
)

var (
	errNat       = errors.New("NAT 설정에 문제가 있습니다")
	errTcpListen = errors.New("TCP Lister 설정에 문제가 있습니다")
)

type FairTcp struct {
	LAddrTCP *net.TCPAddr
	natm     nat.Interface
	listener *net.TCPListener
}

func New() (*FairTcp, error) {
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
		LAddrTCP: LAddrTCP,
		natm:     natm,
	}

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

	return nil

}

func (ft *FairTcp) Stop() error {
	err := ft.listener.Close()
	if err != nil {
		return err
	}
	return nil
}

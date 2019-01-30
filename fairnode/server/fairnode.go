package server

import (
	"github.com/anduschain/go-anduschain/fairnode/server/backend"
	"github.com/anduschain/go-anduschain/fairnode/server/manager"
	"github.com/anduschain/go-anduschain/log"
	"net"
)

// TODO : andus >> timezone 셋팅

var (
//FairnodeKeyError     = errors.New("패어노드키가 언락 되지 않았습니다")
//KeyFileNotExistError = errors.New("페어노드의 키가 생성되지 않았습니다. 생성해 주세요.")
//NATinitError         = errors.New("NAT 설정에 문제가 있습니다")
//KeyPassError         = errors.New("패어노드 키가 입력되지 않았습니다.")
)

type connPool map[string]*net.TCPConn
type connNode map[string][]string

type FairNode struct {
	StopCh  chan struct{} // TODO : andus >> 죽을때 처리..
	Manager *manager.FairManager
}

func New() (*FairNode, error) {

	var err error

	fnNode := &FairNode{
		StopCh: make(chan struct{}),
	}

	fnNode.Manager, err = manager.New()
	if err != nil {
		return nil, err
	}

	return fnNode, nil
}

func (f *FairNode) Start() error {
	srvKey, err := backend.GetServerKey()
	if err != nil {
		return err
	}

	log.Info("Administering Ethereum network", "name", "================")

	if err := f.Manager.Start(srvKey); err != nil {
		return err
	}

	return nil
}

func (f *FairNode) Stop() {
	f.Manager.Stop()
}

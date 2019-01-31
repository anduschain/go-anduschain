package server

import (
	"github.com/anduschain/go-anduschain/fairnode/server/backend"
	"github.com/anduschain/go-anduschain/fairnode/server/manager"
)

// TODO : andus >> timezone 셋팅

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

	if err := f.Manager.Start(srvKey); err != nil {
		return err
	}

	return nil
}

func (f *FairNode) Stop() {
	f.Manager.Stop()
}

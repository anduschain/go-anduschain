package pool

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/fairnode/server/db"
	"github.com/anduschain/go-anduschain/fairnode/transport"
	"log"
	"sync"
)

type Node struct {
	Enode    string
	Coinbase common.Address
	Conn     transport.Transport
}

type OtprnHash common.Hash

type Nodes []Node

type PoolIn struct {
	Hash OtprnHash
	Node Node
}

type LeaguePool struct {
	pool     map[OtprnHash]Nodes
	InsertCh chan PoolIn
	UpdateCh chan PoolIn
	DeleteCh chan OtprnHash
	SnapShot chan OtprnHash
	StopCh   chan struct{}
	Db       *db.FairNodeDB
	mux      sync.RWMutex
}

func New(db *db.FairNodeDB) *LeaguePool {
	pool := &LeaguePool{
		pool:     make(map[OtprnHash]Nodes),
		InsertCh: make(chan PoolIn),
		UpdateCh: make(chan PoolIn),
		DeleteCh: make(chan OtprnHash),
		SnapShot: make(chan OtprnHash),
		StopCh:   make(chan struct{}, 1),
		Db:       db,
	}

	return pool
}

func (l *LeaguePool) Start() error {
	go l.loop()
	return nil
}

func (l *LeaguePool) Stop() error {
	l.StopCh <- struct{}{}
	return nil
}

func (l *LeaguePool) GetLeagueList(h OtprnHash) (Nodes, uint64, []string) {
	l.mux.Lock()
	defer l.mux.Unlock()

	var enodes []string
	if val, ok := l.pool[h]; ok {
		for index := range val {
			enodes = append(enodes, val[index].Enode)
		}
		return val, uint64(len(val)), enodes
	}
	return nil, 0, enodes
}

func (l *LeaguePool) GetNode(h common.Hash, addr common.Address) *Node {
	l.mux.Lock()
	defer l.mux.Unlock()

	if val, ok := l.pool[OtprnHash(h)]; ok {
		for i := range val {
			if val[i].Coinbase == addr {
				return &val[i]
			}
		}
	}

	return nil
}

func (l *LeaguePool) loop() {
	defer log.Println("Info[andus] LeaguePool Kill")

Exit:
	for {
		select {
		case n := <-l.InsertCh:
			if val, ok := l.pool[n.Hash]; ok {
				isDouble := false
				for i := range val {
					// 중복 삽입 방지
					if val[i].Enode == n.Node.Enode {
						isDouble = true
						break
					}
				}

				if !isDouble {
					l.mux.Lock()
					l.pool[n.Hash] = append(val, n.Node)
					l.mux.Unlock()
				}

			} else {
				l.mux.Lock()
				l.pool[n.Hash] = Nodes{n.Node}
				l.mux.Unlock()
			}
		case n := <-l.UpdateCh:
			if val, ok := l.pool[n.Hash]; ok {
				for i := range val {
					if val[i].Enode == n.Node.Enode {
						l.pool[n.Hash][i] = n.Node
						break
					}
				}
			}
		case hash := <-l.DeleteCh:
			if _, ok := l.pool[hash]; ok {
				l.mux.Lock()
				delete(l.pool, hash)
				l.mux.Unlock()
			}
		case hash := <-l.SnapShot:
			if val, ok := l.pool[hash]; ok {
				for i := range val {
					l.Db.SaveMinerNode(common.Hash(hash).String(), val[i].Enode)
					//fmt.Println(hash.String(), val[i].Enode)
				}
			}
		case <-l.StopCh:
			break Exit
		}
	}
}

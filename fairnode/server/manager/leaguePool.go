package manager

import (
	"github.com/anduschain/go-anduschain/fairnode/server/db"
	"log"
	"net"
)

type node struct {
	Enode    string
	Coinbase string
	Conn     *net.TCPConn
}

type otprnHash string

func (otp otprnHash) String() string {
	return string(otp)
}

type nodes []node

type PoolIn struct {
	Hash otprnHash
	Node node
}

type LeaguePool struct {
	pool     map[otprnHash]nodes
	InsertCh chan PoolIn
	UpdateCh chan PoolIn
	DeleteCh chan otprnHash
	SnapShot chan otprnHash
	StopCh   chan struct{}
	Db       *db.FairNodeDB
}

func NewLeaguePool(db *db.FairNodeDB) *LeaguePool {
	pool := &LeaguePool{
		pool:     make(map[otprnHash]nodes),
		InsertCh: make(chan PoolIn, 5),
		UpdateCh: make(chan PoolIn, 5),
		DeleteCh: make(chan otprnHash, 5),
		SnapShot: make(chan otprnHash, 5),
		StopCh:   make(chan struct{}),
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

func (l *LeaguePool) loop() {
	defer log.Println("Info[andus] LeaguePool Kill")
Exit:
	for {
		select {
		case n := <-l.InsertCh:
			if val, ok := l.pool[n.Hash]; ok {
				val = append(val, n.Node)
			} else {
				l.pool[n.Hash] = nodes{n.Node}
			}
		case n := <-l.UpdateCh:
			if val, ok := l.pool[n.Hash]; ok {
				for i := range val {
					if val[i].Coinbase == n.Node.Coinbase {
						val[i] = n.Node
						return
					}
				}
			}
		case hash := <-l.DeleteCh:
			if _, ok := l.pool[hash]; ok {
				delete(l.pool, hash)
			}
		case hash := <-l.SnapShot:
			if val, ok := l.pool[hash]; ok {
				for i := range val {
					l.Db.SaveMinerNode(hash.String(), val[i].Enode)
				}
			}
		case <-l.StopCh:
			break Exit
		}
	}
}

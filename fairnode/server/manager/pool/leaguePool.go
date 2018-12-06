package pool

import (
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/fairnode/server/db"
	"log"
	"net"
	"sync"
)

type Node struct {
	Enode    string
	Coinbase common.Address
	Conn     net.Conn
}

type OtprnHash string

func (otp OtprnHash) String() string {
	return string(otp)
}

func StringToOtprn(str string) OtprnHash {
	return OtprnHash(str)
}

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
	mux      sync.Mutex
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

func (l *LeaguePool) loop() {
	defer log.Println("Info[andus] LeaguePool Kill")

	fmt.Println("리그풀 시작됨")
Exit:
	for {
		select {
		case n := <-l.InsertCh:
			l.mux.Lock()
			if val, ok := l.pool[n.Hash]; ok {
				isDouble := false
				for i := range val {
					// 중복 삽입 방지
					if val[i].Enode == n.Node.Enode {
						isDouble = true
						return
					}
				}

				if !isDouble {
					l.pool[n.Hash] = append(val, n.Node)
					log.Println("-------------저장됨1--------------", l.pool[n.Hash])
				}

			} else {
				l.pool[n.Hash] = Nodes{n.Node}
				log.Println("-------------저장됨2--------------", l.pool[n.Hash])
			}
			l.mux.Unlock()
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
			log.Println("-------------삭제됨--------------")
		case hash := <-l.SnapShot:
			if val, ok := l.pool[hash]; ok {
				for i := range val {
					//l.Db.SaveMinerNode(hash.String(), val[i].Enode)
					fmt.Println(hash.String(), val[i].Enode)
				}
			}
		case <-l.StopCh:
			log.Println("-------------종료됨--------------")
			break Exit
		}
	}
}

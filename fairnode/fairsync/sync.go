package fairsync

import (
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/crypto/sha3"
	"github.com/anduschain/go-anduschain/rlp"
	log "gopkg.in/inconshreveable/log15.v2"
	"net"
	"sync/atomic"
	"time"
)

var logger = log.New("fairnode", "database")

type FnType uint64

const (
	PENDING FnType = iota
	FN_LEADER
	FN_FOLLOWER
)

type database interface {
	InsertActiveFairnode(nodeKey common.Hash, address string)
	GetActiveFairnodes() map[common.Hash]map[string]string
	RemoveActiveFairnode(nodeKey common.Hash)
	UpdateActiveFairnode(nodeKey common.Hash, status uint64)
}

type FnSyncer struct {
	id      common.Hash
	curRole FnType // current role
	roleCh  chan FnType

	address string
	running int64

	db     database
	exitCh chan struct{}
}

func NewFnSyncer(rolech chan FnType, port string) *FnSyncer {
	fnId, ip := getFairnodeID()
	if fnId == nil {
		return nil
	}

	return &FnSyncer{
		id:      *fnId,
		curRole: PENDING,
		roleCh:  rolech,
		address: fmt.Sprintf("%s:%s", ip, port),
		exitCh:  make(chan struct{}),
	}
}

func (fs *FnSyncer) Start(db database) {
	if db != nil {
		fs.db = db
		// register fairnode to db
		fs.db.InsertActiveFairnode(fs.id, fs.address)
		fs.curRole = checkRole(fs.db.GetActiveFairnodes())
	} else {
		// for memory database
		fs.curRole = FN_LEADER
	}

	go fs.roleChecker()
	atomic.StoreInt64(&fs.running, 1)
	logger.Info("Start Fairnode Syncer")
}

func (fs *FnSyncer) Stop() {
	if atomic.LoadInt64(&fs.running) == 1 {
		fs.db.RemoveActiveFairnode(fs.id)
		fs.exitCh <- struct{}{}
		atomic.StoreInt64(&fs.running, 0)
	}
	logger.Info("Stop Fairnode Syncer")
}

func (fs *FnSyncer) roleChecker() {
	defer logger.Warn("Fairnode Sync Role Checker was dead")
	t := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-t.C:
			fs.roleCh <- fs.curRole
		case <-fs.exitCh:
			return
		}
	}
}

func getFairnodeID() (*common.Hash, string) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		logger.Error("Get Fairnode ID", "msg", err)
		return nil, ""
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				hash := rlpHash([]interface{}{
					ipnet.IP.String(),
					time.Now().String(),
				})
				return &hash, ipnet.IP.String()
			}
		}
	}

	return nil, ""
}

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

func checkRole(nodes map[common.Hash]map[string]string) FnType {

	return FN_LEADER
}

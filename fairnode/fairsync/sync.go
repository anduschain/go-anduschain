package fairsync

import (
	"bytes"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	log "gopkg.in/inconshreveable/log15.v2"
	"sync/atomic"
	"time"
)

var logger = log.New("fairnode", "syncer")

type FnType uint64

const (
	PENDING FnType = iota
	FN_LEADER
	FN_FOLLOWER
)

func (t FnType) String() string {
	switch t {
	case PENDING:
		return "PENDING"
	case FN_LEADER:
		return "FN_LEADER"
	case FN_FOLLOWER:
		return "FN_FOLLOWER"
	default:
		return "UNKNOWN"
	}
}

type database interface {
	InsertActiveFairnode(nodeKey common.Hash, address string)
	GetActiveFairnodes() map[common.Hash]map[string]string
	RemoveActiveFairnode(nodeKey common.Hash)
	UpdateActiveFairnode(nodeKey common.Hash, status uint64)
}

type Leagues struct {
	OtprnHash common.Hash
	Status    types.FnStatus
}

type fnode interface {
	Leagues() map[common.Hash]*Leagues
	RoleCheckChannel() chan FnType
	SyncMessageChannel() chan []Leagues
}

type fnNode struct {
	ID      common.Hash
	Address string
	Status  FnType
}

type FnSyncer struct {
	id      common.Hash
	curRole FnType // current role
	roleCh  chan FnType

	address string
	running int64

	leager *common.Hash
	nodes  map[common.Hash]fnNode

	db     database
	fnode  fnode
	exitCh chan struct{}

	syncErrCh chan error

	rpcSever  *rpcSyncServer
	rpcClinet *rpcSyncClinet
}

func NewFnSyncer(fn fnode, port string) *FnSyncer {
	fnId, ip := getFairnodeID()
	if fnId == nil {
		return nil
	}

	return &FnSyncer{
		id:        *fnId,
		curRole:   PENDING,
		fnode:     fn,
		roleCh:    fn.RoleCheckChannel(),
		address:   fmt.Sprintf("%s:%s", ip, port),
		exitCh:    make(chan struct{}),
		syncErrCh: make(chan error),
	}
}

func (fs *FnSyncer) Address() string {
	return fs.address
}

func (fs *FnSyncer) SyncErrChannel() chan error {
	return fs.syncErrCh
}

func (fs *FnSyncer) FairnodeLeagues() map[common.Hash]*Leagues {
	return fs.fnode.Leagues()
}

func (fs *FnSyncer) SyncMessageChannel() chan []Leagues {
	return fs.fnode.SyncMessageChannel()
}

func (fs *FnSyncer) Start(db database) {
	if db != nil {
		fs.db = db
		fs.db.InsertActiveFairnode(fs.id, fs.address)
		fs.nodes = parseNode(fs.db.GetActiveFairnodes())
		fs.curRole = fs.checkRole()
		fs.rpcSever = newRpcSyncServer(fs)
		fs.roleChanger()
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
		if fs.db != nil {
			fs.db.RemoveActiveFairnode(fs.id)
		}
		fs.close()
		fs.exitCh <- struct{}{}
		atomic.StoreInt64(&fs.running, 0)
	}
	logger.Info("Stop Fairnode Syncer")
}

func (fs *FnSyncer) close() {
	switch fs.curRole {
	case FN_LEADER:
		fs.rpcSever.Stop()
	case FN_FOLLOWER:
		fs.rpcClinet.Stop()
	}
}

func (fs *FnSyncer) roleChanger() {
	fs.db.UpdateActiveFairnode(fs.id, uint64(fs.curRole))
	switch fs.curRole {
	case FN_LEADER:
		fs.rpcSever.Start()
	case FN_FOLLOWER:
		if fs.leager != nil {
			if node, ok := fs.nodes[*fs.leager]; ok {
				fs.rpcClinet = newRpcSyncClinet(node.Address, fs)
				fs.rpcClinet.Start()
			}
		}
	}
}

func (fs *FnSyncer) roleChecker() {
	defer logger.Warn("Fairnode Sync Role Checker was dead")
	t := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-t.C:
			fs.roleCh <- fs.curRole
		case err := <-fs.syncErrCh:
			switch fs.curRole {
			case FN_LEADER:
			case FN_FOLLOWER:
				fs.close()
				fs.db.RemoveActiveFairnode(*fs.leager)
				fs.curRole = PENDING
				fs.roleCh <- fs.curRole
				fs.db.UpdateActiveFairnode(fs.id, uint64(PENDING))
				fs.nodes = parseNode(fs.db.GetActiveFairnodes())
				fs.curRole = fs.checkRole()
				fs.roleChanger()
			}
			logger.Warn("Role Checker", "msg", err)
		case <-fs.exitCh:
			return
		}
	}
}

func (fs *FnSyncer) checkRole() FnType {
	if len(fs.nodes) == 0 {
		fs.leager = &fs.id
		return FN_LEADER
	} else if len(fs.nodes) == 1 {
		if _, ok := fs.nodes[fs.id]; ok {
			fs.leager = &fs.id
			return FN_LEADER
		}
	} else {
		top := common.Hash{}
		for _, node := range fs.nodes {
			if node.Status == FN_LEADER {
				fs.leager = &node.ID
				return FN_FOLLOWER
			}
			if bytes.Compare(top.Bytes(), common.Hash{}.Bytes()) == 0 {
				top = node.ID
			}
			// top < node
			if top.Big().Cmp(node.ID.Big()) < 0 {
				top = node.ID
			}
		}
		if bytes.Compare(top.Bytes(), fs.id.Bytes()) == 0 {
			fs.leager = &fs.id
			return FN_LEADER
		}
		fs.leager = &top
	}
	return FN_FOLLOWER
}

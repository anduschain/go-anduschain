package fairdb

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/fairdb/fntype"
	"github.com/anduschain/go-anduschain/rlp"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"math/big"
	"net"
	"strings"
	"sync"
	"time"
)

const DbName = "Anduschain"

var (
	MongDBConnectError = errors.New("fail to connecting mongo database")
)

type MongoDatabase struct {
	mu sync.Mutex

	dbName string

	url      string
	dialInfo *mgo.DialInfo
	mongo    *mgo.Session
	chainID  *big.Int

	chainConfig     *mgo.Collection
	activeNodeCol   *mgo.Collection
	leagues         *mgo.Collection
	otprnList       *mgo.Collection
	voteAggregation *mgo.Collection
	blockChain      *mgo.Collection
	blockChainRaw   *mgo.Collection
	transactions    *mgo.Collection

	activeFairnode *mgo.Collection
}

type config interface {
	GetInfo() (host, port, user, pass, ssl string, chainID *big.Int)
}

// Mongodb url => mongodb://[username:password@]host1[:port1][,host2[:port2],...[,hostN[:portN]]][/[database][?options]]
func NewMongoDatabase(conf config) (*MongoDatabase, error) {
	var db MongoDatabase
	host, port, user, pass, ssl, chainID := conf.GetInfo()
	db.dbName = fmt.Sprintf("%s_%s", DbName, chainID.String())
	if strings.Compare(user, "") != 0 {
		db.url = fmt.Sprintf("mongodb://%s:%s@%s:%s/%s", user, pass, host, port, db.dbName)
	} else {
		db.url = fmt.Sprintf("mongodb://%s:%s/%s", host, port, db.dbName)
	}

	db.chainID = chainID

	// SSL db connection config
	if strings.Compare(ssl, "") != 0 {
		tlsConfig := &tls.Config{}
		tlsConfig.InsecureSkipVerify = true

		roots := x509.NewCertPool()
		if ca, err := ioutil.ReadFile(ssl); err == nil {
			roots.AppendCertsFromPEM(ca)
		}

		tlsConfig.RootCAs = roots

		dialInfo, err := mgo.ParseURL(db.url)
		if err != nil {
			return nil, err
		}

		dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
			conn, err := tls.Dial("tcp", addr.String(), tlsConfig)
			return conn, err
		}

		db.dialInfo = dialInfo
	}

	return &db, nil
}

func (m *MongoDatabase) Start() error {
	var session *mgo.Session
	var err error
	if m.dialInfo == nil {
		session, err = mgo.Dial(m.url)
	} else {
		session, err = mgo.DialWithInfo(m.dialInfo)
	}

	if err != nil {
		logger.Error("Mongo DB Dial", "mongo", err)
		return MongDBConnectError
	}

	session.SetMode(mgo.Strong, true)

	m.mongo = session
	m.chainConfig = session.DB(m.dbName).C("ChainConfig")
	m.activeNodeCol = session.DB(m.dbName).C("ActiveNode")
	m.leagues = session.DB(m.dbName).C("Leagues")
	m.otprnList = session.DB(m.dbName).C("OtprnList")
	m.blockChain = session.DB(m.dbName).C("BlockChain")
	m.blockChainRaw = session.DB(m.dbName).C("BlockChainRaw")
	m.voteAggregation = session.DB(m.dbName).C("VoteAggregation")
	m.transactions = session.DB(m.dbName).C("Transactions")

	m.activeFairnode = session.DB(m.dbName).C("ActiveFairnode")

	logger.Debug("Start fairnode mongo database", "chainID", m.chainID.String(), "url", m.url)
	return nil
}

func (m *MongoDatabase) Stop() {
	m.mongo.Close()
	logger.Debug("Stop fairnode mongo database")
}

func (m *MongoDatabase) GetChainConfig() *types.ChainConfig {
	var num uint64
	current := m.CurrentInfo()
	if current == nil {
		logger.Warn("Get current block number", "database", "mongo", "current", num)
	} else {
		num = current.Number.Uint64()
	}

	conf := new(fntype.Config)
	err := m.chainConfig.Find(bson.M{}).Sort("-timestamp").One(&conf)
	if err != nil {
		logger.Error("Get chain conifg", "msg", err)
		return nil
	}

	return &types.ChainConfig{
		BlockNumber: conf.Config.BlockNumber,
		JoinTxPrice: conf.Config.JoinTxPrice,
		FnFee:       conf.Config.FnFee,
		Mminer:      conf.Config.Mminer,
		Epoch:       conf.Config.Epoch,
		NodeVersion: conf.Config.NodeVersion,
		Sign:        conf.Config.Sign,
	}
}

func (m *MongoDatabase) SaveChainConfig(config *types.ChainConfig) error {
	err := m.chainConfig.Insert(fntype.Config{
		Config: fntype.ChainConfig{
			BlockNumber: config.BlockNumber,
			JoinTxPrice: config.JoinTxPrice,
			FnFee:       config.FnFee,
			Mminer:      config.Mminer,
			Epoch:       config.Epoch,
			NodeVersion: config.NodeVersion,
			Sign:        config.Sign,
		},
		Timestamp: time.Now().Unix(),
	})
	if err != nil {
		logger.Error("Save chain config", "database", "mongo", "msg", err)
	}
	return nil
}

func (m *MongoDatabase) CurrentInfo() *types.CurrentInfo {
	b := new(fntype.BlockHeader)
	err := m.blockChain.Find(bson.M{}).Sort("-header.number").One(b)
	if err != nil {
		logger.Error("Get current info", "database", "mongo", "msg", err)
		return nil
	}
	return &types.CurrentInfo{
		Number: new(big.Int).SetUint64(b.Header.Number),
		Hash:   common.HexToHash(b.Hash),
	}
}

func (m *MongoDatabase) CurrentBlock() *types.Block {
	b := new(fntype.Block)
	err := m.blockChain.Find(bson.M{}).Sort("-header.number").One(b)
	if err != nil {
		if mgo.ErrNotFound != err {
			logger.Error("Get current block", "database", "mongo", "msg", err)
		}
		return nil
	}
	return m.GetBlock(common.HexToHash(b.Hash))
}

func (m *MongoDatabase) CurrentOtprn() *types.Otprn {
	otprn, err := RecvOtprn(m.otprnList.Find(bson.M{}).Sort("-timestamp"))
	if err != nil {
		logger.Error("Get otprn", "database", "mongo", "msg", err)
		return nil
	}
	return otprn
}

func (m *MongoDatabase) InitActiveNode() {
	node := new(fntype.HeartBeat)
	for m.activeNodeCol.Find(bson.M{}).Iter().Next(node) {
		err := m.activeNodeCol.RemoveId(node.Enode)
		if err != nil {
			logger.Error("InitActiveNode Active Node", "database", "mongo", "msg", err)
		}
	}
}

func (m *MongoDatabase) SaveActiveNode(node types.HeartBeat) {
	_, err := m.activeNodeCol.UpsertId(node.Enode, &fntype.HeartBeat{
		Enode:        node.Enode,
		MinerAddress: node.MinerAddress,
		ChainID:      node.ChainID,
		NodeVersion:  node.NodeVersion,
		Host:         node.Host,
		Port:         node.Port,
		Time:         node.Time.Int64(),
		Head:         node.Head.String(),
		Sign:         node.Sign,
	})

	if err != nil {
		logger.Error("Save Active Node", "database", "mongo", "msg", err)
	}
}

func (m *MongoDatabase) GetActiveNode() []types.HeartBeat {
	var nodes []types.HeartBeat
	sNode := new([]fntype.HeartBeat)
	err := m.activeNodeCol.Find(bson.M{}).All(sNode)
	if err != nil {
		logger.Error("Get Active Node", "database", "mongo", "msg", err)
	}
	for _, node := range *sNode {
		nodes = append(nodes, types.HeartBeat{
			Enode:        node.Enode,
			MinerAddress: node.MinerAddress,
			ChainID:      node.ChainID,
			NodeVersion:  node.NodeVersion,
			Host:         node.Host,
			Port:         node.Port,
			Time:         new(big.Int).SetInt64(node.Time),
			Head:         common.HexToHash(node.Head),
			Sign:         node.Sign,
		})
	}
	return nodes
}

func (m *MongoDatabase) RemoveActiveNode(enode string) {
	err := m.activeNodeCol.RemoveId(enode)
	if err != nil {
		logger.Error("Remove Active Node", "database", "mongo", "msg", err)
	}
}

func (m *MongoDatabase) SaveOtprn(otprn types.Otprn) {
	tOtp, err := TransOtprn(otprn)
	if err != nil {
		logger.Error("Save otprn trans otprn", "database", "mongo", "msg", err)
	}

	err = m.otprnList.Insert(tOtp)
	if err != nil {
		logger.Error("Save otprn insert", "database", "mongo", "msg", err)
	}
}

func (m *MongoDatabase) GetOtprn(otprnHash common.Hash) *types.Otprn {
	otprn, err := RecvOtprn(m.otprnList.FindId(otprnHash.String()))
	if err != nil {
		logger.Error("Get otprn", "database", "mongo", "msg", err)
		return nil
	}
	return otprn
}

func (m *MongoDatabase) SaveLeague(otprnHash common.Hash, enode string) {
	node := new(fntype.HeartBeat)
	err := m.activeNodeCol.FindId(enode).One(node)
	if err != nil {
		logger.Error("Save League, Get active node", "database", "mongo", "enode", enode, "msg", err)
	}

	nodes := m.GetLeagueList(otprnHash)
	for _, node := range nodes {
		if node.Enode == enode {
			return
		}
	}

	_, err = m.leagues.UpsertId(otprnHash.String(), bson.M{"$addToSet": bson.M{"nodes": node}})
	if err != nil {
		logger.Error("Save League update or insert", "database", "mongo", "msg", err)
	}
}

func (m *MongoDatabase) GetLeagueList(otprnHash common.Hash) []types.HeartBeat {
	league := new(fntype.League)
	err := m.leagues.FindId(otprnHash.String()).One(league)
	if err != nil {
		if mgo.ErrNotFound != err {
			logger.Error("Gat League List", "msg", err)
		}
	}
	var nodes []types.HeartBeat
	for _, node := range league.Nodes {
		nodes = append(nodes, types.HeartBeat{
			Enode:        node.Enode,
			MinerAddress: node.MinerAddress,
			ChainID:      node.ChainID,
			NodeVersion:  node.NodeVersion,
			Host:         node.Host,
			Port:         node.Port,
			Time:         new(big.Int).SetInt64(node.Time),
			Head:         common.HexToHash(node.Head),
			Sign:         node.Sign,
		})
	}
	return nodes
}

func (m *MongoDatabase) SaveVote(otprn common.Hash, blockNum *big.Int, vote *types.Voter) {
	voteKey := MakeVoteKey(otprn, blockNum)
	_, err := m.voteAggregation.UpsertId(voteKey.String(), bson.M{"$addToSet": bson.M{
		"voters": fntype.Voter{
			Header:   vote.Header,
			Voter:    vote.Voter.String(),
			VoteSign: vote.VoteSign,
		}}})
	if err != nil {
		logger.Error("Save vote update or insert", "database", "mongo", "msg", err)
	}
}

func (m *MongoDatabase) GetVoters(votekey common.Hash) []*types.Voter {
	vt := new(fntype.VoteAggregation)
	err := m.voteAggregation.FindId(votekey.String()).One(vt)
	if err != nil {
		logger.Error("Get voters update or insert", "database", "mongo", "msg", err)
	}
	var voters []*types.Voter
	for _, vote := range vt.Voters {
		voters = append(voters, &types.Voter{
			Header:   vote.Header,
			Voter:    common.HexToAddress(vote.Voter),
			VoteSign: vote.VoteSign,
		})
	}
	return voters
}

func (m *MongoDatabase) SaveFinalBlock(block *types.Block, byteBlock []byte) error {
	err := m.blockChain.Insert(TransBlock(block))
	if err != nil {
		if !mgo.IsDup(err) {
			return err
		}
	}

	err = m.blockChainRaw.Insert(fntype.RawBlock{
		Hash: block.Hash().String(),
		Raw:  common.Bytes2Hex(byteBlock),
	})
	if err != nil {
		if !mgo.IsDup(err) {
			return err
		}
	}

	TransTransaction(block, m.chainID, m.transactions)
	return nil
}

func (m *MongoDatabase) GetBlock(blockHash common.Hash) *types.Block {
	b := new(fntype.RawBlock)
	err := m.blockChainRaw.FindId(blockHash.String()).One(b)
	if err != nil {
		logger.Error("Get block", "database", "mongo", "msg", err)
		return nil
	}

	blockEnc := common.FromHex(b.Raw)
	block := new(types.Block)
	if err := rlp.DecodeBytes(blockEnc, block); err != nil {
		logger.Error("Get block, decode", "database", "mongo", "msg", err)
		return nil
	}
	return block
}

func (m *MongoDatabase) RemoveBlock(blockHash common.Hash) {
	err := m.blockChain.RemoveId(blockHash.String())
	if err != nil {
		logger.Error("Remove Block", "database", "mongo", "msg", err)
	}
	err = m.blockChainRaw.RemoveId(blockHash.String())
	if err != nil {
		logger.Error("Remove Raw Block", "database", "mongo", "msg", err)
	}
}

type stType uint64

const (
	PENDING stType = iota
	LEADER
	FOLLOWER
)

func (s stType) String() string {
	switch s {
	case PENDING:
		return "PENDING"
	case LEADER:
		return "LEADER"
	case FOLLOWER:
		return "FOLLOWER"
	default:
		return "UNKNOWN"
	}
}

func (m *MongoDatabase) InsertActiveFairnode(nodeKey common.Hash, address string) {
	err := m.activeFairnode.Insert(fntype.Fairnode{ID: nodeKey.String(), Address: address, Status: PENDING.String()})
	if err != nil {
		if !mgo.IsDup(err) {
			logger.Error("Insert Active Fairnode", "database", "mongo", "msg", err)
		}
	}
}

func (m *MongoDatabase) GetActiveFairnodes() map[common.Hash]map[string]string {
	res := make(map[common.Hash]map[string]string)
	var nodes []fntype.Fairnode
	err := m.activeFairnode.Find(bson.M{}).All(&nodes)
	if err != nil {
		logger.Error("Get Active Fairnode", "database", "mongo", "msg", err)
		return nil
	}
	for _, node := range nodes {
		res[common.HexToHash(node.ID)] = make(map[string]string)
		res[common.HexToHash(node.ID)]["ADDRESS"] = node.Address
		res[common.HexToHash(node.ID)]["STATUS"] = node.Status
	}
	return res
}

func (m *MongoDatabase) RemoveActiveFairnode(nodeKey common.Hash) {
	err := m.activeFairnode.RemoveId(nodeKey.String())
	if err != nil {
		if mgo.ErrNotFound != err {
			logger.Error("Remove Active Fairnode", "database", "mongo", "msg", err)
		}
	}
}

func (m *MongoDatabase) UpdateActiveFairnode(nodeKey common.Hash, status uint64) {
	node := new(fntype.Fairnode)
	err := m.activeFairnode.FindId(nodeKey.String()).One(node)
	if err != nil {
		logger.Error("Update Active Fairnode, Get node", "database", "mongo", "msg", err)
	}
	node.Status = stType(status).String()
	err = m.activeFairnode.UpdateId(nodeKey.String(), node)
	if err != nil {
		logger.Error("Update Active Fairnode", "database", "mongo", "msg", err)
	}
}

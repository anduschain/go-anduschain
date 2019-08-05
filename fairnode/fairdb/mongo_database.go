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

const DbName = "AndusChainTestnet"

var (
	MongDBConnectError = errors.New("fail to connecting mongodb database")
)

type MongoDatabase struct {
	mu sync.Mutex

	url      string
	dialInfo *mgo.DialInfo
	mongo    *mgo.Session

	chainConfig     *mgo.Collection
	activeNodeCol   *mgo.Collection
	leagues         *mgo.Collection
	otprnList       *mgo.Collection
	voteAggregation *mgo.Collection
	blockChain      *mgo.Collection
	blockChainRaw   *mgo.Collection
	transactions    *mgo.Collection
}

type config interface {
	GetInfo() (host, port, user, pass, ssl string)
}

// Mongodb url => mongodb://[username:password@]host1[:port1][,host2[:port2],...[,hostN[:portN]]][/[database][?options]]
func NewMongoDatabase(conf config) (*MongoDatabase, error) {
	var db MongoDatabase
	host, port, user, pass, ssl := conf.GetInfo()
	if strings.Compare(user, "") != 0 {
		db.url = fmt.Sprintf("mongodb://%s:%s@%s:%s/%s", user, pass, host, ssl, DbName)
	} else {
		db.url = fmt.Sprintf("mongodb://%s:%s/%s", host, port, DbName)
	}

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

	session.SetMode(mgo.Monotonic, true)

	m.mongo = session
	m.chainConfig = session.DB(DbName).C("ChainConfig")
	m.activeNodeCol = session.DB(DbName).C("ActiveNode")
	m.leagues = session.DB(DbName).C("Leagues")
	m.otprnList = session.DB(DbName).C("OtprnList")
	m.blockChain = session.DB(DbName).C("BlockChain")
	m.blockChainRaw = session.DB(DbName).C("BlockChainRaw")
	m.voteAggregation = session.DB(DbName).C("VoteAggregation")
	m.transactions = session.DB(DbName).C("Transactions")

	logger.Debug("Start fairnode mongo database")
	return nil
}

func (m *MongoDatabase) Stop() {
	m.mongo.Close()
	logger.Debug("Stop fairnode mongo database")
}

func (m *MongoDatabase) GetChainConfig() *types.ChainConfig {
	var num uint64
	block := m.CurrentBlock()
	if block == nil {
		logger.Warn("get current block number", "database", "mongo", "current", num)
	} else {
		num = block.Number().Uint64()
	}

	conf := new(fntype.Config)
	err := m.chainConfig.Find(bson.M{}).Sort("-timestamp").One(&conf)
	if err != nil {
		logger.Error("get chain conifg", "msg", err)
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

func (m *MongoDatabase) CurrentBlock() *types.Block {
	b := new(fntype.Block)
	err := m.blockChain.Find(bson.M{}).Sort("-header.number").One(b)
	if err != nil {
		logger.Error("get current block", "database", "mongo", "msg", err)
		return nil
	}
	return m.GetBlock(common.HexToHash(b.Hash))
}

func (m *MongoDatabase) CurrentOtprn() *types.Otprn {
	otprn, err := RecvOtprn(m.otprnList.Find(bson.M{}).Sort("-timestamp"))
	if err != nil {
		logger.Error("get otprn", "database", "mongo", "msg", err)
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
		logger.Error("Gat League List", "msg", err)
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
		return err
	}

	err = m.blockChainRaw.Insert(fntype.RawBlock{
		Hash: block.Hash().String(),
		Raw:  common.Bytes2Hex(byteBlock),
	})
	if err != nil {
		return err
	}
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

package fairdb

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"math/big"
	"net"
	"strings"
	"sync"
)

const dbName = "AndusChain"

var (
	mongDBConnectError = errors.New("fail to connecting mongodb database")
)

type MongoDatabase struct {
	mu sync.Mutex

	url      string
	dialInfo *mgo.DialInfo
	mongo    *mgo.Session

	chainConfig   *mgo.Collection
	activeNodeCol *mgo.Collection
	minerNode     *mgo.Collection
	otprnList     *mgo.Collection
	blockChain    *mgo.Collection
	blockChainRaw *mgo.Collection
	transactions  *mgo.Collection
}

type config interface {
	GetInfo() (host, port, user, pass, ssl string)
}

// Mongodb url => mongodb://[username:password@]host1[:port1][,host2[:port2],...[,hostN[:portN]]][/[database][?options]]
func NewMongoDatabase(conf config) (*MongoDatabase, error) {
	var db MongoDatabase
	host, port, user, pass, ssl := conf.GetInfo()
	if strings.Compare(user, "") != 0 {
		db.url = fmt.Sprintf("mongodb://%s:%s@%s:%s/%s", user, pass, host, ssl, dbName)
	} else {
		db.url = fmt.Sprintf("mongodb://%s:%s/%s", host, port, dbName)
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
		logger.Error("Mongo DB Dial", "error", err)
		return mongDBConnectError
	}

	session.SetMode(mgo.Monotonic, true)

	m.mongo = session
	m.chainConfig = session.DB(dbName).C("ChainConfig")
	m.activeNodeCol = session.DB(dbName).C("ActiveNode")
	m.minerNode = session.DB(dbName).C("MinerNode")
	m.otprnList = session.DB(dbName).C("OtprnList")
	m.blockChain = session.DB(dbName).C("BlockChain")
	m.blockChainRaw = session.DB(dbName).C("BlockChainRaw")
	m.transactions = session.DB(dbName).C("Transactions")
	logger.Debug("Start fairnode mongo database")
	return nil
}

func (m *MongoDatabase) Stop() {
	m.mongo.Close()
	logger.Debug("Stop fairnode mongo database")
}

func (m *MongoDatabase) GetChainConfig() *types.ChainConfig {
	block := m.CurrentBlock()
	if block == nil {
		return nil
	}
	conf := new(types.ChainConfig)
	err := m.chainConfig.Find(bson.M{"blocknumber": bson.M{"$gte": block.Number().Uint64()}}).Sort("-blocknumber").One(conf)
	if err != nil {
		logger.Error("get chain conifg", "msg", err)
		return nil
	}
	return conf
}

func (m *MongoDatabase) SaveChainConfig(config *types.ChainConfig) error {
	err := m.chainConfig.Insert(config)
	if err != nil {
		logger.Error("Save chain config", "msg", err)
	}
	return nil
}

func (m *MongoDatabase) CurrentBlock() *types.Block {
	b := new(types.Block)
	err := m.blockChain.Find(bson.M{}).Sort("-header.number").Limit(1).One(b)
	if err != nil {
		logger.Error("get current block", "msg", err)
		return nil
	}
	return b
}

func (m *MongoDatabase) CurrentOtprn() *types.Otprn {
	return nil
}

func (m *MongoDatabase) InitActiveNode() {

}

func (m *MongoDatabase) SaveActiveNode(node types.HeartBeat) {

}

func (m *MongoDatabase) GetActiveNode() []types.HeartBeat {
	return nil
}

func (m *MongoDatabase) RemoveActiveNode(enode string) {

}

func (m *MongoDatabase) SaveOtprn(otprn types.Otprn) {

}

func (m *MongoDatabase) GetOtprn(otprnHash common.Hash) *types.Otprn {
	return nil
}

func (m *MongoDatabase) SaveLeague(otprnHash common.Hash, enode string) {

}

func (m *MongoDatabase) GetLeagueList(otprnHash common.Hash) []types.HeartBeat {
	return nil
}

func (m *MongoDatabase) SaveVote(otprn common.Hash, blockNum *big.Int, vote *types.Voter) {

}

func (m *MongoDatabase) GetVoters(votekey common.Hash) []*types.Voter {
	return nil
}

func (m *MongoDatabase) SaveFinalBlock(block *types.Block) error {
	return nil
}

func (m *MongoDatabase) GetBlock(blockHash common.Hash) *types.Block {
	return nil
}

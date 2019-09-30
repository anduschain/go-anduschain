package fairdb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/fairdb/fntype"
	"github.com/anduschain/go-anduschain/rlp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"io/ioutil"
	"log"
	"math/big"
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

	url     string
	chainID *big.Int

	context context.Context
	client  *mongo.Client

	chainConfig     *mongo.Collection
	activeNodeCol   *mongo.Collection
	leagues         *mongo.Collection
	otprnList       *mongo.Collection
	voteAggregation *mongo.Collection
	blockChain      *mongo.Collection
	blockChainRaw   *mongo.Collection
	transactions    *mongo.Collection

	activeFairnode *mongo.Collection

	tlsConfig *tls.Config
}

type config interface {
	GetInfo() (host, port, user, pass, ssl, option string, chainID *big.Int)
}

// Mongodb url => mongodb://[username:password@]host1[:port1][,host2[:port2],...[,hostN[:portN]]][/[database][?options]]
func NewMongoDatabase(conf config) (*MongoDatabase, error) {
	var db MongoDatabase
	var err error
	host, port, user, pass, ssl, option, chainID := conf.GetInfo()

	db.chainID = chainID
	db.tlsConfig = &tls.Config{}
	db.dbName = fmt.Sprintf("%s_%s", DbName, chainID.String())
	if strings.Compare(user, "") != 0 {
		opt := fmt.Sprintf("?%s", option)
		db.url = fmt.Sprintf("mongodb+srv://%s:%s@%s/%s%s", user, pass, host, db.dbName, opt)
	} else {
		db.url = fmt.Sprintf("mongodb://%s:%s/%s", host, port, db.dbName)
	}

	// SSL db connection config
	if strings.Compare(ssl, "") != 0 {
		db.tlsConfig.InsecureSkipVerify = true
		roots := x509.NewCertPool()
		if ca, err := ioutil.ReadFile(ssl); err == nil {
			roots.AppendCertsFromPEM(ca)
		}
		db.tlsConfig.RootCAs = roots
	}

	db.client, err = mongo.NewClient(options.Client().ApplyURI(db.url))
	if err != nil {
		fmt.Println("failed", db.url)
		return nil, err
	}
	db.context = context.Background()

	return &db, nil
}

func (m *MongoDatabase) Start() error {
	err := m.client.Connect(m.context)
	if err != nil {
		log.Fatal(err)
		return MongDBConnectError
	}
	m.chainConfig = m.client.Database(m.dbName).Collection("ChainConfig")
	m.activeNodeCol = m.client.Database(m.dbName).Collection("ActiveNode")
	m.leagues = m.client.Database(m.dbName).Collection("Leagues")
	m.otprnList = m.client.Database(m.dbName).Collection("OtprnList")
	m.blockChain = m.client.Database(m.dbName).Collection("BlockChain")
	m.blockChainRaw = m.client.Database(m.dbName).Collection("BlockChainRaw")
	m.voteAggregation = m.client.Database(m.dbName).Collection("VoteAggregation")
	m.transactions = m.client.Database(m.dbName).Collection("Transactions")

	m.activeFairnode = m.client.Database(m.dbName).Collection("ActiveFairnode")

	//logger.Debug("Start fairnode mongo database", "chainID", m.chainID.String(), "url", m.url)
	fmt.Println("Successfully Connected to MongoDB!")

	return nil
}

func (m *MongoDatabase) Stop() {
	if err := m.client.Disconnect(m.context); err != nil {
		fmt.Errorf("%v", err)
	}
	fmt.Println("successfully disconnected")
}

func (m *MongoDatabase) GetChainConfig() *types.ChainConfig {
	var num uint64
	current := m.CurrentInfo()
	if current == nil {
		//logger.Warn("Get current block number", "database", "mongo", "current", num)
		fmt.Println("Get current block number", "database", "mongo", "current", num)
	} else {
		num = current.Number.Uint64()
	}

	findOneOpts := options.FindOne().SetSort(bson.M{"timestamp": -1})
	conf := new(fntype.Config)
	err := m.chainConfig.FindOne(m.context, bson.M{}, findOneOpts).Decode(&conf)
	if err != nil {
		//logger.Error("Get chain conifg", "msg", err)
		fmt.Println("GetChainConfig : ", err)
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

	_, err := m.chainConfig.InsertOne(m.context, fntype.Config{
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
	//err := m.chainConfig.Insert(fntype.Config{
	//	Config: fntype.ChainConfig{
	//		BlockNumber: config.BlockNumber,
	//		JoinTxPrice: config.JoinTxPrice,
	//		FnFee:       config.FnFee,
	//		Mminer:      config.Mminer,
	//		Epoch:       config.Epoch,
	//		NodeVersion: config.NodeVersion,
	//		Sign:        config.Sign,
	//	},
	//	Timestamp: time.Now().Unix(),
	//})
	if err != nil {
		//logger.Error("Save chain config", "database", "mongo", "msg", err)
		fmt.Println("Save chain config", "database", "mongo", "msg", err)
	}
	return nil
}

func (m *MongoDatabase) CurrentInfo() *types.CurrentInfo {
	findOneOpts := options.FindOne().SetSort(bson.M{"header.number": -1})
	b := new(fntype.BlockHeader)
	//err := m.blockChain.Find(bson.M{}).Sort("-header.number").Limit(1).One(b)
	err := m.blockChain.FindOne(m.context, bson.M{}, findOneOpts).Decode(&b)
	if err != nil {
		//logger.Error("Get current info", "database", "mongo", "msg", err)
		return nil
	}
	return &types.CurrentInfo{
		Number: new(big.Int).SetUint64(b.Header.Number),
		Hash:   common.HexToHash(b.Hash),
	}
}

func (m *MongoDatabase) CurrentBlock() *types.Block {
	findOneOpts := options.FindOne().SetSort(bson.M{"header.number": -1})
	b := new(fntype.Block)
	err := m.blockChain.FindOne(m.context, bson.M{}, findOneOpts).Decode(&b)
	if err != nil {
		fmt.Println("err code : ", err)
		//if mgo.ErrNotFound != err {
		//	logger.Error("Get current block", "database", "mongo", "msg", err)
		//}
		return nil
	}
	return m.GetBlock(common.HexToHash(b.Hash))
}

func (m *MongoDatabase) CurrentOtprn() *types.Otprn {
	findOneOpts := options.FindOne().
		SetSort(bson.M{"timestamp": -1})

	otprn, err := RecvOtprn(m.otprnList.FindOne(m.context, bson.M{}, findOneOpts))
	if err != nil {
		//logger.Error("Get otprn", "database", "mongo", "msg", err)
		fmt.Println("Get otprn", "database", "mongo", "msg", err)
		return nil
	}
	return otprn
}

// Fairnode가 기동될 때, 기존 노드 정보를 지운다. (새로 갱신함)
func (m *MongoDatabase) InitActiveNode() {
	//node := new(fntype.HeartBeat)
	_, err := m.activeNodeCol.DeleteMany(m.context, bson.M{})
	if err != nil {
		//logger.Error("InitActiveNode Active Node", "database", "mongo", "msg", err)
		fmt.Println("InitActiveNode Active Node", "database", "mongo", "msg", err)
	}
	//var target []string
	//cur, _ := m.activeNodeCol.Find(m.context, bson.M{})
	//for cur.Next(m.context) {
	//	cur.Decode(node)
	//	target = append(target, node.Enode)
	//}
	//for enode := range target {
	//	_, err := m.activeNodeCol.DeleteMany(m.context, enode)
	//	if err != nil {
	//		//logger.Error("InitActiveNode Active Node", "database", "mongo", "msg", err)
	//		fmt.Println("InitActiveNode Active Node", "database", "mongo", "msg", err)
	//	}
	//}
}

func (m *MongoDatabase) SaveActiveNode(node types.HeartBeat) {
	updateOpts := options.Update().SetUpsert(true)
	_, err := m.activeNodeCol.UpdateOne(m.context, bson.M{"_id": node.Enode}, bson.D{{"$set", fntype.HeartBeat{
		Enode:        node.Enode,
		MinerAddress: node.MinerAddress,
		ChainID:      node.ChainID,
		NodeVersion:  node.NodeVersion,
		Host:         node.Host,
		Port:         node.Port,
		Time:         node.Time.Int64(),
		Head:         node.Head.String(),
		Sign:         node.Sign,
	}}}, updateOpts)
	if err != nil {
		//logger.Error("Save Active Node", "database", "mongo", "msg", err)
		fmt.Println("Save Active Node :", err)
	} else {
		fmt.Println("Save Active Node not nil")
	}

}

func (m *MongoDatabase) GetActiveNode() []types.HeartBeat {
	var nodes []types.HeartBeat
	sNode := new([]fntype.HeartBeat)
	cur, err := m.activeNodeCol.Find(m.context, bson.M{})
	cur.All(m.context, sNode)
	if err != nil {
		//logger.Error("Get Active Node", "database", "mongo", "msg", err)
		fmt.Println("Get Active Node", "database", "mongo", "msg", err)
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
	_, err := m.activeNodeCol.DeleteOne(m.context, bson.M{"_id": enode})
	//err := m.activeNodeCol.RemoveId(enode)
	if err != nil {
		//logger.Error("Remove Active Node", "database", "mongo", "msg", err)
		fmt.Println("Remove Active Node", "database", "mongo", "msg", err)
	}
}

func (m *MongoDatabase) SaveOtprn(otprn types.Otprn) {
	tOtp, err := TransOtprn(otprn)
	if err != nil {
		logger.Error("Save otprn trans otprn", "database", "mongo", "msg", err)
	}
	_, err = m.otprnList.InsertOne(m.context, tOtp)
	if err != nil {
		logger.Error("Save otprn insert", "database", "mongo", "msg", err)
	}
}

func (m *MongoDatabase) GetOtprn(otprnHash common.Hash) *types.Otprn {
	otprn, err := RecvOtprn(m.otprnList.FindOne(m.context, bson.M{"_id": otprnHash.String()}))
	if err != nil {
		logger.Error("Get otprn", "database", "mongo", "msg", err)
		return nil
	}
	return otprn
}

func (m *MongoDatabase) SaveLeague(otprnHash common.Hash, enode string) {
	nodes := m.GetLeagueList(otprnHash)
	for _, node := range nodes {
		if node.Enode == enode {
			return
		}
	}
	node := new(fntype.HeartBeat)
	err := m.activeNodeCol.FindOne(m.context, bson.M{"_id": enode}).Decode(node)
	if err != nil {
		//logger.Error("Save League, Get active node", "database", "mongo", "enode", enode, "msg", err)
		fmt.Println("Save League, Get active node", "database", "mongo", "enode", enode, "msg", err)
	}
	updateOpts := options.Update().SetUpsert(true)
	_, err = m.leagues.UpdateOne(m.context, bson.M{"_id":otprnHash.String()}, bson.M{"$addToSet": bson.M{"nodes": node}}, updateOpts)
	//_, err = m.leagues.UpsertId(otprnHash.String(), bson.M{"$addToSet": bson.M{"nodes": node}})
	if err != nil {
		//logger.Error("Save League update or insert", "database", "mongo", "msg", err)
		fmt.Println("Save League update or insert", "database", "mongo", "msg", err)
	}
}

func (m *MongoDatabase) GetLeagueList(otprnHash common.Hash) []types.HeartBeat {
	league := new(fntype.League)
	err := m.leagues.FindOne(m.context, bson.M{"_id": otprnHash.String()}).Decode(league)
	if err != nil {
		//if mgo.ErrNotFound != err {
		//	logger.Error("Gat League List", "msg", err)
		//}
		fmt.Println("GetLeagueList", err)
		return nil
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
	updateOpts := options.Update().SetUpsert(true)
	_, err := m.voteAggregation.UpdateOne(m.context, bson.M{"_id":voteKey.String()}, bson.M{"$addToSet": bson.M{"voters": fntype.Voter{
		Header:   vote.Header,
		Voter:    vote.Voter.String(),
		VoteSign: vote.VoteSign,
	}}}, updateOpts)
	if err != nil {
		//logger.Error("Save vote update or insert", "database", "mongo", "msg", err)
		fmt.Println("Save vote update or insert", "database", "mongo", "msg", err)
	}
}

func (m *MongoDatabase) GetVoters(votekey common.Hash) []*types.Voter {
	vt := new(fntype.VoteAggregation)
	err := m.voteAggregation.FindOne(m.context, bson.M{"id": votekey.String()}).Decode(vt)
	if err != nil {
		//logger.Error("Get voters update or insert", "database", "mongo", "msg", err)
		fmt.Println("Get voters update or insert", "database", "mongo", "msg", err)
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
	_, err := m.blockChain.InsertOne(m.context, TransBlock(block))
	//err := m.blockChain.Insert(TransBlock(block))
	if err != nil {
		fmt.Println(err)
		//if !mgo.IsDup(err) {
		//	return err
		//}
	}

	_, err = m.blockChainRaw.InsertOne(m.context, fntype.RawBlock{
		Hash: block.Hash().String(),
		Raw:  common.Bytes2Hex(byteBlock),
	})

	//err = m.blockChainRaw.Insert(fntype.RawBlock{
	//	Hash: block.Hash().String(),
	//	Raw:  common.Bytes2Hex(byteBlock),
	//})
	if err != nil {
		fmt.Println(err)
		//if !mgo.IsDup(err) {
		//	return err
		//}
	}

	TransTransaction(block, m.chainID, m.transactions)
	return nil
}

func (m *MongoDatabase) GetBlock(blockHash common.Hash) *types.Block {
	b := new(fntype.RawBlock)
	err := m.blockChainRaw.FindOne(m.context, bson.M{"_id": blockHash.String()}).Decode(&b)
	if err != nil {
		//if err != mgo.ErrNotFound {
		//logger.Error("Get block", "database", "mongo", "msg", err)
		fmt.Println("Get block", "database", "mongo", "msg", err)
		//}
		return nil
	}

	blockEnc := common.FromHex(b.Raw)
	block := new(types.Block)
	if err := rlp.DecodeBytes(blockEnc, block); err != nil {
		//logger.Error("Get block, decode", "database", "mongo", "msg", err)
		fmt.Println("Get block", "database", "mongo", "msg", err)
		return nil
	}
	return block
}

func (m *MongoDatabase) RemoveBlock(blockHash common.Hash) {
	_, err := m.blockChain.DeleteOne(m.context, bson.M{"_id": blockHash.String()})
	//err := m.activeNodeCol.RemoveId(enode)
	if err != nil {
		//logger.Error("Remove Block", "database", "mongo", "msg", err)
		fmt.Println("Remove Block", "database", "mongo", "msg", err)
	}
	_, err = m.blockChainRaw.DeleteOne(m.context, bson.M{"_id": blockHash.String()})
	//err := m.activeNodeCol.RemoveId(enode)
	if err != nil {
		//logger.Error("Remove Raw Block", "database", "mongo", "msg", err)
		fmt.Println("Remove Raw Block", "database", "mongo", "msg", err)
	}

	//
	//err := m.blockChain.RemoveId(blockHash.String())
	//if err != nil {
	//	logger.Error("Remove Block", "database", "mongo", "msg", err)
	//}
	//err = m.blockChainRaw.RemoveId(blockHash.String())
	//if err != nil {
	//	logger.Error("Remove Raw Block", "database", "mongo", "msg", err)
	//}
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
_, err := m.activeFairnode.InsertOne(m.context, fntype.Fairnode{ID: nodeKey.String(), Address: address, Status: PENDING.String()})
if err != nil {
fmt.Println("Insert Active Fairnode", "database", "mongo", "msg", err)
//if !mgo.IsDup(err) {
//	logger.Error("Insert Active Fairnode", "database", "mongo", "msg", err)
//}
}
}

func (m *MongoDatabase) GetActiveFairnodes() map[common.Hash]map[string]string {
	res := make(map[common.Hash]map[string]string)
	var nodes []fntype.Fairnode
	cur, err := m.activeFairnode.Find(m.context, bson.M{})
	cur.All(m.context, nodes)
	if err != nil {
		//logger.Error("Get Active Fairnode", "database", "mongo", "msg", err)
		fmt.Println("Get Active Fairnode", "database", "mongo", "msg", err)
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
	_, err := m.activeFairnode.DeleteOne(m.context, bson.M{"_id": nodeKey.String()})
	if err != nil {
		fmt.Println("Remove Active Fairnode", "database", "mongo", "msg", err)
		if err != mongo.ErrNoDocuments {
			fmt.Println("Remove Active Fairnode", "database", "mongo", "msg", err)
		}
		//if mgo.ErrNotFound != err {
		//	logger.Error("Remove Active Fairnode", "database", "mongo", "msg", err)
		//}
	}
}

//nodeKey 에 해당하는 node의 status 업데이트
func (m *MongoDatabase) UpdateActiveFairnode(nodeKey common.Hash, status uint64) {
	node := new(fntype.Fairnode)
	//err := m.activeFairnode.FindId(nodeKey.String()).One(node)
	//if err != nil {
	//	logger.Error("Update Active Fairnode, Get node", "database", "mongo", "msg", err)
	//}
	node.Status = stType(status).String()

	_, err := m.activeFairnode.UpdateOne(m.context, bson.M{"_id":nodeKey.String()}, bson.M{"$set":bson.M{"status": node.Status}})
	//err = m.activeFairnode.UpdateId(nodeKey.String(), node)
	if err != nil {
		//logger.Error("Update Active Fairnode", "database", "mongo", "msg", err)
		fmt.Println("Update Active Fairnode", "database", "mongo", "msg", err)
	}
}

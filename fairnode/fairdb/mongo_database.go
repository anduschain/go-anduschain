package fairdb

import (
	"context"
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/fairdb/fntype"
	"github.com/anduschain/go-anduschain/rlp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"math/big"
	"strings"
	"sync"
	"time"
)

const DbName = "Anduschain"

var (
	MongDBConnectError = errors.New("fail to connecting mongo database")
	MongDBPingFail     = errors.New("ping failed")
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
	activeFairnode  *mongo.Collection
}

type config interface {
	GetInfo() (useSRV bool, host, port, user, pass, ssl, option string, chainID *big.Int)
}

// E11000 duplicate key error
// reference mgo.IsDup
func IsDup(err error) bool {
	switch e := err.(type) {
	case mongo.CommandError:
		if e.Code == 11000 {
			return true
		}
		logger.Error("CommandError", e.Code, e.Message)
	case mongo.WriteError:
		if e.Code == 11000 {
			return true
		}
		logger.Error("WriteError", e.Code, e.Message)
	case mongo.WriteConcernError:
		if e.Code == 11000 {
			return true
		}
		logger.Error("WriteConcernError", e.Code, e.Message)
	case mongo.BulkWriteError:
		if e.Code == 11000 {
			return true
		}
		logger.Error("BulkWriteError", e.Code, e.Message)
	case mongo.WriteException:
		for _, we := range e.WriteErrors {
			if we.Code == 11000 {
				return true
			}
			logger.Error("WriteException")
			logger.Error("err idx : ", we.Index)
			logger.Error("err code : ", we.Code)
			logger.Error("err msg : ", we.Message)
		}
	case mongo.BulkWriteException:
		for _, we := range e.WriteErrors {
			if we.Code == 11000 {
				return true
			}
			logger.Error("BulkWriteException")
			logger.Error("err idx : ", we.Index)
			logger.Error("err code : ", we.Code)
			logger.Error("err msg : ", we.Message)
		}
	default:
		logger.Error("default : ", e.Error())
	}
	return false
}

func NewMongoDatabase(conf config) (*MongoDatabase, error) {
	var db MongoDatabase
	var err error
	var protocol, userPass, dbOpt string
	useSRV, host, port, user, pass, ssl, option, chainID := conf.GetInfo()
	// prevent unused
	_ = ssl
	_ = port
	_ = userPass

	if useSRV {
		protocol = fmt.Sprint("mongodb+srv")
	} else {
		protocol = fmt.Sprint("mongodb")
	}
	//if strings.Compare(user, "") != 0 {
	//	userPass = fmt.Sprintf("%s:%s@", user, pass)
	//}
	//if strings.Compare(dbname, "") != 0 {
	//	db.dbName = fmt.Sprintf("%s", dbname)
	//}
	if strings.Compare(option, "") != 0 {
		dbOpt = fmt.Sprintf("?%s", option)
	}
	db.chainID = chainID
	db.dbName = fmt.Sprintf("%s_%s", DbName, chainID.String())

	//db.url = "mongodb://localhost:27020,localhost:27021,localhost:27022/AndusChain_91386209?replSet=replication"
	//db.url = fmt.Sprintf("%s://%s%s/%s%s", protocol, userPass, host, db.dbName, dbOpt)
	db.url = fmt.Sprintf("%s://%s/%s%s", protocol, host, db.dbName, dbOpt)

	// 필요시 ApplyURI() 대신 직접 options.Client()을 Set...() 을 수행
	credential := options.Credential{
		Username: user,
		Password: pass,
	}
	log.Println(db.url, user, pass)
	db.client, err = mongo.NewClient(options.Client().ApplyURI(db.url).SetAuth(credential))
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return &db, nil
}

func (m *MongoDatabase) Start() error {
	m.context = context.Background()
	err := m.client.Connect(m.context)
	if err != nil {
		log.Fatal(err)
		return MongDBConnectError
	}

	err = m.client.Ping(m.context, readpref.Primary())
	if err != nil {
		log.Fatal(err)
		return MongDBPingFail
	}

	fmt.Println("Successfully Connected to MongoDB!")

	m.chainConfig = m.client.Database(m.dbName).Collection("ChainConfig")
	m.activeNodeCol = m.client.Database(m.dbName).Collection("ActiveNode")
	m.leagues = m.client.Database(m.dbName).Collection("Leagues")
	m.otprnList = m.client.Database(m.dbName).Collection("OtprnList")
	m.blockChain = m.client.Database(m.dbName).Collection("BlockChain")
	m.blockChainRaw = m.client.Database(m.dbName).Collection("BlockChainRaw")
	m.voteAggregation = m.client.Database(m.dbName).Collection("VoteAggregation")
	m.transactions = m.client.Database(m.dbName).Collection("Transactions")

	m.activeFairnode = m.client.Database(m.dbName).Collection("ActiveFairnode")

	logger.Debug("Start fairnode mongo database", "chainID", m.chainID.String(), "url", m.url)

	return nil
}

func (m *MongoDatabase) Stop() {
	if err := m.client.Disconnect(m.context); err != nil {
		fmt.Errorf("%v", err)
	}
	fmt.Println("successfully disconnected")
}

func (m *MongoDatabase) GetChainConfig() *types.ChainConfig {
	//fmt.Println("getchainconfig")
	var num uint64
	current := m.CurrentInfo()
	if current == nil {
		logger.Warn("Get current block number", "database", "mongo", "current", num)
	} else {
		num = current.Number.Uint64()
	}
	findOneOpts := options.FindOne().SetSort(bson.M{"timestamp": -1})
	conf := new(fntype.Config)
	err := m.chainConfig.FindOne(m.context, bson.M{}, findOneOpts).Decode(conf)
	if err != nil {
		logger.Error("Get chain conifg", "msg", err)
		return nil
	}
	return &types.ChainConfig{
		MinMiner:    conf.Config.MinMiner,
		BlockNumber: conf.Config.BlockNumber,
		FnFee:       conf.Config.FnFee,
		Mminer:      conf.Config.Mminer,
		Epoch:       conf.Config.Epoch,
		NodeVersion: conf.Config.NodeVersion,
		Sign:        conf.Config.Sign,
		Price: types.Price{
			GasLimit:    conf.Config.Price.GasLimit,
			GasPrice:    conf.Config.Price.GasPrice,
			JoinTxPrice: conf.Config.Price.JoinTxPrice,
		},
	}
}

func (m *MongoDatabase) SaveChainConfig(config *types.ChainConfig) error {
	//fmt.Println("SaveChainConfig")
	_, err := m.chainConfig.InsertOne(m.context, fntype.Config{
		Config: fntype.ChainConfig{
			MinMiner:    config.MinMiner,
			BlockNumber: config.BlockNumber,
			FnFee:       config.FnFee,
			Mminer:      config.Mminer,
			Epoch:       config.Epoch,
			NodeVersion: config.NodeVersion,
			Sign:        config.Sign,
			Price: fntype.Price{
				GasLimit:    config.Price.GasLimit,
				GasPrice:    config.Price.GasPrice,
				JoinTxPrice: config.Price.JoinTxPrice,
			},
		},
		Timestamp: time.Now().Unix(),
	})
	if err != nil {
		logger.Error("Save chain config", "database", "mongo", "msg", err)
	}
	return nil
}

func (m *MongoDatabase) CurrentInfo() *types.CurrentInfo {
	findOneOpts := options.FindOne().SetSort(bson.M{"header.number": -1})
	b := new(fntype.BlockHeader)
	err := m.blockChain.FindOne(m.context, bson.M{}, findOneOpts).Decode(b)
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
	//fmt.Println("CurrentBlock")
	findOneOpts := options.FindOne().SetSort(bson.M{"header.number": -1})
	b := new(fntype.Block)
	err := m.blockChain.FindOne(m.context, bson.M{}, findOneOpts).Decode(b)
	if err != nil {
		if err != mongo.ErrNoDocuments {
			logger.Error("Get current block", "database", "mongo", "msg", err)
		}
		return nil
	}
	return m.GetBlock(common.HexToHash(b.Hash))
}

func (m *MongoDatabase) CurrentOtprn() *types.Otprn {
	//fmt.Println("CurrentOtprn")
	findOneOpts := options.FindOne().
		SetSort(bson.M{"timestamp": -1})

	otprn, err := RecvOtprn(m.otprnList.FindOne(m.context, bson.M{}, findOneOpts))
	if err != nil {
		logger.Error("Get otprn", "database", "mongo", "msg", err)
		return nil
	}
	return otprn
}

// Fairnode가 기동될 때, 기존 노드 정보를 지운다. (새로 갱신함)
func (m *MongoDatabase) InitActiveNode() {
	//fmt.Println("InitActiveNode")
	//node := new(fntype.HeartBeat)
	_, err := m.activeNodeCol.DeleteMany(m.context, bson.M{})
	if err != nil {
		logger.Error("InitActiveNode Active Node", "database", "mongo", "msg", err)
	}
}

func (m *MongoDatabase) SaveActiveNode(node types.HeartBeat) {
	//fmt.Println("SaveActiveNode : ")
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
		logger.Error("Save Active Node", "database", "mongo", "msg", err)
	}

}

func (m *MongoDatabase) GetActiveNode() []types.HeartBeat {
	//fmt.Println("GetActiveNode")
	var nodes []types.HeartBeat
	var sNode []fntype.HeartBeat
	cur, err := m.activeNodeCol.Find(m.context, bson.M{})
	if err != nil {
		logger.Error("Get Active Node", "database", "mongo", "msg", err)
		return nil
	}
	err = cur.All(m.context, &sNode)
	if err != nil {
		logger.Error("Get Active Node", "database", "mongo", "msg fetch all", err)
		return nil
	}
	for _, node := range sNode {
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
	//fmt.Println("RemoveActiveNode")
	_, err := m.activeNodeCol.DeleteOne(m.context, bson.M{"_id": enode})
	if err != nil {
		logger.Error("Remove Active Node", "database", "mongo", "msg", err)
	}
}

func (m *MongoDatabase) SaveOtprn(otprn types.Otprn) {
	//fmt.Println("SaveOtprn")
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
	//fmt.Println("GetOtprn")
	otprn, err := RecvOtprn(m.otprnList.FindOne(m.context, bson.M{"_id": otprnHash.String()}))
	if err != nil {
		logger.Error("Get otprn", "database", "mongo", "msg", err)
		return nil
	}
	return otprn
}

func (m *MongoDatabase) SaveLeague(otprnHash common.Hash, enode string) {
	//fmt.Println("SaveLeague")
	nodes := m.GetLeagueList(otprnHash)
	for _, node := range nodes {
		if node.Enode == enode {
			return
		}
	}
	node := new(fntype.HeartBeat)
	err := m.activeNodeCol.FindOne(m.context, bson.M{"_id": enode}).Decode(node)
	if err != nil {
		logger.Error("Save League, Get active node", "database", "mongo", "enode", enode, "msg", err)
	}
	updateOpts := options.Update().SetUpsert(true)
	_, err = m.leagues.UpdateOne(m.context, bson.M{"_id": otprnHash.String()}, bson.M{"$addToSet": bson.M{"nodes": node}}, updateOpts)
	if err != nil {
		logger.Error("Save League update or insert", "database", "mongo", "msg", err)
	}
}

func (m *MongoDatabase) GetLeagueList(otprnHash common.Hash) []types.HeartBeat {
	//fmt.Println("GetLeagueList")
	league := new(fntype.League)
	err := m.leagues.FindOne(m.context, bson.M{"_id": otprnHash.String()}).Decode(league)
	if err != nil {
		if err != mongo.ErrNoDocuments {
			logger.Error("Gat League List", "msg", err)
		}
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
	//fmt.Println("SaveVote")
	voteKey := MakeVoteKey(otprn, blockNum)
	updateOpts := options.Update().SetUpsert(true)
	_, err := m.voteAggregation.UpdateOne(m.context, bson.M{"_id": voteKey.String()}, bson.M{"$addToSet": bson.M{"voters": fntype.Voter{
		Header:   vote.Header,
		Voter:    vote.Voter.String(),
		VoteSign: vote.VoteSign,
	}}}, updateOpts)
	if err != nil {
		logger.Error("Save vote update or insert", "database", "mongo", "msg", err)
	}
}

func (m *MongoDatabase) GetVoters(votekey common.Hash) []*types.Voter {
	//fmt.Println("GetVoters")
	vt := new(fntype.VoteAggregation)
	err := m.voteAggregation.FindOne(m.context, bson.M{"_id": votekey.String()}).Decode(vt)
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
	//fmt.Println("SaveFinalBlock")
	_, err := m.blockChain.InsertOne(m.context, TransBlock(block))
	if err != nil {
		if !IsDup(err) {
			logger.Error("SaveFinalBlock", "insert", "database", "mongo", "msg", err)
			return err
		}
	}
	_, err = m.blockChainRaw.InsertOne(m.context, fntype.RawBlock{
		Hash: block.Hash().String(),
		Raw:  common.Bytes2Hex(byteBlock),
	})
	if err != nil {
		if !IsDup(err) {
			return err
		}
	}

	TransTransaction(block, m.chainID, m.transactions)
	return nil
}

func (m *MongoDatabase) GetBlock(blockHash common.Hash) *types.Block {
	//fmt.Println("GetBlock")
	b := new(fntype.RawBlock)
	err := m.blockChainRaw.FindOne(m.context, bson.M{"_id": blockHash.String()}).Decode(b)
	if err != nil {
		if err != mongo.ErrNoDocuments {
			logger.Error("Get block", "database", "mongo", "msg", err)
		}
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
	//fmt.Println("RemoveBlock")
	_, err := m.blockChain.DeleteOne(m.context, bson.M{"_id": blockHash.String()})
	if err != nil {
		logger.Error("Remove Block", "database", "mongo", "msg", err)
	}
	_, err = m.blockChainRaw.DeleteOne(m.context, bson.M{"_id": blockHash.String()})
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
	//fmt.Println("InsertActiveFairnode")
	_, err := m.activeFairnode.InsertOne(m.context, fntype.Fairnode{ID: nodeKey.String(), Address: address, Status: PENDING.String(), Timestamp: time.Now().Unix()})
	if err != nil {
		if !IsDup(err) {
			logger.Error("Insert Active Fairnode", "database", "mongo", "msg", err)
		}
	}
}

func (m *MongoDatabase) GetActiveFairnodes() map[common.Hash]map[string]string {
	//fmt.Println("GetActiveFairnodes")
	res := make(map[common.Hash]map[string]string)
	var nodes []fntype.Fairnode
	cur, err := m.activeFairnode.Find(m.context, bson.M{})
	if err != nil {
		logger.Error("Get Active Fairnode", "database", "mongo", "msg", err)
		return nil
	}
	err = cur.All(m.context, &nodes)
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
	//fmt.Println("RemoveActiveFairnode")
	_, err := m.activeFairnode.DeleteOne(m.context, bson.M{"_id": nodeKey.String()})
	if err != nil {
		if err != mongo.ErrNoDocuments {
			logger.Error("Remove Active Fairnode", "database", "mongo", "msg", err)
		}
	}
}

func (m *MongoDatabase) UpdateActiveFairnode(nodeKey common.Hash, status uint64) {
	//fmt.Println("UpdateActiveFairnode")
	node := new(fntype.Fairnode)
	node.Status = stType(status).String()
	_, err := m.activeFairnode.UpdateOne(m.context, bson.M{"_id": nodeKey.String()}, bson.M{"$set": bson.M{"status": node.Status}})
	if err != nil {
		logger.Error("Update Active Fairnode", "database", "mongo", "msg", err)
	}
}

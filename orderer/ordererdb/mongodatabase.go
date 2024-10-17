package ordererdb

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"math/big"
	"strings"
	"sync"
)

const DbName = "AnduschainOrderer"

var (
	MongDBConnectError = errors.New("fail to connecting mongo database")
	MongDBPingFail     = errors.New("ping failed")
)

type MongoDatabase struct {
	mu sync.Mutex

	dbName string

	url     string
	chainID *big.Int

	txPool *mongo.Collection

	context context.Context
	client  *mongo.Client
}

type config interface {
	GetInfo() (useSRV bool, host, port, user, pass, ssl, option string, chainID *big.Int)
}

func NewMongoDatabase(conf config) (*MongoDatabase, error) {
	var db MongoDatabase
	var err error
	var protocol, userPass, dbOpt string
	useSRV, host, port, user, pass, ssl, option, chainID := conf.GetInfo()
	// prevent unused
	_ = ssl
	_ = port
	//_ = userPass

	if useSRV {
		protocol = fmt.Sprint("mongodb+srv")
	} else {
		protocol = fmt.Sprint("mongodb")
	}
	if strings.Compare(user, "") != 0 {
		userPass = fmt.Sprintf("%s:%s@", user, pass)
	}
	//if strings.Compare(dbname, "") != 0 {
	//	db.dbName = fmt.Sprintf("%s", dbname)
	//}
	if strings.Compare(option, "") != 0 {
		dbOpt = fmt.Sprintf("?%s", option)
	}
	db.chainID = chainID
	db.dbName = fmt.Sprintf("%s_%s", DbName, chainID.String())

	db.url = "mongodb://localhost:27020,localhost:27021,localhost:27022/AndusChain_91386209?replSet=replication"
	db.url = fmt.Sprintf("%s://%s%s/%s%s", protocol, userPass, host, db.dbName, dbOpt)
	//db.url = fmt.Sprintf("%s://%s/%s%s", protocol, host, db.dbName, dbOpt)

	// 필요시 ApplyURI() 대신 직접 options.Client()을 Set...() 을 수행
	//credential := options.Credential{
	//	Username: user,
	//	Password: pass,
	//}
	log.Println(db.url, user, pass)
	db.client, err = mongo.NewClient(options.Client().ApplyURI(db.url))
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

	m.txPool = m.client.Database(m.dbName).Collection("txPool")

	log.Println("Start orderer mongo database", "chainID", m.chainID.String(), "url", m.url)

	return nil
}

func (m *MongoDatabase) Stop() {
	if err := m.client.Disconnect(m.context); err != nil {
		fmt.Errorf("%v", err)
	}
	fmt.Println("successfully disconnected")
}

func (m *MongoDatabase) InsertTransactionToTxPool(sender string, nonce uint64, hash string, tx []byte) error {
	doc := map[string]interface{}{
		"txhash": hash,
		"from":   sender,
		"nonce":  nonce,
		"tx":     tx,
	}
	_, err := m.txPool.InsertOne(m.context, doc)
	if err != nil {
		log.Println("InsertTransactionToTxPool", "tx", tx, "msg", err)
	}
	return nil
}

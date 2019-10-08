package export

import (
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/fairdb"
	"github.com/anduschain/go-anduschain/fairnode/fairdb/fntype"
	"github.com/anduschain/go-anduschain/rlp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"context"
	log "gopkg.in/inconshreveable/log15.v2"
	"io"
	"strings"
	"time"
)

var (
	logger = log.New("godaon", "chain_export")
	dbName = fairdb.DbName
)

type FairNodeDB struct {
	url           string
	context       context.Context
	client        *mongo.Client
	BlockChain    *mongo.Collection
	BlockChainRaw *mongo.Collection
}

func NewSession(issrv bool, user, pass, host, dbname, dbopt string) (*FairNodeDB, error) {
	var fnb FairNodeDB
	var err error
	var protocol, userPass, dbName, dbOpt string
	if issrv {
		protocol = fmt.Sprint("mongodb+srv")
	} else {
		protocol = fmt.Sprint("mongodb")
	}
	if strings.Compare(user, "") != 0 {
		userPass = fmt.Sprintf("%s:%s@", user, pass)
	}
	if strings.Compare(dbname, "") != 0 {
		dbName = fmt.Sprintf("%s", dbname)
	}
	if strings.Compare(dbopt, "") != 0 {
		dbOpt = fmt.Sprintf("?%s", dbopt)
	}
	fnb.url = fmt.Sprintf("%s://%s%s/%s%s", protocol, userPass, host, dbName, dbOpt)
	fmt.Println("aaaaaaaa", fnb.url)
	fnb.client, err = mongo.NewClient(options.Client().ApplyURI(fnb.url))
	if err != nil {
		return nil, err
	}

	return &fnb, nil
}

func (fnb *FairNodeDB) Start() error {
	fnb.context = context.Background()
	err := fnb.client.Connect(fnb.context)
	if err != nil {
		logger.Error("Mongo DB Connection fail", "error", err)
		return fairdb.MongDBConnectError
	}

	err = fnb.client.Ping(fnb.context, readpref.Primary())
	if err != nil {
		logger.Error("Mongo DB ping fail", "error", err)
		return fairdb.MongDBPingFail
	}

	fnb.BlockChain = fnb.client.Database("Anduschain_91386209").Collection("BlockChain")
	fnb.BlockChainRaw = fnb.client.Database("Anduschain_91386209").Collection("BlockChainRaw")

	logger.Info("connected to database", "url", fnb.url)

	return nil
}

func (fnb *FairNodeDB) Stop() error {
	if err := fnb.client.Disconnect(fnb.context); err != nil {
		fmt.Errorf("%v", err)
		return err
	}
	fmt.Println("successfully disconnected")
	return nil
}

func (fnb *FairNodeDB) Export(w io.Writer) error {
	count, err := fnb.BlockChain.EstimatedDocumentCount(fnb.context,
		options.EstimatedDocumentCount().SetMaxTime(1*time.Second))
	if err != nil {
		return err
	}

	return fnb.ExportN(w, uint64(0), uint64(count))
}

func (fnb *FairNodeDB) ExportN(w io.Writer, first uint64, last uint64) error {
	logger.Info("ExportN", "first", first, "last", last)

	clientOpts := options.Find().
		SetSort(bson.M{"header.number":1})

	cur, err := fnb.BlockChain.Find(fnb.context,
		bson.M{"header.number": bson.M{"$gte": first, "$lte": last}}, clientOpts)
	fmt.Println(len(cur.Current.String()))
	fmt.Println("err : ", err)
	if err != nil {
		return err
	}
	b := new(fntype.Block)
	for cur.Next(fnb.context) {
		cur.Decode(b)
		block, err := fnb.GetRawBlock(b.Hash)
		if err != nil {
			return err
		}
		logger.Info("export block", "number", block.Number(), "hash", block.Hash())
		if err := block.EncodeRLP(w); err != nil {
			return err
		}
	}
	return nil
}

func (fnb *FairNodeDB) GetRawBlock(blockHash string) (*types.Block, error) {
	b := new(fntype.RawBlock)
	err := fnb.BlockChainRaw.FindOne(fnb.context, bson.M{"_id":blockHash}).Decode(b)
	if err != nil {
		logger.Error("Get block", "database", "mongo", "msg", err)
		return nil, err
	}

	blockEnc := common.FromHex(b.Raw)
	block := new(types.Block)
	if err := rlp.DecodeBytes(blockEnc, block); err != nil {
		logger.Error("Get block, decode", "database", "mongo", "msg", err)
		return nil, err
	}
	return block, nil
}

package export

import (
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/fairdb"
	"github.com/anduschain/go-anduschain/fairnode/fairdb/fntype"
	"github.com/anduschain/go-anduschain/rlp"
	log "gopkg.in/inconshreveable/log15.v2"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"strings"
)

var (
	logger = log.New("godaon", "chain_export")
	dbName = fairdb.DbName
)

type FairNodeDB struct {
	url           string
	Mongo         *mgo.Session
	BlockChain    *mgo.Collection
	BlockChainRaw *mgo.Collection
}

func NewSession(host, port, user, pwd string) (*FairNodeDB, error) {
	var fnb FairNodeDB
	if strings.Compare(user, "") != 0 {
		fnb.url = fmt.Sprintf("mongodb://%s:%s@%s:%s/%s", user, pwd, host, port, dbName)
	} else {
		fnb.url = fmt.Sprintf("mongodb://%s:%s", host, port)
	}

	return &fnb, nil
}

func (fnb *FairNodeDB) Start() error {

	session, err := mgo.Dial(fnb.url)
	if err != nil {
		logger.Error("Mongo DB Dial", "error", err)
		return fairdb.MongDBConnectError
	}

	session.SetMode(mgo.Monotonic, true)

	fnb.Mongo = session
	fnb.BlockChain = session.DB(dbName).C("BlockChain")
	fnb.BlockChainRaw = session.DB(dbName).C("BlockChainRaw")

	logger.Info("connected to database", "url", fnb.url)

	return nil
}

func (fnb *FairNodeDB) Stop() error {
	fnb.Mongo.Close()
	return nil
}

func (fnb *FairNodeDB) Export(w io.Writer) error {
	count, err := fnb.BlockChain.Find(nil).Count()
	if err != nil {
		return err
	}
	return fnb.ExportN(w, uint64(0), uint64(count))
}

func (fnb *FairNodeDB) ExportN(w io.Writer, first uint64, last uint64) error {
	logger.Info("ExportN", "first", first, "last", last)
	iter := fnb.BlockChain.Find(bson.M{"header.number": bson.M{"$gte": first, "$lte": last}}).Sort("header.number").Iter()
	b := new(fntype.Block)
	for iter.Next(&b) {
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
	err := fnb.BlockChainRaw.FindId(blockHash).One(b)
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

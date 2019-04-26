package export

import (
	"fmt"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/fairnode/server/db"
	log "gopkg.in/inconshreveable/log15.v2"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"strings"
)

type FairNodeDB struct {
	url           string
	Mongo         *mgo.Session
	BlockChain    *mgo.Collection
	BlockChainRaw *mgo.Collection
	logger        log.Logger
}

func NewSession(host, port, user, pwd string) (*FairNodeDB, error) {
	var fnb FairNodeDB
	fnb.logger = log.New("fairnode", "chain export")
	if strings.Compare(user, "") != 0 {
		fnb.url = fmt.Sprintf("mongodb://%s:%s@%s:%s/%s", user, pwd, host, port, db.DBNAME)
	} else {
		fnb.url = fmt.Sprintf("mongodb://%s:%s", host, port)
	}

	return &fnb, nil
}

func (fnb *FairNodeDB) Start() error {

	session, err := mgo.Dial(fnb.url)
	if err != nil {
		fnb.logger.Error("Mongo DB Dial", "error", err)
		return db.MongDBConnectError
	}

	session.SetMode(mgo.Monotonic, true)

	fnb.Mongo = session
	fnb.BlockChain = session.DB(db.DBNAME).C("BlockChain")
	fnb.BlockChainRaw = session.DB(db.DBNAME).C("BlockChainRaw")

	fnb.logger.Info("connected to database", "url", fnb.url)

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
	fnb.logger.Info("ExportN", "first", first, "last", last)
	iter := fnb.BlockChain.Find(nil).Sort("header.number").Limit(int(last)).Iter()
	var sBlock db.StoredBlock
	for iter.Next(&sBlock) {
		block, err := fnb.GetRawBlock(sBlock.BlockHash)
		if err != nil {
			return err
		}
		fnb.logger.Info("export block", "number", block.Number(), "hash", block.Hash())
		if err := block.EncodeRLP(w); err != nil {
			return err
		}
	}
	return nil
}

func (fnb *FairNodeDB) GetRawBlock(blockHash string) (*types.Block, error) {
	var res []db.StoreFinalBlockRaw
	var b []byte
	err := fnb.BlockChainRaw.Find(bson.M{"blockhash": blockHash}).Sort("order").All(&res)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(res); i++ {
		b = append(b, res[i].Raw...)
	}

	block := fairtypes.DecodeBlock(b)
	return block, nil
}

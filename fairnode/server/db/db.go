package db

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/fairnode/fairutil"
	"github.com/anduschain/go-anduschain/fairnode/server/config"
	log "gopkg.in/inconshreveable/log15.v2"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"math/big"
	"net"
	"strings"
	"time"
)

const DBNAME = "AndusChain"

type FairNodeDB struct {
	url      string
	dialInfo *mgo.DialInfo
	Mongo    *mgo.Session

	ChainConfig   *mgo.Collection
	ActiveNodeCol *mgo.Collection
	MinerNode     *mgo.Collection
	OtprnList     *mgo.Collection
	BlockChain    *mgo.Collection
	BlockChainRaw *mgo.Collection

	Transactions *mgo.Collection

	signer types.Signer
	logger log.Logger
	config *config.Config
}

var (
	MongDBConnectError = errors.New("MongoDB 접속에 문제가 있습니다")
)

// Mongodb url => mongodb://[username:password@]host1[:port1][,host2[:port2],...[,hostN[:portN]]][/[database][?options]]
func New(signer types.Signer) (*FairNodeDB, error) {
	var fnb FairNodeDB

	fnb.logger = log.New("fairnode", "mongodb")
	fnb.config = config.DefaultConfig

	if strings.Compare(fnb.config.DBuser, "") != 0 {
		fnb.url = fmt.Sprintf("mongodb://%s:%s@%s:%s/%s", fnb.config.DBuser, fnb.config.DBpass, fnb.config.DBhost, fnb.config.DBport, DBNAME)
	} else {
		fnb.url = fmt.Sprintf("mongodb://%s:%s/%s", fnb.config.DBhost, fnb.config.DBport, DBNAME)
	}

	// SSL db connection config
	if strings.Compare(fnb.config.SSL_path, "") != 0 {
		tlsConfig := &tls.Config{}
		tlsConfig.InsecureSkipVerify = true

		roots := x509.NewCertPool()
		if ca, err := ioutil.ReadFile(fnb.config.SSL_path); err == nil {
			roots.AppendCertsFromPEM(ca)
		}

		tlsConfig.RootCAs = roots

		dialInfo, err := mgo.ParseURL(fnb.url)
		if err != nil {
			return nil, err
		}

		dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
			conn, err := tls.Dial("tcp", addr.String(), tlsConfig)
			return conn, err
		}

		fnb.dialInfo = dialInfo
	}

	fnb.signer = signer

	return &fnb, nil
}

func (fnb *FairNodeDB) Start() error {
	var session *mgo.Session
	var err error

	if fnb.dialInfo == nil {
		session, err = mgo.Dial(fnb.url)
	} else {
		session, err = mgo.DialWithInfo(fnb.dialInfo)
	}

	if err != nil {
		fnb.logger.Error("Mongo DB Dial", "error", err)
		return MongDBConnectError
	}

	session.SetMode(mgo.Monotonic, true)

	fnb.Mongo = session
	fnb.ChainConfig = session.DB(DBNAME).C("ChainConfig")
	fnb.ActiveNodeCol = session.DB(DBNAME).C("ActiveNode")
	fnb.MinerNode = session.DB(DBNAME).C("MinerNode")
	fnb.OtprnList = session.DB(DBNAME).C("OtprnList")
	fnb.BlockChain = session.DB(DBNAME).C("BlockChain")
	fnb.BlockChainRaw = session.DB(DBNAME).C("BlockChainRaw")
	fnb.Transactions = session.DB(DBNAME).C("Transactions")

	return nil
}

func (fnb *FairNodeDB) Stop() error {
	fnb.Mongo.Close()
	return nil
}

func (fnb *FairNodeDB) SetChainConfig() {
	cnt, err := fnb.ChainConfig.Find(nil).Count()
	if err != nil || cnt == 0 {
		err := fnb.ChainConfig.Insert(&ChainConfig{Miner: fnb.config.Miner, Epoch: fnb.config.Epoch, Fee: fnb.config.Fee, Version: fnb.config.GethVersion})
		if err != nil {
			fnb.logger.Warn("SetChainConfig ", "error", err)
		}
	}
	fnb.logger.Debug("SetChainConfig", "Miner", fnb.config.Miner, "Epoch", fnb.config.Epoch, "Fee", fnb.config.Fee, "Version", fnb.config.GethVersion)
}

func (fnb *FairNodeDB) GetChainConfig() *ChainConfig {
	cfg := &ChainConfig{}
	err := fnb.ChainConfig.Find(nil).One(&cfg)
	if err != nil {
		fnb.logger.Warn("GetChainConfig ", "error", err)
		// 디비에 값이 조회되지 않을경우
		cfg.Miner = fnb.config.Miner
		cfg.Epoch = fnb.config.Epoch
		cfg.Fee = fnb.config.Fee
		cfg.Version = fnb.config.GethVersion
	}
	fnb.config.SetMiningConf(cfg.Miner, cfg.Epoch, cfg.Fee, cfg.Version)
	fnb.logger.Debug("GetChainConfig", "Miner", cfg.Miner, "Epoch", cfg.Epoch, "Fee", cfg.Fee, "Version", cfg.Version)
	return cfg
}

func (fnb *FairNodeDB) SaveActiveNode(enode string, coinbase common.Address, clientport, ip, version string, chaiID uint64) {
	trial := net.ParseIP(ip)
	if trial.To4() == nil {
		fnb.logger.Warn("to4 nil")
		return
	}
	if strings.Compare(version, fnb.config.GethVersion) == 0 && fnb.config.ChainID == chaiID {
		tmp := activeNode{EnodeId: enode, Coinbase: coinbase.Hex(), Ip: trial.To4().String(), Time: time.Now(), Port: clientport, Version: version}
		if _, err := fnb.ActiveNodeCol.UpsertId(tmp.EnodeId, bson.M{"$set": tmp}); err != nil {
			fnb.logger.Warn("SaveActiveNode ", "error", err)
		}
	} else {
		fnb.logger.Error("SaveActiveNode", "msg", "노드 정보가 다르다", "Geth", version, "DB", fnb.config.GethVersion, "ChainID", chaiID)
		return
	}
}

func (fnb *FairNodeDB) GetActiveNodeNum() int {

	num, err := fnb.ActiveNodeCol.Find(nil).Count()
	if err != nil {
		fnb.logger.Warn("GetActiveNodeNum", "error", err)
	}
	// TODO : andus >> DB에서 Active node 갯수 조회
	fnb.logger.Debug("GetActiveNodeNum", "nodeCount", num)

	return num
}

func (fnb *FairNodeDB) GetActiveNodeList() []activeNode {
	var actlist []activeNode
	// andus >> 모든 activenode 리스트 전달.
	fnb.ActiveNodeCol.Find(nil).All(&actlist)
	return actlist
}

func (fnb *FairNodeDB) JobCheckActiveNode() {
	var activelist []activeNode
	now := time.Now()
	fnb.ActiveNodeCol.Find(nil).All(&activelist)
	for index := range activelist {
		if now.Sub(activelist[index].Time) >= (3 * time.Minute) {
			err := fnb.ActiveNodeCol.RemoveId(activelist[index].EnodeId)
			if err != nil {
				fnb.logger.Warn("Remove enode", "error", err)
			}
		}
	}
}

func (fnb *FairNodeDB) CheckEnodeAndCoinbse(enodeId string, coinbase string) bool {
	// TODO : andus >> 1. Enode가 맞는지 확인 ( 조회 되지 않으면 팅김 )
	// TODO : andus >> 2. 해당하는 Enode가 이전에 보낸 코인베이스와 일치하는지
	var actnode activeNode
	err := fnb.ActiveNodeCol.FindId(enodeId).One(&actnode)
	if err != nil {
		fnb.logger.Warn("CheckEnodeAndCoinbse find one", "error", err)
	}
	if actnode.EnodeId == "" {
		return false
	} else {
		if actnode.Coinbase != coinbase {
			return false
		}
	}
	return true
}

func (fnb *FairNodeDB) SaveMinerNode(otprnHash string, enode string) {
	// TODO : andus >> 실제 TCP에 접속한 채굴마이너를 저장
	m := minerNode{Otprnhash: otprnHash, Nodes: []string{enode}, Timestamp: time.Now()}
	_, err := fnb.MinerNode.UpsertId(m.Otprnhash, bson.M{"$push": bson.M{"nodes": enode}, "$set": bson.M{"timestamp": m.Timestamp}})
	if err != nil {
		fnb.logger.Warn("MinerNodeInsert", "error", err)
	}
}

func (fnb *FairNodeDB) SaveOtprn(tsotprn fairtypes.TransferOtprn) {
	err := fnb.OtprnList.Insert(&saveotprn{OtprnHash: tsotprn.Hash.String(), TsOtprn: tsotprn})
	if err != nil {
		fnb.logger.Warn("saveotprn", "error", err)
	}
}

func (fnb *FairNodeDB) GetMinerNode(otprnHash string) []string {
	var minerlist minerNode
	err := fnb.MinerNode.FindId(otprnHash).One(&minerlist)
	if err != nil {
		fnb.logger.Warn("GetMinerNode", "error", err)
	}

	return minerlist.Nodes
}

func (fnb *FairNodeDB) GetMinerNodeNum(otprnHash string) uint64 {
	var minerlist minerNode
	err := fnb.MinerNode.FindId(otprnHash).One(&minerlist)
	if err != nil {
		fnb.logger.Warn("GetMinerNodeNum", "error", err)
		return 0
	}
	return uint64(len(minerlist.Nodes))
}

func (fnb *FairNodeDB) GetCurrentBlock() *big.Int {

	var sBlock *StoredBlock

	err := fnb.BlockChain.Find(bson.M{}).Sort("-header.number").Limit(1).One(&sBlock)
	if err != nil {
		fnb.logger.Warn("GetCurrentBlock", "error", err)
	}

	if sBlock == nil {
		return big.NewInt(0)
	}

	return big.NewInt(sBlock.Header.Number)
}

func (fnb *FairNodeDB) SaveFianlBlock(block *types.Block) {
	header := header{
		ParentHash: block.ParentHash().String(),
		Coinbase:   block.Coinbase().String(),
		Root:       block.Root().String(),

		VoteHash: block.VoterHash().String(),

		TxHash:     block.Header().TxHash.String(),
		JoinTxHash: block.Header().JoinTxHash.String(),

		ReceiptHash:     block.Header().ReceiptHash.String(),
		JoinReceiptHash: block.Header().JoinReceiptHash.String(),

		Difficulty: block.Header().Difficulty.String(),
		Number:     block.Header().Number.Int64(),
		GasLimit:   int64(block.Header().GasLimit),
		GasUsed:    int64(block.Header().GasUsed),
		Time:       block.Header().Time.String(),
		Extra:      block.Header().Extra,
		Nonce:      int64(block.Nonce()),

		FairnodeSig: block.FairNodeSig(),
	}

	// General transaction
	var txs []transaction
	for i := range block.Transactions() {
		tx := block.Transactions()[i]
		txhash := block.Transactions()[i].Hash()
		from, _ := types.Sender(fnb.signer, tx)
		to := "contract"
		if tx.To() != nil {
			to = tx.To().String()
		}

		txs = append(txs, transaction{
			Txhash:       txhash.String(),
			From:         from.String(),
			To:           to,
			AccountNonce: int64(tx.Nonce()),
			Price:        tx.GasPrice().String(),
			Amount:       tx.Value().String(),
			Payload:      tx.Data(),
		})
	}

	// Join transaction
	var joinTxs []transaction
	for i := range block.JoinTransactions() {
		joinTx := block.JoinTransactions()[i]
		joinTxhash := block.JoinTransactions()[i].Hash()
		from, _ := types.Sender(fnb.signer, joinTx)
		to := "contract"
		if joinTx.To() != nil {
			to = joinTx.To().String()
		}

		txs = append(txs, transaction{
			Txhash:       joinTxhash.String(),
			From:         from.String(),
			To:           to,
			AccountNonce: int64(joinTx.Nonce()),
			Price:        joinTx.GasPrice().String(),
			Amount:       joinTx.Value().String(),
			Payload:      joinTx.Data(),
		})
	}

	var voters []vote
	for i := range block.Voters() {
		voter := block.Voters()[i]
		voters = append(voters, vote{voter.Addr.String(), voter.Sig, voter.Difficulty})
	}

	b := StoredBlock{
		BlockHash:        block.Hash().String(),
		Header:           header,
		GenTransactions:  txs,
		JoinTransactions: joinTxs,
		Voters:           voters,
	}

	if err := fnb.SaveRawBlock(block); err != nil {
		fnb.logger.Warn("SaveRawBlock", "error", err)
	}

	if err := fnb.BlockChain.Insert(b); err != nil {
		fnb.logger.Warn("SaveFianlBlock", "error", err)
	}
}

// 바이트로 블록 저장 size 100000 * 1024
func (fnb *FairNodeDB) SaveRawBlock(block *types.Block) error {
	SPLIT_BYTE := 10 * 1024 * 1024 // 10MB
	blockByte := fairtypes.EncodeBlock(block)
	raws := fairutil.ByteTrimSize(blockByte, SPLIT_BYTE)

	if len(raws) < 1 {
		return errors.New("Raw block make error")
	}

	for i := 0; i < len(raws); i++ {
		raw := StoreFinalBlockRaw{block.Hash().String(), int64(i), int64(len(raws[i])), raws[i]}
		if err := fnb.BlockChainRaw.Insert(raw); err != nil {
			return err
		}

		fnb.logger.Debug("Save Block raw", "hash", block.Hash(), "order", i, "size", len(raws[i]))
	}

	return nil
}

func (fnb *FairNodeDB) GetRawBlock(blockHash string) (*types.Block, error) {
	var res []StoreFinalBlockRaw
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

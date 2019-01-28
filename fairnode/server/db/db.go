package db

import (
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"math/big"
	"net"
	"time"
)

const DBNAME = "AndusChain"

type FairNodeDB struct {
	url           string
	Mongo         *mgo.Session
	ActiveNodeCol *mgo.Collection
	MinerNode     *mgo.Collection
	OtprnList     *mgo.Collection
	BlockChain    *mgo.Collection
	signer        types.Signer
}

var (
	MongDBConnectError = errors.New("MongoDB 접속에 문제가 있습니다")
)

// Mongodb url => mongodb://[username:password@]host1[:port1][,host2[:port2],...[,hostN[:portN]]][/[database][?options]]
func New(dbhost string, dbport string, pwd string, user string, signer types.Signer) (*FairNodeDB, error) {
	var fnb FairNodeDB

	if user != "" {
		fnb.url = fmt.Sprintf("mongodb://%s:%s@%s:%s/%s", user, pwd, dbhost, dbport, DBNAME)
	} else {
		fnb.url = fmt.Sprintf("mongodb://%s:%s", dbhost, dbport)
	}

	fnb.signer = signer

	return &fnb, nil
}

func (fnb *FairNodeDB) Start() error {

	session, err := mgo.Dial(fnb.url)
	if err != nil {
		fmt.Println(err)
		return MongDBConnectError
	}

	session.SetMode(mgo.Monotonic, true)

	fnb.Mongo = session
	fnb.ActiveNodeCol = session.DB(DBNAME).C("ActiveNode")
	fnb.MinerNode = session.DB(DBNAME).C("MinerNode")
	fnb.OtprnList = session.DB(DBNAME).C("OtprnList")
	fnb.BlockChain = session.DB(DBNAME).C("BlockChain")

	return nil
}

func (fnb *FairNodeDB) Stop() error {
	fnb.Mongo.Close()
	return nil
}

func (fnb *FairNodeDB) SaveActiveNode(enode string, coinbase common.Address, clientport string, ip string) {
	trial := net.ParseIP(ip)
	if trial.To4() == nil {
		return
	}

	tmp := activeNode{EnodeId: enode, Coinbase: coinbase.Hex(), Ip: trial.To4().String(), Time: time.Now(), Port: clientport}

	if _, err := fnb.ActiveNodeCol.UpsertId(tmp.EnodeId, bson.M{"$set": tmp}); err != nil {
		log.Println("Error[DB] : SaveActiveNode ", err)
	}
}

func (fnb *FairNodeDB) GetActiveNodeNum() int {

	num, err := fnb.ActiveNodeCol.Find(nil).Count()
	if err != nil {
		log.Println("Error[DB] : GetActiveNodeNum err : ", err)
	}
	// TODO : andus >> DB에서 Active node 갯수 조회
	log.Println("Info[DB] :Db.GetActiveNodeNum -> ", num)

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
				log.Println("Error[DB] : Remove enode err : ", err)
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
		log.Println("Error[DB] : CheckEnodeAndCoinbse find one err : ", err)
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
		log.Println("Error[DB] : MinerNodeInsert err : ", err)
	}
}

func (fnb *FairNodeDB) SaveOtprn(tsotprn fairtypes.TransferOtprn) {
	err := fnb.OtprnList.Insert(&saveotprn{OtprnHash: tsotprn.Hash.String(), TsOtprn: tsotprn})
	if err != nil {
		log.Println("Error[DB] : saveotprn err : ", err)
	}
}

func (fnb *FairNodeDB) GetMinerNode(otprnHash string) []string {
	var minerlist minerNode
	err := fnb.MinerNode.FindId(otprnHash).One(&minerlist)
	if err != nil {
		log.Println("Error[DB] : GetMinerNode", err)
	}

	return minerlist.Nodes
}

func (fnb *FairNodeDB) GetMinerNodeNum(otprnHash string) uint64 {
	var minerlist minerNode
	err := fnb.MinerNode.FindId(otprnHash).One(&minerlist)
	if err != nil {
		log.Println("Error[DB] : GetMinerNodeNum", err)
		return 0
	}
	return uint64(len(minerlist.Nodes))
}

func (fnb *FairNodeDB) GetCurrentBlock() *big.Int {

	var sBlock *storedBlock

	err := fnb.BlockChain.Find(bson.M{}).Sort("-header.number").Limit(1).One(&sBlock)
	if err != nil {
		log.Println("Error[DB] : GetCurrentBlock", err)
	}

	if sBlock == nil {
		return big.NewInt(0)
	}

	return big.NewInt(sBlock.Header.Number)
}

func (fnb *FairNodeDB) SaveFianlBlock(block *types.Block) {
	header := header{
		block.Header().ParentHash.String(),
		block.Header().UncleHash.String(),
		block.Header().Coinbase.String(),
		block.Header().Root.String(),
		block.Header().TxHash.String(),
		block.Header().ReceiptHash.String(),
		block.Header().Difficulty.String(),
		block.Header().Number.Int64(),
		int64(block.Header().GasLimit),
		int64(block.Header().GasUsed),
		block.Header().Time.String(),
		block.Header().Extra,
		block.Header().MixDigest.String(),
		int64(block.Header().Nonce.Uint64()),
	}

	var txs []transaction
	for i := range block.Transactions() {
		tx := block.Transactions()[i]
		from, _ := types.Sender(fnb.signer, tx)
		txs = append(txs, transaction{
			From:         from.String(),
			To:           tx.To().String(),
			AccountNonce: int64(tx.Nonce()),
			Price:        tx.GasPrice().Int64(),
			Amount:       tx.Value().Int64(),
			Payload:      tx.Data(),
		})
	}

	var voter []vote
	for i := range block.Voter {
		voter = append(voter, vote{block.Voter[i].Addr.String(), block.Voter[i].Sig})
	}

	b := storedBlock{
		header,
		txs,
		common.BytesToHash(block.FairNodeSig).String(),
		voter,
	}

	err := fnb.BlockChain.Insert(b)
	if err != nil {
		log.Println("Error[DB] : SaveFianlBlock", err)
	}

}

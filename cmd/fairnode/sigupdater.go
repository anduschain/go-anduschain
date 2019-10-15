package main

import (
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/crypto/sha3"
	"github.com/anduschain/go-anduschain/fairnode"
	"github.com/anduschain/go-anduschain/fairnode/fairdb"
	"github.com/anduschain/go-anduschain/rlp"
	log "gopkg.in/inconshreveable/log15.v2"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

//type MongoDatabase struct {
//	context context.Context
//	client  *mongo.Client
//
//	url string
//
//	blockChain    *mongo.Collection
//	blockChainRaw *mongo.Collection
//}

type MongoBlock struct {
	Hash   string `json:"blockHash" bson:"_id,omitempty"`
	Header struct {
		Number       int    `json:"number" bson:"number"`
		FairnodeSign []byte `json:"fairnodeSign" bson:"fairnodeSign"`
	}
	Raw string `json:"rowBlock" bson:"rowBlock"`
}

func updateSignature(ctx *cli.Context) error {
	keyfilePath := ctx.String("keypath")
	keyfile := filepath.Join(keyfilePath, "fairkey.json")
	if _, err := os.Stat(keyfile); err != nil {
		log.Error("Keyfile not exists", "path", keyfilePath)
		return err
	}

	var err error

	fmt.Println("Input fairnode keystore password")
	passphrase := promptPassphrase(false)
	pk, err := fairnode.GetPriveKey(keyfilePath, passphrase)
	if err != nil {
		return err
	}
	_ = pk

	fmt.Println("Input fairnode database password")
	dbpass := promptPassphrase(false)

	/*
		1. Connect() : 내부적으로 client 생성 db 연결
		2. 업데이트 해야 할 fairnodeSign binary 값 검색
		3. 검색 결과 검증 ( 카운트, 실제 비어 있는지, 해당 블록 넘버, 블록 넘버를 파일로 저장해두면 나중에 검증이 용이하다.)
		4. 업데이트 결과 재생성
		5. 실제 업데이트
		6. 확인
		6.1 잘못된 경우 삭제 필요.
	*/

	time.Sleep(2 * time.Second)

	conf := &dbConfig{
		host:    "cluster0-g9ryp.mongodb.net/",
		port:    "",
		user:    "anduschain",
		pass:    dbpass,
		ssl:     "",
		chainID: big.NewInt(14288641),
		option:  "?retryWrites=true&w=majority",
	}

	db, err := fairdb.NewMongoDatabase(conf)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = db.Start()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer db.Stop()

	if err := SaveBlockUsingRLPForm(db); err != nil {
		log.Crit(err.Error())
	} else {
		log.Info("end SaveBlockUsingRLPForm")
	}

	return nil
	//
	//if _, err := db.SearchBinData(pk); err != nil {
	//	log.Crit(err.Error())
	//} else {
	//	log.Info("end updateSignature")
	//}
	//
	//if err := db.VerifyFairnodeSign(); err != nil {
	//	log.Crit(err.Error())
	//} else {
	//	log.Info("end VerifyFairnodeSign")
	//}

	return nil
}

var (
	// local
	//Protocol = "mongodb"
	Userpass = ""
	//Host = "localhost:27020,localhost:27021,localhost:27022/"
	//Dbname = "Anduschain_91386209"
	//Dboption = "?replSet=replication"

	// atlas
	Protocol = "mongodb+srv"
	Host = "cluster0-g9ryp.mongodb.net/"
	Dbname = "Anduschain_14288641"
	Dboption = "?retryWrites=true&w=majority"
)

// mongodb 정합성이 꺠졌을 때,
// node로부터 rlp문자열을 받아 db로 저장하는 함수
func SaveBlockUsingRLPForm(db *fairdb.MongoDatabase) error {
	buf, err := ioutil.ReadFile("/home/jhp/Downloads/Untitled")
	if err != nil {
		return fmt.Errorf("error read file %v", err)
	}
	//fmt.Println(string(buf))
	//fmt.Println(common.Bytes2Hex(buf))
	blockEnc := common.FromHex(string(buf))
	//block := new(types.Block)
	var block types.Block
	if err := rlp.DecodeBytes(blockEnc, &block); err != nil {
		fmt.Println("decode error: ", err)
		return err
	}
	//
	//if err := rlp.DecodeBytes(buf, &block); err != nil {
	//	fmt.Println("decode error: ", err)
	//	return err
	//}

	fmt.Println(block.Hash().String())
	fmt.Println(block.Number())

	db.SaveFinalBlock(&block, blockEnc)

	return nil
}


// BlockChain Collection에서 fairnodeSign이 빠진, blockhash를 전달받는다.
// block hash를 이용해 BlockChainRaw의 해당 도큐먼트를 디코드하고,
// signature block hash + voter sign 시그니처를 생성
// 해당 signature를 BlockChain Collection에 업데이트
// 해당 signature를 포함한 데이터를 인코딩해 BlockChainRaw에 업데이트
//func SearchBinData(db *fairdb.MongoDatabase, pk *ecdsa.PrivateKey) (string, error) {
//	fmt.Println("SearchBinData")
//	return "", nil
//	var empty primitive.Binary
//	//db.getCollection('BlockChain').find({"header.fairnodeSign":{$eq:BinData(0,"")}})
//	cur, err := m.blockChain.Find(m.context, bson.M{"header.fairnodeSign": empty})
//	if err != nil {
//		if err != mongo.ErrNoDocuments {
//			return "", err
//		} else {
//			fmt.Println("no documents")
//			os.Exit(1)
//		}
//	}
//
//	var set MongoBlock
//	for cur.Next(m.context) {
//		cur.Decode(&set)
//		m.SaveRawBlock(&set, pk)
//	}
//	fmt.Println("end")
//
//	return "filename", err
//}
//
//func (m *MongoDatabase) SaveRawBlock(set *MongoBlock, pk *ecdsa.PrivateKey) error {
//	fmt.Println("SaveRawBlock")
//
//	fmt.Println("hash :", set.Hash)
//	err := m.blockChainRaw.FindOne(m.context, bson.M{"_id": set.Hash}).Decode(&set)
//	if err != nil {
//		if err != mongo.ErrNoDocuments {
//			fmt.Println("errrrrrr", err)
//			return err
//		} else {
//			fmt.Println("no documents")
//		}
//	}
//
//	blockEnc := common.FromHex(set.Raw)
//	block := new(types.Block)
//	if err := rlp.DecodeBytes(blockEnc, block); err != nil {
//		fmt.Println("decode error: ", err)
//		return err
//	}
//
//	fmt.Println("signature(raw) :", common.Bytes2Hex(block.Header().FairnodeSign))
//	hash := rlpHash([]interface{}{
//		block.Hash(),
//		block.VoterHash(),
//	})
//	signature, err := crypto.Sign(hash.Bytes(), pk)
//	if err != nil {
//		log.Crit(fmt.Sprintf("config signature error msg = %s", err.Error()))
//		return nil
//	}
//	fmt.Println("signature(new) :", common.Bytes2Hex(signature))
//
//	// signature update (blockchain)
//	_, err = m.blockChain.UpdateOne(m.context, bson.M{"_id": set.Hash}, bson.M{"$set": bson.M{"header.fairnodeSign": signature}})
//	if err != nil {
//		fmt.Println(err)
//	}
//
//	//blockchainraw encoding
//	block = block.WithFairnodeSign(signature)
//	var en bytes.Buffer
//	err = block.EncodeRLP(&en)
//	if err != nil {
//		fmt.Println("Request Fairnode block EncodeRLP", "msg", err)
//	}
//	// block update (blockchainraw)
//	_, err = m.blockChainRaw.UpdateOne(m.context, bson.M{"_id": set.Hash}, bson.M{"$set": bson.M{"rowBlock": common.Bytes2Hex(en.Bytes())}})
//	if err != nil {
//		fmt.Println(err)
//	}
//	return nil
//}
//
//func (m *MongoDatabase) VerifyFairnodeSign() error {
//	//db.getCollection('BlockChain').find({"header.fairnodeSign":{$eq:BinData(0,"")}})
//	var mb MongoBlock
//	findOpts := options.Find().
//		SetLimit(10)
//	cur, err := m.blockChain.Find(m.context, bson.M{"header.number": bson.M{"$gt" : 5000, "$lt":30000 }}, findOpts)
//	if err != nil {
//		if err != mongo.ErrNoDocuments {
//			return err
//		} else {
//			fmt.Println("no documents")
//			os.Exit(1)
//		}
//	}
//	for cur.Next(m.context) {
//		cur.Decode(&mb)
//		fmt.Println("num :", mb.Header.Number)
//		fmt.Println("blk :", common.Bytes2Hex(mb.Header.FairnodeSign))
//		m.CompareWithRawBlock(mb.Hash)
//	}
//
//	return nil
//}
//
//func (m *MongoDatabase) CompareWithRawBlock(hash string) error {
//	var raw MongoBlock
//	err := m.blockChainRaw.FindOne(m.context, bson.M{"_id": hash}).Decode(&raw)
//	if err != nil {
//		if err != mongo.ErrNoDocuments {
//			fmt.Println("errrrrrr", err)
//			return err
//		} else {
//			fmt.Println("no documents")
//		}
//	}
//	blockEnc := common.FromHex(raw.Raw)
//	block := new(types.Block)
//	if err := rlp.DecodeBytes(blockEnc, block); err != nil {
//		fmt.Println("decode error: ", err)
//		return err
//	}
//	fmt.Println("raw :", common.Bytes2Hex(block.Header().FairnodeSign))
//	return nil
//}

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

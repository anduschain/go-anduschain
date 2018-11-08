package fairnodeclient

// TODO : andus >> Geth - FairNode 사이에 연결 되는 부분..

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/anduschain/go-anduschain/accounts/keystore"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/fairnode/fairutil"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/p2p/discv5"
	"github.com/anduschain/go-anduschain/rlp"
	"log"
	"math/big"
	"net"
	"sync"
	"time"
)

type DebMiner interface {
	StartMining(threads int) error
	StopMining()
	IsMining() bool
}

const (
	FAIRNODE_CON_INFO = "127.0.0.1:50900"
)

type FairnodeClient struct {
	Otprn *otprn.Otprn
	//OtprnCh chan *otprn.Otprn
	WinningBlockCh chan *types.TransferBlock // TODO : andus >> worker의 위닝 블록을 받는 채널... -> Fairnode에게 쏜다
	FinalBlockCh   chan *types.TransferBlock
	Running        bool
	wg             sync.WaitGroup
	BlockChain     *core.BlockChain
	Miner          DebMiner
	Coinbase       *common.Address
	keystore       *keystore.KeyStore
	txPool         *core.TxPool
	PrivateKey     *ecdsa.PrivateKey
	SAddrUDP       *net.UDPAddr
	LAddrUDP       *net.UDPAddr
	TcpConnStartCh chan struct{}
}

func New(wbCh chan *types.TransferBlock, fbCh chan *types.TransferBlock, blockChain *core.BlockChain, miner DebMiner, coinbase *common.Address, ks *keystore.KeyStore, tp *core.TxPool) *FairnodeClient {

	serverAddr, err := net.ResolveUDPAddr("udp", FAIRNODE_CON_INFO)
	if err != nil {
		log.Fatal("andus >> UDPtoFairNode, ServerAddr", err)
	}

	localAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1")
	if err != nil {
		log.Fatal("andus >> UDPtoFairNode, LocalAddr", err)
	}

	fcClient := &FairnodeClient{
		Otprn:          nil,
		WinningBlockCh: wbCh,
		FinalBlockCh:   fbCh,
		Running:        false,
		BlockChain:     blockChain,
		Miner:          miner,
		Coinbase:       coinbase,
		keystore:       ks,
		txPool:         tp,
		SAddrUDP:       serverAddr,
		LAddrUDP:       localAddr,
		TcpConnStartCh: make(chan struct{}),
	}

	return fcClient
}

//TODO : andus >> fairNode 관련 함수....
func (fc *FairnodeClient) StartToFairNode() error {
	fc.Running = true

	unlockedKey, err := fc.keystore.GetUnlockedPrivKey(*fc.Coinbase)
	if err != nil {
		log.Println("andus >>", err)
	}

	fc.PrivateKey = unlockedKey

	// TODO : andus >> 마이닝 켜저 있으면 종료

	if fc.Miner.IsMining() {
		fc.Miner.StopMining()
	}

	// udp
	go fc.UDPtoFairNode()

	// tcp
	go fc.TCPtoFairNode()

	fc.wg.Wait()

	return nil
}
func (fc *FairnodeClient) UDPtoFairNode() {
	//TODO : andus >> udp 통신 to FairNode
	go fc.submitEnode()
	go fc.receiveOtprn()
}

func (fc *FairnodeClient) submitEnode() {
	// TODO : andus >> FairNode IP : localhost UDP Listener 11/06 -- start --
	Conn, err := net.DialUDP("udp", fc.LAddrUDP, fc.SAddrUDP)
	if err != nil {
		log.Fatal("andus >> UDPtoFairNode, DialUDP", err)
	}

	defer Conn.Close()

	// TODO : andus >> FairNode IP : localhost UDP Listener 11/06 -- end --
	t := time.NewTicker(60 * time.Second)

	nodeUrl := discv5.NewNode(
		discv5.PubkeyID(&ecdsa.PublicKey{fc.PrivateKey.PublicKey.Curve, fc.PrivateKey.X, fc.PrivateKey.Y}),
		fc.LAddrUDP.IP,
		uint16(fc.LAddrUDP.Port),
		0,
	)

	enode := nodeUrl.String()                  // TODO : andus >> enode
	enodeByte, err := rlp.EncodeToBytes(enode) // TODO : andus >> enode to byte
	log.Println("andus >> enode >>>", enode)
	if err != nil {
		log.Fatal("andus >> EncodeToBytes", err)
	}

	for {
		select {
		case <-t.C:
			//TODO : andus >> FairNode에게 enode값 전송 ( 1분단위)
			// TODO : andus >> enode Sender -- start --
			// TODO : andus >> rlp encode -> byte ( enode type )
			_, err = Conn.Write(enodeByte) // TODO : andus >> enode url 전송
			if err != nil {
				log.Fatal("andus >> Write", err)
			}
		}
	}

}

func (fc *FairnodeClient) receiveOtprn() {

	//TODO : andus >> 1. OTPRN 수신
	localServerConn, err := net.ListenUDP("udp", fc.SAddrUDP)
	if err != nil {
		log.Fatal("Udp Server", err)
	}

	defer localServerConn.Close()

	tsOtprnByte := make([]byte, 4096)

	for {
		n, fairServerAddr, err := localServerConn.ReadFromUDP(tsOtprnByte)
		log.Println("andus >> otprn 수신", string(tsOtprnByte[:n]), " from ", fairServerAddr)
		if err != nil {
			log.Println("andus >> otprn 수신 에러", err)
		}
	}
	// TODO : andus >> 수신된 otprn디코딩
	var tsOtprn otprn.TransferOtprn
	rlp.DecodeBytes(tsOtprnByte, &tsOtprn)

	//TODO : andus >> 2. OTRRN 검증
	fairPubKey, err := crypto.SigToPub(tsOtprn.Hash.Bytes(), tsOtprn.Sig)
	if err != nil {
		log.Println("andus >> OTPRN 공개키 로드 에러")
	}

	if crypto.VerifySignature(crypto.FromECDSAPub(fairPubKey), tsOtprn.Hash.Bytes(), tsOtprn.Sig) {
		otprnHash := tsOtprn.Otp.HashOtprn()
		if otprnHash == tsOtprn.Hash {
			// TODO: andus >> 검증완료, Otprn 저장
			fc.Otprn = &tsOtprn.Otp
			//TODO : andus >> 3. 참여여부 확인

			if ok := fairutil.IsJoinOK(fc.Otprn, fc.GetCurrentJoinNunce(), fc.Coinbase); ok {
				//TODO : andus >> 참가 가능할 때 처리
				//TODO : andus >> 6. TCP 연결 채널에 메세지 보내기
				fc.TcpConnStartCh <- struct{}{}
			}

		} else {
			// TODO: andus >> 검증실패..
			log.Println("andus >> OTPRN 검증 실패")

		}
	} else {
		// TODO: andus >> 서명 검증실패..
		log.Println("andus >> OTPRN 공개키 검증 실패")
	}
}

func (fc *FairnodeClient) TCPtoFairNode() {
	defer fc.wg.Done()

	for {
		<-fc.TcpConnStartCh

		//TODO : andus >> TCP 통신 to FairNode
		//TODO : andus >> 1. fair Node에 TCP 연결
		//TODO : andus >> 2. OTPRN, enode값 전달

		// TODO : andus >> 1. 채굴 리스 리스트와 총 채굴리그 해시 수신

		// TODO : andus >> 1.1 추후 서명값 검증 해야함...

		// TODO : andus >> 4. JoinTx 생성 ( fairnode를 수신자로 하는 tx, 참가비 보냄...)

		var fairNodeAddr common.Address // TODO : andus >> 보내는 fairNode의 Address(주소)

		// TODO : andus >> joinNonce 현재 상태 조회
		stateDb, err := fc.BlockChain.State()
		if err != nil {
			log.Println("andus >> 상태DB을 읽어오는데 문제 발생")
		}
		currentJoinNonce := stateDb.GetJoinNonce(*fc.Coinbase)

		signer := types.NewEIP155Signer(big.NewInt(18))

		// TODO : andus >> joinNonce Fairnode에게 보내는 Tx
		tx, err := types.SignTx(types.NewTransaction(currentJoinNonce, fairNodeAddr, new(big.Int), 0, new(big.Int), nil), signer, fc.PrivateKey)
		if err != nil {
			log.Println("andus >> JoinTx 서명 에러")
		}

		log.Println("andus >> JoinTx 생성 Success", tx)

		// TODO : andus >> txpool에 추가.. 알아서 이더리움 프로세스 타고 날라감....
		fc.txPool.AddLocal(tx)

		// TODO : andus >> 2. 각 enode값을 이용해서 피어 접속

		//enodes := []string{"enode://12121@111.111.111:3303"}
		//for _ := range enodes {
		//	old, _ := disco.ParseNode(boot)
		//	srv.AddPeer(old)
		//}

		// TODO : andsu >> 3. mining.start()
		if !fc.Miner.IsMining() {
			fc.Miner.StartMining(1)
		}
		select {
		// type : types.TransferBlock
		case signedBlock := <-fc.WinningBlockCh:
			// TODO : andus >> 위닝블록 TCP전송
			fmt.Println("위닝 블록을 받아서 페어노드에게 보내요", signedBlock)
		}

	}

	// TODO : andus >> 페어노드가 전송한 최종 블록을 받는다.
	// TODO : andus >> 받은 블록을 검증한다
	// TODO : andus >> worker에 블록이 등록 되게 한다

	fc.FinalBlockCh <- &types.TransferBlock{}
}

func (fc *FairnodeClient) Stop() {
	fc.Running = false
}

func (fc *FairnodeClient) GetCurrentJoinNunce() uint64 {
	stateDb, err := fc.BlockChain.State()
	if err != nil {
		log.Println("andus >> 상태DB을 읽어오는데 문제 발생")
	}

	return stateDb.GetJoinNonce(*fc.Coinbase)
}

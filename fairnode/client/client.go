package fairnodeclient

// TODO : andus >> Geth - FairNode 사이에 연결 되는 부분..

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/accounts/keystore"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/fairnode/fairutil"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/p2p/discv5"
	"github.com/anduschain/go-anduschain/p2p/nat"
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

type FairnodeClient struct {
	Otprn *otprn.Otprn
	//OtprnCh chan *otprn.Otprn
	WinningBlockCh     chan *types.TransferBlock // TODO : andus >> worker의 위닝 블록을 받는 채널... -> Fairnode에게 쏜다
	FinalBlockCh       chan *types.TransferBlock
	Running            bool
	wg                 sync.WaitGroup
	BlockChain         *core.BlockChain
	Miner              DebMiner
	Coinbase           *common.Address
	keystore           *keystore.KeyStore
	txPool             *core.TxPool
	PrivateKey         *ecdsa.PrivateKey
	SAddrUDP           *net.UDPAddr
	LAddrUDP           *net.UDPAddr
	TcpConnStartCh     chan struct{}
	submitEnodeExitCh  chan struct{}
	receiveOtprnExitCh chan struct{}
}

func New(wbCh chan *types.TransferBlock, fbCh chan *types.TransferBlock, blockChain *core.BlockChain, miner DebMiner, tp *core.TxPool) *FairnodeClient {

	fmt.Println("andus >> fair node client New 패어노드 클라이언트 실행 했다.")

	serverAddr, err := net.ResolveUDPAddr("udp", "121.134.35.45:60002") // 전송 60002
	if err != nil {
		log.Println("andus >> UDPtoFairNode, ServerAddr", err)
	}

	localAddr, err := net.ResolveUDPAddr("udp", ":50002") // 수신 50002
	if err != nil {
		log.Println("andus >> UDPtoFairNode, LocalAddr", err)
	}

	fcClient := &FairnodeClient{
		Otprn:              nil,
		WinningBlockCh:     wbCh,
		FinalBlockCh:       fbCh,
		Running:            false,
		BlockChain:         blockChain,
		Miner:              miner,
		txPool:             tp,
		SAddrUDP:           serverAddr,
		LAddrUDP:           localAddr,
		TcpConnStartCh:     make(chan struct{}),
		submitEnodeExitCh:  make(chan struct{}),
		receiveOtprnExitCh: make(chan struct{}),
	}

	return fcClient
}

//TODO : andus >> fairNode 관련 함수....
func (fc *FairnodeClient) StartToFairNode(coinbase *common.Address, ks *keystore.KeyStore) error {
	fc.Running = true
	fc.keystore = ks
	fc.Coinbase = coinbase

	if unlockedKey := fc.keystore.GetUnlockedPrivKey(*coinbase); unlockedKey == nil {
		return errors.New("andus >> 코인베이스가 언락되지 않았습니다.")
	} else {
		fc.PrivateKey = unlockedKey

		// udp
		go fc.UDPtoFairNode()

		// tcp
		//go fc.TCPtoFairNode()
	}
	return nil
}
func (fc *FairnodeClient) UDPtoFairNode() {
	//TODO : andus >> udp 통신 to FairNode
	go fc.submitEnode()
	go fc.receiveOtprn()
}

func (fc *FairnodeClient) submitEnode() {
	// TODO : andus >> FairNode IP : localhost UDP Listener 11/06 -- start --
	Conn, err := net.DialUDP("udp", nil, fc.SAddrUDP)
	if err != nil {
		log.Println("andus >> UDPtoFairNode, DialUDP", err)
	}

	defer Conn.Close()

	// TODO : andus >> FairNode IP : localhost UDP Listener 11/06 -- end --
	t := time.NewTicker(60 * time.Second)

	realaddr := Conn.LocalAddr().(*net.UDPAddr)
	node := discv5.NewNode(
		discv5.PubkeyID(&fc.PrivateKey.PublicKey),
		realaddr.IP,
		uint16(realaddr.Port),
		uint16(realaddr.Port),
	)

	enode := node.String()                     // TODO : andus >> enode
	enodeByte, err := rlp.EncodeToBytes(enode) // TODO : andus >> enode to byte
	if err != nil {
		log.Fatal("andus >> EncodeToBytes", err)
	}

	writeData := func() {
		_, err = Conn.Write(enodeByte) // TODO : andus >> enode url 전송
		fmt.Println("andus >> enode 전송")
		if err != nil {
			log.Println("andus >> Write", err)
		}
	}

	writeData()

	for {
		select {
		case <-t.C:
			//TODO : andus >> FairNode에게 enode값 전송 ( 1분단위)
			// TODO : andus >> enode Sender -- start --
			// TODO : andus >> rlp encode -> byte ( enode type )
			writeData()
		case <-fc.submitEnodeExitCh:
			fmt.Println("andus >> submitEnode 종료됨")
			return
		}
	}
}

func (fc *FairnodeClient) receiveOtprn() {

	//TODO : andus >> 1. OTPRN 수신

	localServerConn, err := net.ListenUDP("udp", fc.LAddrUDP)
	if err != nil {
		log.Println("Udp Server", err)
	}

	// TODO : andus >> NAT 추가 --- start ---

	natm, err := nat.Parse("any")
	if err != nil {
		log.Fatalf("-nat: %v", err)
	}

	realaddr := localServerConn.LocalAddr().(*net.UDPAddr)
	if true {
		if !realaddr.IP.IsLoopback() {
			go nat.Map(natm, nil, "udp", realaddr.Port, realaddr.Port, "andus fairnode discovery")
		}
		// TODO: react to external IP changes over time.
		if ext, err := natm.ExternalIP(); err == nil {
			realaddr = &net.UDPAddr{IP: ext, Port: realaddr.Port}
		}
	}

	// TODO : andus >> NAT 추가 --- end ---

	defer localServerConn.Close()

	tsOtprnByte := make([]byte, 4096)

	for {
		select {
		case <-fc.receiveOtprnExitCh:
			fmt.Println("andus >> receiveOtprn 종료됨")
			return
		default:
			localServerConn.SetReadDeadline(time.Now().Add(3 * time.Second))
			n, _, err := localServerConn.ReadFromUDP(tsOtprnByte)
			//fmt.Println("andus >> otprn 수신 from ", fairServerAddr)
			if err != nil {
				log.Println("andus >> otprn 수신 에러", err)
				if err.(net.Error).Timeout() {
					continue
				}
				return
			}

			if n > 0 {
				// TODO : andus >> 수신된 otprn디코딩
				var tsOtprn otprn.TransferOtprn
				rlp.DecodeBytes(tsOtprnByte, &tsOtprn)

				//TODO : andus >> 2. OTRRN 검증
				fairPubKey, err := crypto.SigToPub(tsOtprn.Hash.Bytes(), tsOtprn.Sig)
				if err != nil {
					log.Println("andus >> OTPRN 공개키 로드 에러")
				}

				if crypto.VerifySignature(crypto.FromECDSAPub(fairPubKey), tsOtprn.Hash.Bytes(), tsOtprn.Sig[:64]) {
					otprnHash := tsOtprn.Otp.HashOtprn()
					if otprnHash == tsOtprn.Hash {
						// TODO: andus >> 검증완료, Otprn 저장
						fc.Otprn = &tsOtprn.Otp
						//TODO : andus >> 3. 참여여부 확인

						fmt.Println("andus >> OTPRN 검증 완료")

						if ok := fairutil.IsJoinOK(fc.Otprn, fc.GetCurrentJoinNonce(), fc.Coinbase); ok {
							//TODO : andus >> 참가 가능할 때 처리
							//TODO : andus >> 6. TCP 연결 채널에 메세지 보내기
							//fc.TcpConnStartCh <- struct{}{}

							fmt.Println("andus >> 채굴 참여 대상자 확인")

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
		}
	}
}

func (fc *FairnodeClient) TCPtoFairNode() {
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

		currentJoinNonce := fc.GetCurrentJoinNonce()

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
	fc.submitEnodeExitCh <- struct{}{}
	fc.receiveOtprnExitCh <- struct{}{}
}

func (fc *FairnodeClient) GetCurrentJoinNonce() uint64 {
	stateDb, err := fc.BlockChain.State()
	if err != nil {
		log.Println("andus >> 상태DB을 읽어오는데 문제 발생")
	}

	return stateDb.GetJoinNonce(*fc.Coinbase)
}

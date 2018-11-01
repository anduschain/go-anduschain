package fairnodeclient

// TODO : andus >> Geth - FairNode 사이에 연결 되는 부분..

import (
	"fmt"
	"github.com/anduschain/go-anduschain/accounts/keystore"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/fairutil"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"log"
	"math/big"
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
	WinningBlockCh chan *types.TransferBlock // TODO : andus >> worker의 위닝 블록을 받는 채널... -> Fairnode에게 쏜다
	FinalBlockCh   chan *types.TransferBlock
	Running        bool
	wg             sync.WaitGroup
	BlockChain     *core.BlockChain
	Miner          DebMiner
	Coinbase       *common.Address
	keystore       *keystore.KeyStore
	txPool         *core.TxPool
}

func New(wbCh chan *types.TransferBlock, fbCh chan *types.TransferBlock, blockChain *core.BlockChain, miner DebMiner, coinbase *common.Address, ks *keystore.KeyStore, tp *core.TxPool) *FairnodeClient {
	return &FairnodeClient{
		Otprn:          nil,
		WinningBlockCh: wbCh,
		FinalBlockCh:   fbCh,
		Running:        false,
		BlockChain:     blockChain,
		Miner:          miner,
		Coinbase:       coinbase,
		keystore:       ks,
		txPool:         tp,
	}
}

//TODO : andus >> fairNode 관련 함수....
func (fc *FairnodeClient) StartToFairNode() error {
	fc.Running = true
	fc.wg.Add(2)

	tcpStart := make(chan interface{})

	// TODO : andus >> 마이닝 켜저 있으면 종료

	if fc.Miner.IsMining() {
		fc.Miner.StopMining()
	}

	// udp
	go fc.UDPtoFairNode(tcpStart)

	// tcp
	go fc.TCPtoFairNode(tcpStart)

	return nil
}
func (fc *FairnodeClient) UDPtoFairNode(ch chan interface{}) {
	defer fc.wg.Done()
	//TODO : andus >> udp 통신 to FairNode

	t := time.NewTicker(60 * time.Second)
	select {
	case <-t.C:
		//TODO : andus >> FairNode에게 enode값 전송 ( 1분단위)
	}

	//TODO : andus >> 1. OTPRN 수신
	//TODO : andus >> 2. OTRRN 검증
	otp, _ := otprn.New()
	checkedOtprn, err := otp.CheckOtprn("수신된 otprn을 넣고")

	//TODO : andus >> Otprn 저장
	fc.Otprn = checkedOtprn

	if err != nil {

	}

	//TODO : andus >> 3. 참여여부 확인
	if ok := fairutil.IsJoinOK(); ok {
		//TODO : andus >> 참가 가능할 때 처리

		//TODO : andus >> 5. StatDB join_noonce를 더하기 1 ( join_nonce++ ) >> 블록 확정시 joinTx( 수신처가 페어노드인 tx) 를 검사해서 joinNounce 값 변경 ( 위치 조절 됨 )

		//TODO : andus >> 6. TCP 연결 채널에 메세지 보내기
		ch <- "start"
	}

}

func (fc *FairnodeClient) TCPtoFairNode(ch chan interface{}) {
	defer fc.wg.Done()
	//TODO : andus >> TCP 통신 to FairNode
	//TODO : andus >> 1. fair Node에 TCP 연결
	//TODO : andus >> 2. OTPRN, enode값 전달

	for {
		<-ch

		// TODO : andus >> 1. 채굴 리스 리스트와 총 채굴리그 해시 수신

		// TODO : andus >> 1.1 추후 서명값 검증 해야함...

		// TODO : andus >> 4. JoinTx 생성 ( fairnode를 수신자로 하는 tx, 참가비 보냄...)

		var fairNodeAddr common.Address // TODO : andus >> 보내는 fairNode의 Address(주소)

		unlockedKey, err := fc.keystore.GetUnlockedPrivKey(*fc.Coinbase)
		if err != nil {
			log.Println("andus >>", err)
		}

		// TODO : andus >> joinNonce 현재 상태 조회
		stateDb, err := fc.BlockChain.State()
		if err != nil {
			log.Println("andus >> 상태DB을 읽어오는데 문제 발생")
		}
		currentJoinNonce := stateDb.GetJoinNonce(*fc.Coinbase)

		signer := types.NewEIP155Signer(big.NewInt(18))

		// TODO : andus >> joinNonce Fairnode에게 보내는 Tx
		tx, err := types.SignTx(types.NewTransaction(currentJoinNonce, fairNodeAddr, new(big.Int), 0, new(big.Int), nil), signer, unlockedKey)
		if err != nil {
			log.Println("andus >> JoinTx 서명 에러")
		}

		log.Println("andus >> JoinTx 생성 Success", tx)

		// TODO : andus >> txpool에 추가.. 알아서 이더리움 프로세스 타고 날라감....
		fc.txPool.AddLocal(tx)

		// TODO : andus >> 2. 각 enode값을 이용해서 피어 접속

		//enodes := []string{"enode://12121@111.111.111:3303"}
		//for _ := range enodes {
		//	//old, _ := disco.ParseNode(boot)
		//	//srv.AddPeer(old)
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

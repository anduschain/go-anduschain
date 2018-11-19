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
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/p2p/discv5"
	"log"
	"net"
	"sync"
)

type FairnodeClient struct {
	Otprn *otprn.Otprn
	//OtprnCh chan *otprn.Otprn
	WinningBlockCh     chan *types.TransferBlock // TODO : andus >> worker의 위닝 블록을 받는 채널... -> Fairnode에게 쏜다
	FinalBlockCh       chan *types.TransferBlock
	Running            bool
	wg                 sync.WaitGroup
	BlockChain         *core.BlockChain
	Coinbase           *common.Address
	keystore           *keystore.KeyStore
	txPool             *core.TxPool
	CoinBasePrivateKey *ecdsa.PrivateKey

	SAddrUDP *net.UDPAddr
	LAddrUDP *net.UDPAddr

	SAddrTCP *net.TCPAddr
	LaddrTCP *net.TCPAddr

	TcpConnStartCh      chan struct{}
	submitEnodeExitCh   chan struct{}
	receiveOtprnExitCh  chan struct{}
	tcptoFairNodeStopCh chan struct{}

	NodeKey *ecdsa.PrivateKey

	Enode *discv5.Node
}

func New(wbCh chan *types.TransferBlock, fbCh chan *types.TransferBlock, blockChain *core.BlockChain, tp *core.TxPool) *FairnodeClient {

	fmt.Println("andus >> fair node client New 패어노드 클라이언트 실행 했다.")

	// TODO : andus >> UDP Resolve Udp
	serverAddr, err := net.ResolveUDPAddr("udp", "121.134.35.45:60002") // 전송 60002
	if err != nil {
		log.Println("andus >> UDPtoFairNode, ServerAddr", err)
	}

	localAddr, err := net.ResolveUDPAddr("udp", ":50002") // 수신 50002
	if err != nil {
		log.Println("andus >> UDPtoFairNode, LocalAddr", err)
	}

	// TODO : andus >> TCP Resolve Tcp
	localAddrTcp, err := net.ResolveTCPAddr("tcp", ":50002")
	if err != nil {
		log.Println("andus >> UDPtoFairNode, ServerAddr", err)
	}

	serverAddrTcp, err := net.ResolveTCPAddr("tcp", "121.134.35.45:60002") // 전송 60002
	if err != nil {
		log.Println("andus >> UDPtoFairNode, ServerAddr", err)
	}

	fcClient := &FairnodeClient{
		Otprn:              nil,
		WinningBlockCh:     wbCh,
		FinalBlockCh:       fbCh,
		Running:            false,
		BlockChain:         blockChain,
		txPool:             tp,
		SAddrUDP:           serverAddr,
		LAddrUDP:           localAddr,
		LaddrTCP:           localAddrTcp,
		SAddrTCP:           serverAddrTcp,
		TcpConnStartCh:     make(chan struct{}),
		submitEnodeExitCh:  make(chan struct{}),
		receiveOtprnExitCh: make(chan struct{}),
	}

	return fcClient
}

//TODO : andus >> fairNode 관련 함수....
func (fc *FairnodeClient) StartToFairNode(coinbase *common.Address, ks *keystore.KeyStore, NodeKey *ecdsa.PrivateKey) error {
	fc.Running = true
	fc.keystore = ks
	fc.Coinbase = coinbase
	fc.NodeKey = NodeKey

	if unlockedKey := fc.keystore.GetUnlockedPrivKey(*coinbase); unlockedKey == nil {
		return errors.New("andus >> 코인베이스가 언락되지 않았습니다.")
	} else {
		fc.CoinBasePrivateKey = unlockedKey

		// udp
		go fc.UDPtoFairNode()

		// tcp
		go fc.TCPtoFairNode()
	}
	return nil
}

func (fc *FairnodeClient) Stop() {
	if fc.Running {
		fc.Running = false
		fc.submitEnodeExitCh <- struct{}{}
		fc.receiveOtprnExitCh <- struct{}{}
		fc.tcptoFairNodeStopCh <- struct{}{}
	}
}

func (fc *FairnodeClient) GetCurrentJoinNonce() uint64 {
	stateDb, err := fc.BlockChain.State()
	if err != nil {
		log.Println("andus >> 상태DB을 읽어오는데 문제 발생")
	}

	return stateDb.GetJoinNonce(*fc.Coinbase)
}

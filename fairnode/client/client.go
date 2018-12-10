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
	"github.com/anduschain/go-anduschain/fairnode/client/config"
	clinetTcp "github.com/anduschain/go-anduschain/fairnode/client/tcp"
	clinetTypes "github.com/anduschain/go-anduschain/fairnode/client/types"
	clinetUdp "github.com/anduschain/go-anduschain/fairnode/client/udp"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/p2p"
	"log"
	"math/big"
	"net"
	"sync"
)

const (
	TcpStop = iota
	StopTCPtoFairNode

	//// TODO : andus >> Fair Node Address
	//FAIRNODE_ADDRESS = "0x5922af64E91f4B10AF896De8Fd372075569a1440"
	//TICKET_PRICE     = 100
)

type FairnodeClient struct {
	Otprn *otprn.Otprn
	//OtprnCh chan *otprn.Otprn
	WinningBlockCh     chan *types.TransferBlock // TODO : andus >> worker의 위닝 블록을 받는 채널... -> Fairnode에게 쏜다
	FinalBlockCh       chan *types.TransferBlock
	Running            bool
	wg                 sync.WaitGroup
	BlockChain         *core.BlockChain
	Coinbase           common.Address
	keystore           *keystore.KeyStore
	txPool             *core.TxPool
	CoinBasePrivateKey ecdsa.PrivateKey
	FairPubKey         ecdsa.PublicKey

	//SAddrUDP *net.UDPAddr
	//LAddrUDP *net.UDPAddr

	//SAddrTCP *net.TCPAddr
	//LaddrTCP *net.TCPAddr

	TcpConnStartCh     chan struct{}
	submitEnodeExitCh  chan struct{}
	receiveOtprnExitCh chan struct{}
	readLoopStopCh     chan struct{}
	writeLoopStopCh    chan struct{}

	tcptoFairNodeExitCh chan int
	tcpConnStopCh       chan int

	tcpRunning bool
	TcpDialer  *net.TCPConn

	Srv *p2p.Server
	NAT string

	mux sync.Mutex

	StartCh chan struct{} // 블록생성 시작 채널

	Services map[string]clinetTypes.ServiceFunc
}

func New(wbCh chan *types.TransferBlock, fbCh chan *types.TransferBlock, blockChain *core.BlockChain, tp *core.TxPool) *FairnodeClient {

	fc := &FairnodeClient{
		Otprn:              nil,
		WinningBlockCh:     wbCh,
		FinalBlockCh:       fbCh,
		Running:            false,
		BlockChain:         blockChain,
		txPool:             tp,
		TcpConnStartCh:     make(chan struct{}),
		submitEnodeExitCh:  make(chan struct{}),
		receiveOtprnExitCh: make(chan struct{}),

		tcptoFairNodeExitCh: make(chan int),
		tcpConnStopCh:       make(chan int),
		tcpRunning:          false,
		NAT:                 config.DefaultConfig.NAT,
		StartCh:             make(chan struct{}),

		Services: make(map[string]clinetTypes.ServiceFunc),
	}

	// Default Setting  [ FairServer : 121.134.35.45:60002, GethPort : 50002 ]
	faiorServerString := fmt.Sprintf("%s:%s", config.DefaultConfig.FairServerIp, config.DefaultConfig.FairServerPort)
	clientString := fmt.Sprintf(":%s", config.DefaultConfig.ClientPort)

	//// UDP
	//fc.SAddrUDP, _ = net.ResolveUDPAddr("udp", faiorServerString)
	//fc.LAddrUDP, _ = net.ResolveUDPAddr("udp", clientString)

	// TCP
	//fc.SAddrTCP, _ = net.ResolveTCPAddr("tcp", faiorServerString)
	//fc.LaddrTCP, _ = net.ResolveTCPAddr("tcp", clientString)

	t, err := clinetTcp.New(faiorServerString, clientString, fc)
	if err != nil {

	}

	u, err := clinetUdp.New(faiorServerString, clientString, fc)
	if err != nil {

	}

	fc.Services["clinetTcp"] = t
	fc.Services["clinetUdp"] = u

	return fc
}

//TODO : andus >> fairNode 관련 함수....
func (fc *FairnodeClient) StartToFairNode(coinbase *common.Address, ks *keystore.KeyStore, srv *p2p.Server) error {
	fmt.Println("andus >> fair node client New 패어노드 클라이언트 실행 했다.")
	fc.Running = true
	fc.keystore = ks
	fc.Coinbase = *coinbase
	fc.Srv = srv

	// coinbase unlock check
	if unlockedKey, ok := fc.keystore.GetUnlockedPrivKey(fc.Coinbase); ok {

		fc.CoinBasePrivateKey = *unlockedKey

		//// fairudp
		//go fc.UDPtoFairNode()
		//
		//// fairtcp
		//go fc.TCPtoFairNode()

	} else {
		return errors.New("andus >> 코인베이스가 언락되지 않았습니다.")
	}

	return nil
}

func (fc *FairnodeClient) Stop() {
	if fc.Running {
		fc.Running = false
		fc.submitEnodeExitCh <- struct{}{}
		fc.receiveOtprnExitCh <- struct{}{}

		if fc.tcpRunning {
			// loop kill, fairtcp kill
			fc.tcpConnStopCh <- TcpStop
			fc.tcpRunning = false
		} else {
			fc.tcptoFairNodeExitCh <- StopTCPtoFairNode
		}

		// 마이너 종료시 계정 Lock
		if err := fc.keystore.Lock(fc.Coinbase); err != nil {
			log.Println("Error[andus] : ", err)
		}
	}
}

func (fc *FairnodeClient) GetP2PServer() *p2p.Server            { return fc.Srv }
func (fc *FairnodeClient) GetCoinbase() common.Address          { return fc.Coinbase }
func (fc *FairnodeClient) SetOtprn(otprn *otprn.Otprn)          { fc.Otprn = otprn }
func (fc *FairnodeClient) GetOtprn() *otprn.Otprn               { return fc.Otprn }
func (fc *FairnodeClient) GetTxpool() *core.TxPool              { return fc.txPool }
func (fc *FairnodeClient) GetBlockChain() *core.BlockChain      { return fc.BlockChain }
func (fc *FairnodeClient) GetCoinbsePrivKey() *ecdsa.PrivateKey { return &fc.CoinBasePrivateKey }

func (fc *FairnodeClient) GetCurrentJoinNonce() uint64 {
	stateDb, err := fc.BlockChain.State()
	if err != nil {
		log.Println("Error[andus] : 상태DB을 읽어오는데 문제 발생")
	}

	return stateDb.GetJoinNonce(fc.Coinbase)
}

func (fc *FairnodeClient) GetCurrentBalance() *big.Int {
	stateDb, err := fc.BlockChain.State()
	if err != nil {
		log.Println("Error[andus] : 상태DB을 읽어오는데 문제 발생")
	}

	return stateDb.GetBalance(fc.Coinbase)
}

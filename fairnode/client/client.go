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
	"github.com/anduschain/go-anduschain/fairnode/client/interface"
	clinetTcp "github.com/anduschain/go-anduschain/fairnode/client/tcp"
	clinetTypes "github.com/anduschain/go-anduschain/fairnode/client/types"
	clinetUdp "github.com/anduschain/go-anduschain/fairnode/client/udp"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/fairnode/fairutil/queue"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/p2p"
	"log"
	"math/big"
	"sync"
)

var (
	errUnlockCoinbase = errors.New("코인베이스가 언락되지 않았습니다.")
)

type FairnodeClient struct {
	Services           map[string]_interface.ServiceFunc
	Srv                *p2p.Server
	CoinBasePrivateKey ecdsa.PrivateKey
	txPool             *core.TxPool
	BlockChain         *core.BlockChain
	Coinbase           common.Address
	keystore           *keystore.KeyStore
	Running            bool
	wg                 sync.WaitGroup

	TcpConnStartCh     chan struct{}
	submitEnodeExitCh  chan struct{}
	receiveOtprnExitCh chan struct{}
	readLoopStopCh     chan struct{}
	writeLoopStopCh    chan struct{}

	StartCh chan struct{} // 블록생성 시작 채널

	chans  fairtypes.Channals
	Signer types.EIP155Signer

	mux sync.Mutex

	wBlocks     map[common.Hash]map[common.Hash]*types.Block // 위닝 블록 임시 저장
	IsBlockMine bool

	UsingOtprn *clinetTypes.OtprnWithSig
	OtprnQueue *queue.Queue
}

func New(chans fairtypes.Channals, blockChain *core.BlockChain, tp *core.TxPool) *FairnodeClient {

	fc := &FairnodeClient{
		chans:              chans,
		Running:            false,
		BlockChain:         blockChain,
		txPool:             tp,
		TcpConnStartCh:     make(chan struct{}),
		submitEnodeExitCh:  make(chan struct{}),
		receiveOtprnExitCh: make(chan struct{}),
		StartCh:            make(chan struct{}),
		Services:           make(map[string]_interface.ServiceFunc),
		Signer:             types.NewEIP155Signer(blockChain.Config().ChainID),
		wBlocks:            make(map[common.Hash]map[common.Hash]*types.Block),
		IsBlockMine:        false,
		OtprnQueue:         queue.NewQueue(1),
		UsingOtprn:         nil,
	}

	// Default Setting  [ FairServer : 121.134.35.45:60002, GethPort : 50002 ]
	faiorServerString := fmt.Sprintf("%s:%s", config.DefaultConfig.FairServerIp, config.DefaultConfig.FairServerPort)
	clientString := fmt.Sprintf(":%s", config.DefaultConfig.ClientPort)

	t, _ := clinetTcp.New(faiorServerString, clientString, fc)

	u, _ := clinetUdp.New(faiorServerString, clientString, fc, t)

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
	} else {
		return errUnlockCoinbase
	}

	// Udp Service running
	for name, serv := range fc.Services {
		log.Println(fmt.Sprintf("Info[andus] : %s Running", name))
		err := serv.Start()
		if err != nil {
			return err
		}
	}

	return nil
}

func (fc *FairnodeClient) Stop() {
	//if fc.Running {
	//	fc.Running = false
	//	fc.submitEnodeExitCh <- struct{}{}
	//	fc.receiveOtprnExitCh <- struct{}{}
	//
	//	if fc.tcpRunning {
	//		// loop kill, fairtcp kill
	//		fc.tcpConnStopCh <- TcpStop
	//		fc.tcpRunning = false
	//	} else {
	//		fc.tcptoFairNodeExitCh <- StopTCPtoFairNode
	//	}
	//
	//}

	for name, serv := range fc.Services {
		log.Println(fmt.Sprintf("Info[andus] : %s Stop", name))
		err := serv.Stop()
		if err != nil {
			log.Println("Error[andus] : ", err)
		}
	}

	if fc.Coinbase != (common.Address{}) {
		// 마이너 종료시 계정 Lock
		if err := fc.keystore.Lock(fc.Coinbase); err != nil {
			log.Println("Error[andus] : ", err)
		}
	}

}

func (fc *FairnodeClient) StoreOtprnWidthSig(otprn *otprn.Otprn, sig []byte) {
	fc.mux.Lock()
	defer fc.mux.Unlock()
	fc.OtprnQueue.Push(&clinetTypes.OtprnWithSig{otprn, sig})
}

func (fc *FairnodeClient) GetStoreOtprnWidthSig() *otprn.Otprn {
	fc.mux.Lock()
	defer fc.mux.Unlock()

	otprnSig := fc.OtprnQueue.Pop().(*clinetTypes.OtprnWithSig)
	if otprnSig != nil {
		fc.UsingOtprn = otprnSig
		return otprnSig.Otprn
	}

	return nil
}

func (fc *FairnodeClient) GetUsingOtprnWithSig() *clinetTypes.OtprnWithSig { return fc.UsingOtprn }
func (fc *FairnodeClient) GetSavedOtprnHashs() []common.Hash {
	fc.mux.Lock()
	defer fc.mux.Unlock()
	var hashs []common.Hash
	if fc.OtprnQueue.Len() > 0 {
		for i := range fc.OtprnQueue.All() {
			otprnWithSig := fc.OtprnQueue.All()[i].(*clinetTypes.OtprnWithSig)
			hashs = append(hashs, otprnWithSig.Otprn.HashOtprn())
		}
	}

	return hashs
}
func (fc *FairnodeClient) FindOtprn(otprnHash common.Hash) *clinetTypes.OtprnWithSig {
	if fc.OtprnQueue.Len() > 0 {
		for i := range fc.OtprnQueue.All() {
			otprnWithSig := fc.OtprnQueue.All()[i].(*clinetTypes.OtprnWithSig)
			if otprnWithSig.Otprn.HashOtprn() == otprnHash {
				return otprnWithSig
			}
		}
	}

	return nil
}

func (fc *FairnodeClient) SetBlockMine(status bool) { fc.IsBlockMine = status }
func (fc *FairnodeClient) GetBlockMine() bool       { return fc.IsBlockMine }

func (fc *FairnodeClient) GetP2PServer() *p2p.Server   { return fc.Srv }
func (fc *FairnodeClient) GetCoinbase() common.Address { return fc.Coinbase }

func (fc *FairnodeClient) GetTxpool() *core.TxPool               { return fc.txPool }
func (fc *FairnodeClient) GetBlockChain() *core.BlockChain       { return fc.BlockChain }
func (fc *FairnodeClient) GetCoinbsePrivKey() *ecdsa.PrivateKey  { return &fc.CoinBasePrivateKey }
func (fc *FairnodeClient) BlockMakeStart() chan struct{}         { return fc.StartCh }
func (fc *FairnodeClient) VoteBlock() chan *fairtypes.Vote       { return fc.chans.GetWinningBlockCh() }
func (fc *FairnodeClient) FinalBlock() chan fairtypes.FinalBlock { return fc.chans.GetFinalBlockCh() }
func (fc *FairnodeClient) GetSigner() types.Signer               { return fc.Signer }

func (fc *FairnodeClient) GetCurrentJoinNonce() uint64 {
	stateDb, err := fc.BlockChain.StateAt(fc.BlockChain.CurrentHeader().Root)
	if err != nil {
		log.Println("Error[andus] : GetCurrentJoinNonce 상태DB을 읽어오는데 문제 발생", err)
	}

	return stateDb.GetJoinNonce(fc.Coinbase)
}

func (fc *FairnodeClient) GetCurrentBalance() *big.Int {
	stateDb, err := fc.BlockChain.StateAt(fc.BlockChain.CurrentHeader().Root)
	if err != nil {
		log.Println("Error[andus] : GetCurrentBalance 상태DB을 읽어오는데 문제 발생", err)
	}

	return stateDb.GetBalance(fc.Coinbase)
}

func (fc *FairnodeClient) GetCurrentNonce(addr common.Address) uint64 {
	stateDb, err := fc.BlockChain.StateAt(fc.BlockChain.CurrentHeader().Root)
	if err != nil {
		log.Println("Error[andus] : GetCurrentNonce 상태DB을 읽어오는데 문제 발생", err)
	}

	return stateDb.GetNonce(addr)
}

func (fc *FairnodeClient) SaveWiningBlock(otprnHash common.Hash, block *types.Block) {
	fc.mux.Lock()
	defer fc.mux.Unlock()
	if v, ok := fc.wBlocks[otprnHash]; ok {
		v[block.Hash()] = block
	} else {
		v = make(map[common.Hash]*types.Block)
		v[block.Hash()] = block
	}
}

func (fc *FairnodeClient) GetWinningBlock(otprnHash common.Hash, hash common.Hash) *types.Block {
	fc.mux.Lock()
	defer fc.mux.Unlock()
	if v, ok := fc.wBlocks[otprnHash]; ok {
		if block, ex := v[hash]; ex {
			return block
		}
	}
	return nil
}

func (fc *FairnodeClient) DelWinningBlock(otprnHash common.Hash) {
	fc.mux.Lock()
	defer fc.mux.Unlock()
	if _, ok := fc.wBlocks[otprnHash]; ok {
		delete(fc.wBlocks, otprnHash)
	}
}

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
	CurrnetOtprnHash   common.Hash
	OtprnWithSig       map[common.Hash]*clinetTypes.OtprnWithSig
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

	wBlocks map[common.Hash]*types.Block // 위닝 블록 임시 저장
}

func New(chans fairtypes.Channals, blockChain *core.BlockChain, tp *core.TxPool) *FairnodeClient {

	fc := &FairnodeClient{
		OtprnWithSig:       make(map[common.Hash]*clinetTypes.OtprnWithSig),
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
		wBlocks:            make(map[common.Hash]*types.Block),
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

func (fc *FairnodeClient) GetSavedOtprnHashs() []common.Hash {
	var re []common.Hash
	for key := range fc.OtprnWithSig {
		re = append(re, key)
	}

	return re
}
func (fc *FairnodeClient) GetP2PServer() *p2p.Server   { return fc.Srv }
func (fc *FairnodeClient) GetCoinbase() common.Address { return fc.Coinbase }
func (fc *FairnodeClient) SetOtprnWithSig(otprn *otprn.Otprn, sig []byte) {
	fc.OtprnWithSig[otprn.HashOtprn()] = &clinetTypes.OtprnWithSig{otprn, sig}
}
func (fc *FairnodeClient) GetOtprnWithSig(otprnHash common.Hash) *clinetTypes.OtprnWithSig {
	if val, ok := fc.OtprnWithSig[otprnHash]; ok {
		return val
	}
	return nil
}
func (fc *FairnodeClient) SetCurrnetOtprnHash(otprnHash common.Hash) { fc.CurrnetOtprnHash = otprnHash }
func (fc *FairnodeClient) GetCurrnetOtprnHash() common.Hash          { return fc.CurrnetOtprnHash }
func (fc *FairnodeClient) GetTxpool() *core.TxPool                   { return fc.txPool }
func (fc *FairnodeClient) GetBlockChain() *core.BlockChain           { return fc.BlockChain }
func (fc *FairnodeClient) GetCoinbsePrivKey() *ecdsa.PrivateKey      { return &fc.CoinBasePrivateKey }
func (fc *FairnodeClient) BlockMakeStart() chan struct{}             { return fc.StartCh }
func (fc *FairnodeClient) VoteBlock() chan *fairtypes.Vote           { return fc.chans.GetWinningBlockCh() }
func (fc *FairnodeClient) FinalBlock() chan fairtypes.FinalBlock     { return fc.chans.GetFinalBlockCh() }
func (fc *FairnodeClient) GetSigner() types.Signer                   { return fc.Signer }

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

func (fc *FairnodeClient) SaveWiningBlock(block *types.Block) {
	fc.mux.Lock()
	defer fc.mux.Unlock()
	fc.wBlocks[block.Hash()] = block
}

func (fc *FairnodeClient) GetWinningBlock(hash common.Hash) *types.Block {
	fc.mux.Lock()
	defer fc.mux.Unlock()
	if block, ok := fc.wBlocks[hash]; ok {
		return block
	}
	return nil
}

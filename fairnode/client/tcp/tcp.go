package tcp

import (
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	gethTypes "github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/client/config"
	"github.com/anduschain/go-anduschain/fairnode/client/interface"
	"github.com/anduschain/go-anduschain/fairnode/client/types"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/fairnode/transport"
	logger "github.com/anduschain/go-anduschain/log"
	"github.com/anduschain/go-anduschain/p2p/discover"
	"github.com/anduschain/go-anduschain/rlp"
	"math/big"
	"net"
	"time"
)

var (
	errorMakeJoinTx = errors.New("JoinTx 서명 에러")
	errorLeakCoin   = errors.New("잔액 부족으로 마이닝을 할 수 없음")
	errorAddTxPool  = errors.New("TxPool 추가 에러")
)

type Tcp struct {
	SAddrTCP *net.TCPAddr
	LaddrTCP *net.TCPAddr
	manger   _interface.Client
	services map[common.Hash]map[string]types.Goroutine
	IsRuning map[common.Hash]bool
	logger   logger.Logger
}

func New(faiorServerString string, clientString string, manger _interface.Client) (*Tcp, error) {

	SAddrTCP, err := net.ResolveTCPAddr("tcp", faiorServerString)
	if err != nil {
		return nil, err
	}

	LaddrTCP, err := net.ResolveTCPAddr("tcp", clientString)
	if err != nil {
		return nil, err
	}

	tcp := &Tcp{
		SAddrTCP: SAddrTCP,
		LaddrTCP: LaddrTCP,
		manger:   manger,
		services: make(map[common.Hash]map[string]types.Goroutine),
		IsRuning: make(map[common.Hash]bool),
		logger:   logger.New("fairclient", "TCP"),
	}

	return tcp, nil
}

func (t *Tcp) Start(otprnHash common.Hash) error {

	t.services[otprnHash] = make(map[string]types.Goroutine)
	t.services[otprnHash]["tcploop"] = types.Goroutine{t.tcpLoop, make(chan struct{})}

	if !t.IsRuning[otprnHash] {
		for name, serv := range t.services[otprnHash] {
			t.logger.Info("Service Start", "Service : ", name)
			go serv.Fn(serv.Exit, otprnHash)
		}
		t.IsRuning[otprnHash] = true
	}

	return nil
}

func (t *Tcp) Stop(otprnHash common.Hash) error {
	if t.IsRuning[otprnHash] {
		for _, srv := range t.services[otprnHash] {
			srv.Exit <- struct{}{}
		}
		t.IsRuning[otprnHash] = false
	}

	return nil
}

func (t *Tcp) tcpLoop(exit chan struct{}, v interface{}) {
	otprnHash, ok := v.(common.Hash)
	if !ok {
		return
	}

	otprnWithSig := t.manger.FindOtprn(otprnHash)
	if otprnWithSig == nil {
		return
	}

	defer func() {
		t.IsRuning[otprnHash] = false
		t.logger.Debug("TcpLoop Killed")
	}()

	conn, err := net.DialTCP("tcp", nil, t.SAddrTCP)
	if err != nil {
		t.logger.Error("DialTCP ", "error", err)
		return
	}

	tsp := transport.New(conn)

	node, err := discover.ParseNode(t.manger.GetP2PServer().NodeInfo().Enode)
	if err != nil {
		return
	}

	if node.IP.To4() == nil {
		addr, err := net.ResolveTCPAddr("tcp", conn.LocalAddr().String())
		if err != nil {
			return
		}

		node.IP = addr.IP
	}

	//참가 여부 확인
	transport.Send(tsp, transport.ReqLeagueJoinOK,
		fairtypes.TransferCheck{*otprnWithSig.Otprn, t.manger.GetCoinbase(), node.String()})

	notify := make(chan error)
	go func() {
		for {
			if err := t.handleMsg(tsp, otprnWithSig); err != nil {
				notify <- err
				return
			}
		}
	}()

Exit:
	for {
		select {
		case err := <-notify:
			t.logger.Error("handelMsg", "error", err)
			tsp.Close()
			break Exit
		case <-exit:
			tsp.Close()
			break Exit
		case vote := <-t.manger.VoteBlock():
			transport.Send(tsp, transport.SendBlockForVote, vote)
			t.logger.Info("Block Vote", "HeaderHash", vote.HeaderHash.String(), "Voter", vote.Voter.String())
		}
	}
}

func (t *Tcp) handleMsg(rw transport.MsgReadWriter, leagueOtprnwithsig *types.OtprnWithSig) error {
	msg, err := rw.ReadMsg()
	if err != nil {
		return err
	}
	defer func() {
		msg.Discard()
	}()

	var str string
	switch msg.Code {
	case transport.ResLeagueJoinFalse:
		// 참여 불가, Dial Close
		msg.Decode(&str)
		t.logger.Debug("FairNode Msg", str)
	case transport.MinerLeageStop:
		// 종료됨
		msg.Decode(&str)
		t.logger.Debug("FairNode Msg", str)
	case transport.SendLeageNodeList:
		// Add peer
		var nodeList []string
		msg.Decode(&nodeList)
		//otprn 교체
		t.manger.GetStoreOtprnWidthSig()
		t.logger.Info("SendLeageNodeList", "leagueCount", len(nodeList))
		for index := range nodeList {
			// addPeer 실행
			node, err := discover.ParseNode(nodeList[index])
			if err != nil {
				t.logger.Error("노드 URL 파싱에러", "error ", err)
				continue
			}
			t.logger.Debug("addPeer", "enode", nodeList[index])
			t.manger.GetP2PServer().AddPeer(node)
		}
	case transport.MakeJoinTx:
		// JoinTx 생성
		var m fairtypes.BlockMakeMessage
		msg.Decode(&m)

		if m.OtprnHash != leagueOtprnwithsig.Otprn.HashOtprn() {
			return errors.New("otprn이 다릅니다")
		}
		if m.Number != t.manger.GetBlockChain().CurrentBlock().Number().Uint64()+1 {
			return errors.New("동기화가 맞지 않습니다")
		}
		err := t.makeJoinTx(t.manger.GetBlockChain().Config().ChainID, leagueOtprnwithsig.Otprn, leagueOtprnwithsig.Sig)
		if err != nil {
			t.logger.Error("MakeJoinTx", "error", err)
			return err
		}
	case transport.MakeBlock:
		var m fairtypes.BlockMakeMessage
		msg.Decode(&m)
		if m.OtprnHash != leagueOtprnwithsig.Otprn.HashOtprn() {
			return errors.New("otprn이 다릅니다")
		}
		if m.Number != t.manger.GetBlockChain().CurrentBlock().Number().Uint64()+1 {
			return errors.New("동기화가 맞지 않습니다")
		}
		t.manger.SetBlockMine(true)
		t.logger.Info("블록 생성", "blockNum", m.Number, "otprnHash", m.OtprnHash.String())
		t.manger.BlockMakeStart() <- struct{}{}

	case transport.SendFinalBlock:
		tsFb := &fairtypes.TsFinalBlock{}
		msg.Decode(&tsFb)
		fb := tsFb.GetFinalBlock()
		t.logger.Info("파이널블록 수신", "blockNum", fb.Block.Number().String(), "miner", fb.Block.Coinbase().String(), "voteCount", len(fb.Block.Voter))
		t.manger.FinalBlock() <- *fb
	case transport.FinishLeague:
		var otprnhash common.Hash
		msg.Decode(&otprnhash)

		if otprnhash == leagueOtprnwithsig.Otprn.HashOtprn() {
			//otprn 교체 및 저장된 블록 제거
			t.manger.SetBlockMine(false)
			t.manger.GetStoreOtprnWidthSig()
			t.manger.DelWinningBlock(leagueOtprnwithsig.Otprn.HashOtprn())
			return errors.New("리그 종료")

		}
		return nil

	case transport.RequestWinningBlock:

		var headerHash common.Hash
		msg.Decode(&headerHash)
		block := t.manger.GetWinningBlock(leagueOtprnwithsig.Otprn.HashOtprn(), headerHash)
		if block == nil {
			break
		}
		fr := &fairtypes.ResWinningBlock{Block: block, OtprnHash: leagueOtprnwithsig.Otprn.HashOtprn()}

		transport.Send(rw, transport.SendWinningBlock, fr.GetTsResWinningBlock())
	default:
		return errors.New(fmt.Sprintf("알수 없는 메시지 코드 : %d", msg.Code))
	}

	return nil
}

func (t *Tcp) makeJoinTx(chanID *big.Int, otprn *otprn.Otprn, sig []byte) error {
	// TODO : andus >> JoinTx 생성 ( fairnode를 수신자로 하는 tx, 참가비 보냄...)
	// TODO : andus >> 잔액 조사 ( 임시 : 100 * 10^18 wei ) : 참가비 ( 수수료가 없는 tx )
	// TODO : andus >> joinNonce 현재 상태 조회
	currentBalance := t.manger.GetCurrentBalance()
	epoch := big.NewInt(int64(t.manger.GetBlockChain().Config().Deb.Epoch))

	// balance will be more then ticket price multiplex epoch.
	price := config.Price
	totalPrice := big.NewInt(int64(epoch.Uint64() * price.Uint64()))

	if currentBalance.Cmp(totalPrice) > 0 {
		currentJoinNonce := t.manger.GetCurrentJoinNonce()
		data := types.JoinTxData{
			JoinNonce:    currentJoinNonce,
			OtprnHash:    otprn.HashOtprn(),
			FairNodeSig:  sig,
			TimeStamp:    time.Now(),
			NextBlockNum: t.manger.GetBlockChain().CurrentBlock().Header().Number.Uint64() + 1,
		}
		joinTxData, err := rlp.EncodeToBytes(&data)
		if err != nil {
			t.logger.Error("makeJoinTx EncodeToBytes", "error", err)
		}
		txNonce := t.manger.GetTxpool().State().GetNonce(t.manger.GetCoinbase())

		// TODO : andus >> joinNonce Fairnode에게 보내는 Tx
		tx, err := gethTypes.SignTx(
			gethTypes.NewTransaction(txNonce, t.manger.GetBlockChain().Config().Deb.FairAddr, config.Price, 90000, big.NewInt(0), joinTxData), t.manger.GetSigner(), t.manger.GetCoinbsePrivKey())
		if err != nil {
			return errorMakeJoinTx
		}
		t.logger.Info("joinTx생성", "blockNum", data.NextBlockNum, "joinNonce", data.JoinNonce, "txHash", tx.Hash().String())
		// TODO : andus >> txpool에 추가.. 알아서 이더리움 프로세스 타고 날라감....
		if err := t.manger.GetTxpool().AddLocal(tx); err != nil {
			return errorAddTxPool
		}

		t.logger.Debug("Current Peer", "count", t.manger.GetP2PServer().PeerCount())
		//for i := range t.manger.GetP2PServer().Peers() {
		//		//	if peer := t.manger.GetP2PServer().Peers()[i]; peer != nil {
		//		//		t.logger.Debug("Current Peer", "peer", peer.String())
		//		//	}
		//		//}
	} else {
		// 잔액이 부족한 경우
		// 마이닝을 하지 못함..참여 불가, Dial Close
		return errorLeakCoin
	}

	return nil
}

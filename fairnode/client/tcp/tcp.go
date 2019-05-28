package tcp

import (
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/client/config"
	"github.com/anduschain/go-anduschain/fairnode/client/interface"
	ftype "github.com/anduschain/go-anduschain/fairnode/client/types"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/fairnode/transport"
	logger "github.com/anduschain/go-anduschain/log"
	"github.com/anduschain/go-anduschain/p2p/discover"
	"github.com/anduschain/go-anduschain/params"
	"io"
	"math/big"
	"net"
)

var (
	errorMakeJoinTx = errors.New("JoinTx 서명 에러")
	errorLeakCoin   = errors.New("잔액 부족으로 마이닝을 할 수 없음")
	errorAddTxPool  = errors.New("TxPool 추가 에러")
)

type Tcp struct {
	SAddrTCP   *net.TCPAddr
	LaddrTCP   *net.TCPAddr
	manger     _interface.Client
	services   map[common.Hash]map[string]ftype.Goroutine
	IsRuning   map[common.Hash]bool
	logger     logger.Logger
	leagueList []string
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
		services: make(map[common.Hash]map[string]ftype.Goroutine),
		IsRuning: make(map[common.Hash]bool),
		logger:   logger.New("fairclient", "TCP"),
	}

	return tcp, nil
}

func (t *Tcp) Start(otprnHash common.Hash) error {

	t.services[otprnHash] = make(map[string]ftype.Goroutine)
	t.services[otprnHash]["tcploop"] = ftype.Goroutine{t.tcpLoop, make(chan struct{})}

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

	if ok, _ := t.checkBalance(otprnWithSig.Otprn); !ok {
		t.logger.Error("Tcp connect error", "msg", errorLeakCoin.Error())
		return
	}

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
	if err := transport.Send(tsp, transport.ReqLeagueJoinOK,
		fairtypes.TransferCheck{
			Otprn:    *otprnWithSig.Otprn,
			Coinbase: t.manger.GetCoinbase(),
			Enode:    node.String(),
			Version:  params.Version,
			ChanID:   t.manger.GetBlockChain().Config().ChainID.Uint64(),
		}); err != nil {
		return
	}

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
			if err == io.EOF {
				t.logger.Debug("OTPRN 삭제", "msg", err)
				t.manger.DeleteStoreOtprnWidthSig()
			}
			tsp.Close()
			break Exit
		case <-exit:
			tsp.Close()
			break Exit
		case vote := <-t.manger.VoteBlock():
			t.logger.Debug("Block Vote Start", "HeaderHash", vote.HeaderHash.String(), "Voter", vote.Voter.String())
			err := transport.Send(tsp, transport.SendBlockForVote, vote)
			if err != nil {
				t.logger.Error("Block Vote", "msg", err)
				continue
			}
			t.logger.Debug("Block Vote End", "HeaderHash", vote.HeaderHash.String(), "Voter", vote.Voter.String())
		}
	}
}

func (t *Tcp) handleMsg(rw transport.MsgReadWriter, leagueOtprnwithsig *types.OtprnWithSig) error {
	msg, err := rw.ReadMsg()
	if err != nil {
		return err
	}

	if msg == nil {
		return nil
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
		t.leagueList = nodeList
		for index := range nodeList {
			// addPeer 실행
			node, err := discover.ParseNode(nodeList[index])
			if err != nil {
				t.logger.Error("노드 URL 파싱에러", "error ", err)
				continue
			}
			t.manger.GetP2PServer().AddPeer(node)
			t.logger.Debug("addPeer", "enode", nodeList[index])
		}
	case transport.MakeJoinTx:
		// JoinTx 생성
		var m fairtypes.BlockMakeMessage
		msg.Decode(&m)
		t.logger.Debug("TCP Message arrived", "mgs", "Join Tx 생성 메시지 도착")
		if m.OtprnHash != leagueOtprnwithsig.Otprn.HashOtprn() {
			t.manger.SetBlockMine(false)
			return errors.New("otprn이 다릅니다")
		}

		if m.Number != t.manger.GetBlockChain().CurrentBlock().Number().Uint64()+1 {
			t.logger.Error("동기화가 맞지 않습니다", "makeBlockNum", m.Number, "currentBlockNum", t.manger.GetBlockChain().CurrentBlock().Number().Uint64())
			t.manger.SetBlockMine(false)
			return nil
		}

		err := t.makeJoinTx(t.manger.GetBlockChain().Config().ChainID, leagueOtprnwithsig.Otprn, leagueOtprnwithsig.Sig)
		if err != nil {
			t.logger.Error("MakeJoinTx", "error", err)
			return err
		}
	case transport.MakeBlock:
		var m fairtypes.BlockMakeMessage
		msg.Decode(&m)
		t.logger.Debug("TCP Message arrived", "mgs", "블록 생성 메시지 도착")
		if m.OtprnHash != leagueOtprnwithsig.Otprn.HashOtprn() {
			t.manger.SetBlockMine(false)
			return errors.New("otprn이 다릅니다")
		}
		if m.Number != t.manger.GetBlockChain().CurrentBlock().Number().Uint64()+1 {
			t.logger.Error("동기화가 맞지 않습니다", "makeBlockNum", m.Number, "currentBlockNum", t.manger.GetBlockChain().CurrentBlock().Number().Uint64())
			t.manger.SetBlockMine(false)
			return nil
		}
		t.manger.SetBlockMine(true)
		t.logger.Info("블록 생성", "blockNum", m.Number, "otprnHash", m.OtprnHash.String())
		t.manger.BlockMakeStart() <- struct{}{}

	case transport.SendFinalBlock:
		t.logger.Debug("TCP Message arrived", "mgs", "파이널블록 수신 메시지 도착")
		if t.manger.GetBlockMine() {
			tsFb := &fairtypes.TsFinalBlock{}
			msg.Decode(&tsFb)
			fb := tsFb.GetFinalBlock()
			t.logger.Info("파이널블록 수신", "blockNum", fb.Block.Number().String(), "miner", fb.Block.Coinbase().String(), "voteCount", len(fb.Block.Voters()))
			t.manger.FinalBlock() <- *fb
		}
	case transport.FinishLeague:
		var otprnhash common.Hash
		msg.Decode(&otprnhash)
		if otprnhash == leagueOtprnwithsig.Otprn.HashOtprn() {
			//otprn 교체 및 저장된 블록 제거
			t.manger.SetBlockMine(false)
			t.manger.GetStoreOtprnWidthSig()
			t.manger.DelWinningBlock(leagueOtprnwithsig.Otprn.HashOtprn())

			// static node 제거
			for index := range t.leagueList {
				// addPeer 실행
				node, err := discover.ParseNode(t.leagueList[index])
				if err != nil {
					t.logger.Error("노드 URL 파싱에러", "error ", err)
					continue
				}
				t.manger.GetP2PServer().RemovePeer(node)
				t.logger.Debug("DelPeer", "enode", t.leagueList[index])
			}

			return errors.New("리그 종료")
		}
	case transport.RequestWinningBlock:
		if t.manger.GetBlockMine() {
			var headerHash common.Hash
			msg.Decode(&headerHash)
			block := t.manger.GetWinningBlock(leagueOtprnwithsig.Otprn.HashOtprn(), headerHash)
			if block == nil {
				return nil
			}
			fr := &fairtypes.ResWinningBlock{Block: block, OtprnHash: leagueOtprnwithsig.Otprn.HashOtprn()}
			transport.Send(rw, transport.SendWinningBlock, fr.GetTsResWinningBlock())
		}
	case transport.WinningBlockVote:
		if t.manger.GetBlockMine() {
			// 블록 투표 시작
			t.logger.Info("블록 투표 시작")
			t.manger.WinningBlockVoteStart() <- struct{}{}
		}
	default:
		return errors.New(fmt.Sprintf("알수 없는 메시지 코드 : %d", msg.Code))
	}

	return nil
}

func (t *Tcp) checkBalance(otprn *types.Otprn) (bool, *big.Int) {
	currentBalance := t.manger.GetCurrentBalance()
	epoch := big.NewInt(int64(otprn.Epoch))

	// balance will be more then ticket price multiplex epoch.
	price := config.DefaultConfig.SetFee(int64(otprn.Fee))
	totalPrice := big.NewInt(int64(epoch.Uint64() * price.Uint64()))

	return currentBalance.Cmp(totalPrice) > 0, price
}

func (t *Tcp) makeJoinTx(chanID *big.Int, otprn *types.Otprn, sig []byte) error {
	if ok, price := t.checkBalance(otprn); ok {
		currentJoinNonce := t.manger.GetCurrentJoinNonce()
		jtd := types.NewJoinTxData(currentJoinNonce, otprn, sig, t.manger.GetBlockChain().CurrentBlock().Header().Number.Uint64()+1)

		// FIXME(hakuna) : join tx nonce로 변경
		txNonce := t.manger.GetTxpool().State().GetNonce(t.manger.GetCoinbase())

		jtx := types.NewJoinTransaction(txNonce, t.manger.GetBlockChain().Config().Deb.FairAddr, price, jtd)
		signedJoinTx, err := types.SignJoinTx(jtx, t.manger.GetSigner(), t.manger.GetCoinbsePrivKey())
		if err != nil {
			return errorMakeJoinTx
		}

		t.logger.Info("Maked JoinTx", "blockNum", jtd.NextBlockNum, "joinNonce", jtd.JoinNonce, "joinTxHash", signedJoinTx.Hash(), "fee", price)

		////add To txPool // TODO(hakuna) : join txpool로 변경해야함
		//if err := t.manger.GetTxpool().AddLocal(tx); err != nil {
		//	return errorAddTxPool
		//}

		t.logger.Debug("Current Peer", "count", t.manger.GetP2PServer().PeerCount())

	} else {
		// 잔액이 부족한 경우
		// 마이닝을 하지 못함..참여 불가, Dial Close
		return errorLeakCoin
	}

	return nil
}

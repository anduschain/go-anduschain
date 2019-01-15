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
	"github.com/anduschain/go-anduschain/p2p/discover"
	"github.com/anduschain/go-anduschain/rlp"
	"log"
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
	}

	return tcp, nil
}

func (t *Tcp) Start(otprnHash common.Hash) error {

	t.services[otprnHash] = make(map[string]types.Goroutine)
	t.services[otprnHash]["tcploop"] = types.Goroutine{t.tcpLoop, make(chan struct{})}

	if !t.IsRuning[otprnHash] {
		for name, serv := range t.services[otprnHash] {
			log.Println(fmt.Sprintf("Info[andus] : %s Running", name))
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

	defer func() {
		t.IsRuning[otprnHash] = false
		fmt.Println("tcpLoop kill")
	}()

	//conn, err := net.DialTCP("tcp", nil, t.SAddrTCP)
	conn, err := net.Dial(t.SAddrTCP.Network(), t.SAddrTCP.String())
	if err != nil {
		log.Println("Error [andus] : DialTCP 에러", err)
		return
	}

	tsp := transport.New(conn)

	//참가 여부 확인
	transport.Send(tsp, transport.ReqLeagueJoinOK,
		fairtypes.TransferCheck{*t.manger.GetOtprnWithSig(otprnHash).Otprn, t.manger.GetCoinbase(), t.manger.GetP2PServer().NodeInfo().Enode})

	notify := make(chan error)
	go func() {
		for {
			if err := t.handleMsg(tsp, otprnHash); err != nil {
				notify <- err
				return
			}
		}
	}()

Exit:
	for {
		select {
		case err := <-notify:
			log.Println("Error [andus] : handelMsg 에러", err)
			tsp.Close()
			break Exit
		case <-exit:
			tsp.Close()
			break Exit
		case vote := <-t.manger.VoteBlock():
			transport.Send(tsp, transport.SendBlockForVote, vote)
			log.Println(fmt.Printf("Info[andus] block Vote headerHash %s voter %s", vote.HeaderHash.String(), vote.Voter.String()))
		}
	}
}

func (t *Tcp) handleMsg(rw transport.MsgReadWriter, leagueOtprnHash common.Hash) error {
	msg, err := rw.ReadMsg()
	if err != nil {
		return err
	}
	defer msg.Discard()

	var str string
	switch msg.Code {
	case transport.ResLeagueJoinFalse:
		// 참여 불가, Dial Close
		msg.Decode(&str)
		log.Println("Debug[andus] : ", str)
	case transport.MinerLeageStop:
		// 종료됨
		msg.Decode(&str)
		log.Println("Debug[andus] : ", str)
	case transport.SendLeageNodeList:
		// Add peer
		t.manger.SetCurrnetOtprnHash(leagueOtprnHash)
		var nodeList []string
		msg.Decode(&nodeList)
		log.Println("Info[andus] : SendLeageNodeList 수신", len(nodeList))
		for index := range nodeList {
			// addPeer 실행
			node, err := discover.ParseNode(nodeList[index])
			if err != nil {
				fmt.Println("Error[andus] : 노드 url 파싱에러 : ", err)
			}
			t.manger.GetP2PServer().AddPeer(node)
		}
	case transport.MakeJoinTx:
		// JoinTx 생성
		otprnWithSig := t.manger.GetOtprnWithSig(leagueOtprnHash)
		if otprnWithSig == nil {
			return errors.New("해당하는 otprn이 없습니다")
		}

		err := t.makeJoinTx(t.manger.GetBlockChain().Config().ChainID, otprnWithSig.Otprn, otprnWithSig.Sig)
		if err != nil {
			log.Println("Error[andus] : ", err)
			return err
		}
	case transport.MakeBlock:
		otprnWithSig := t.manger.GetOtprnWithSig(leagueOtprnHash)
		if otprnWithSig == nil {
			return errors.New("해당하는 otprn이 없습니다")
		}

		fmt.Println("-------- 블록 생성 tcp -------")
		t.manger.BlockMakeStart() <- struct{}{}

	case transport.SendFinalBlock:
		tsFb := &fairtypes.TsFinalBlock{}
		msg.Decode(&tsFb)
		fb := tsFb.GetFinalBlock()
		block := fb.Block
		fmt.Println("----파이널 블록 수신됨----", common.BytesToHash(block.FairNodeSig).String())
		t.manger.FinalBlock() <- *fb
	case transport.FinishLeague:
		return errors.New("리그 종료")

	case transport.RequestWinningBlock:
		var headerHash common.Hash
		msg.Decode(&headerHash)
		block := t.manger.GetWinningBlock(headerHash)
		if block == nil {
			break
		}

		fr := &fairtypes.ResWinningBlock{Block: block, OtprnHash: leagueOtprnHash}

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
			log.Println("Error[andus] : EncodeToBytes", err)
		}

		txNonce := t.manger.GetTxpool().State().GetNonce(t.manger.GetCoinbase())

		// TODO : andus >> joinNonce Fairnode에게 보내는 Tx
		tx, err := gethTypes.SignTx(
			gethTypes.NewTransaction(txNonce, common.HexToAddress(config.FAIRNODE_ADDRESS),
				config.Price, 90000, big.NewInt(0), joinTxData), t.manger.GetSigner(), t.manger.GetCoinbsePrivKey())
		if err != nil {
			return errorMakeJoinTx
		}

		fmt.Println("----------Tx value------", tx.Value().Uint64())

		// TODO : andus >> txpool에 추가.. 알아서 이더리움 프로세스 타고 날라감....
		if err := t.manger.GetTxpool().AddRemote(tx); err != nil {
			return errorAddTxPool
		}
	} else {
		// 잔액이 부족한 경우
		// 마이닝을 하지 못함..참여 불가, Dial Close
		return errorLeakCoin
	}

	return nil
}

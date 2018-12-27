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
	closeConnection = errors.New("close")
)

type Tcp struct {
	SAddrTCP *net.TCPAddr
	LaddrTCP *net.TCPAddr
	manger   _interface.Client
	services map[string]types.Goroutine
	IsRuning bool
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
		services: make(map[string]types.Goroutine),
		IsRuning: false,
	}

	tcp.services["tcpLoop"] = types.Goroutine{tcp.tcpLoop, make(chan struct{})}

	return tcp, nil
}

func (t *Tcp) Start() error {
	fmt.Println("Tcp 접속 시작", t.IsRuning)
	if !t.IsRuning {
		for name, serv := range t.services {
			log.Println(fmt.Sprintf("Info[andus] : %s Running", name))
			go serv.Fn(serv.Exit)
		}
		t.IsRuning = true
	}

	return nil
}

func (t *Tcp) Stop() error {
	if t.IsRuning {
		for _, srv := range t.services {
			srv.Exit <- struct{}{}
		}
		t.IsRuning = false
	}

	return nil
}

func (t *Tcp) tcpLoop(exit chan struct{}) {
	defer func() {
		t.IsRuning = false
		fmt.Println("tcpLoop kill")
	}()

	conn, err := net.DialTCP("tcp", nil, t.SAddrTCP)
	if err != nil {
		fmt.Println("-------tcp loop 에러", err)
		return
	}

	tsp := transport.New(conn)
	m, err := transport.MakeTsMsg(transport.ReqLeagueJoinOK,
		fairtypes.TransferCheck{*t.manger.GetOtprnWithSig().Otprn, t.manger.GetCoinbase(), t.manger.GetP2PServer().NodeInfo().Enode})
	if err != nil {
		log.Println("Error[andus] : ", err)
	}

	if err := tsp.WriteMsg(m); err != nil {
		// 전송 받은 otprn을 이용해서 참가 여부 확인
		log.Println("Error[andus] : ", err)
	}

	noify := make(chan error)

	go func() {
		defer func() {
			fmt.Println("----------Tcp loop kill--------")
			noify <- closeConnection
		}()

		for {
			fromFaionodeMsg, err := tsp.ReadMsg()
			if err != nil {
				noify <- err
				continue
			}
			var str string
			switch fromFaionodeMsg.Code {
			case transport.ResLeagueJoinFalse:
				// 참여 불가, Dial Close
				fromFaionodeMsg.Decode(&str)
				log.Println("Debug[andus] : ", str)
				return
			case transport.MinerLeageStop:
				// 종료됨
				fromFaionodeMsg.Decode(&str)
				log.Println("Debug[andus] : ", str)
				return
			case transport.SendLeageNodeList:
				// Add peer
				var nodeList []string
				fromFaionodeMsg.Decode(&nodeList)
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
				err := t.makeJoinTx(t.manger.GetBlockChain().Config().ChainID, t.manger.GetOtprnWithSig().Otprn, t.manger.GetOtprnWithSig().Sig)
				if err != nil {
					log.Println("Error[andus] : ", err)
				}
			case transport.MakeBlock:
				fmt.Println("-------- 블록 생성 tcp -------")
				t.manger.BlockMakeStart() <- struct{}{}
			case transport.SendFinalBlock:
				tsFb := &fairtypes.TsFinalBlock{}
				if err := fromFaionodeMsg.Decode(&tsFb); err != nil {
					log.Println("Error[andus] : ", err)
					break
				}

				fb := tsFb.GetFinalBlock()
				if fb.Block == nil {
					return
				}

				block := fb.Block

				if len(block.FairNodeSig) != 0 {
					fmt.Println("----파이널 블록 수신됨----", common.BytesToHash(block.FairNodeSig).String())
					t.manger.FinalBlock() <- *fb
					noify <- closeConnection
				}

			}
		}
	}()

Exit:
	for {
		select {
		case <-time.After(time.Second * 1):
			//fmt.Println("tcp timeout 1, still alive")
		case err := <-noify:
			if "close" == err.Error() {
				conn.Close()
			}
			log.Println("Error[andus] : ------------------- ", err)
		case <-exit:
			conn.Close()
			break Exit
		case winingBlock := <-t.manger.VoteBlock():

			//fmt.Println("----tx len----", winingBlock.Block.Transactions().Len(), len(winingBlock.Receipts))
			fmt.Println("----블록 투표 번호 -----", winingBlock.Block.NumberU64(), winingBlock.Block.Coinbase().String())
			m, err := transport.MakeTsMsg(transport.SendBlockForVote, winingBlock.GetTsVoteBlock())
			if err != nil {
				log.Println("Error[andus] : ", err)
			}
			if err := tsp.WriteMsg(m); err != nil {
				log.Println("Error[andus] : ", err)
			}
		}
	}
}

func (t *Tcp) makeJoinTx(chanID *big.Int, otprn *otprn.Otprn, sig []byte) error {
	// TODO : andus >> JoinTx 생성 ( fairnode를 수신자로 하는 tx, 참가비 보냄...)
	// TODO : andus >> 잔액 조사 ( 임시 : 100 * 10^18 wei ) : 참가비 ( 수수료가 없는 tx )
	// TODO : andus >> joinNonce 현재 상태 조회
	currentBalance := t.manger.GetCurrentBalance()

	if currentBalance.Cmp(config.Price) > 0 {
		currentJoinNonce := t.manger.GetCurrentJoinNonce()
		fmt.Println("chainID : ", chanID.Uint64())
		data := types.JoinTxData{
			OtprnHash:    otprn.HashOtprn(),
			FairNodeSig:  sig,
			TimeStamp:    time.Now(),
			NextBlockNum: t.manger.GetBlockChain().CurrentBlock().Header().Number.Uint64() + 1,
		}

		joinTxData, err := rlp.EncodeToBytes(&data)
		if err != nil {
			log.Println("Error[andus] : EncodeToBytes", err)
		}

		// TODO : andus >> joinNonce Fairnode에게 보내는 Tx
		tx, err := gethTypes.SignTx(
			gethTypes.NewTransaction(currentJoinNonce, common.HexToAddress(config.FAIRNODE_ADDRESS),
				config.Price, 90000, big.NewInt(1000000000), joinTxData), t.manger.GetSigner(), t.manger.GetCoinbsePrivKey())

		if err != nil {
			return errorMakeJoinTx
		}

		fmt.Println("Info[andus] : JoinTx 생성 Success", tx.Hash().String())
		// TODO : andus >> txpool에 추가.. 알아서 이더리움 프로세스 타고 날라감....
		if err := t.manger.GetTxpool().AddLocal(tx); err != nil {
			log.Println("Error[andus] fc.txPool.AddLocal: ", err)
			return errorAddTxPool
		}
	} else {
		// 잔액이 부족한 경우
		// 마이닝을 하지 못함..참여 불가, Dial Close
		return errorLeakCoin
	}

	return nil
}

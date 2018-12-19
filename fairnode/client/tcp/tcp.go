package tcp

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	gethTypes "github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/client/config"
	"github.com/anduschain/go-anduschain/fairnode/client/interface"
	"github.com/anduschain/go-anduschain/fairnode/client/types"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes/msg"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/p2p/discover"
	"github.com/anduschain/go-anduschain/rlp"
	"io"
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
	conn, err := net.DialTCP("tcp", nil, t.SAddrTCP)
	if err != nil {
		return
	}

	// 전송 받은 otprn을 이용해서 참가 여부 확인
	if err := msg.Send(msg.ReqLeagueJoinOK,
		fairtypes.TransferCheck{
			*t.manger.GetOtprnWithSig().Otprn,
			t.manger.GetCoinbase(),
			t.manger.GetP2PServer().NodeInfo().Enode}, conn); err != nil {
		log.Println("Error[andus] : ", err)
	}

	noify := make(chan error)

	go func() {
		data := make([]byte, 4096)
		for {
			n, err := conn.Read(data)
			if err != nil {
				noify <- err
				if err == io.EOF {
					return
				}
				if _, ok := err.(*net.OpError); ok {
					return
				}
			}

			if n > 0 {
				var str string
				fromFaionodeMsg := msg.ReadMsg(data)
				switch fromFaionodeMsg.Code {
				case msg.ResLeagueJoinFalse:
					// 참여 불가, Dial Close
					fromFaionodeMsg.Decode(&str)
					log.Println("Debug[andus] : ", str)
					noify <- closeConnection
				case msg.MinerLeageStop:
					// 종료됨
					fromFaionodeMsg.Decode(&str)
					log.Println("Debug[andus] : ", str)
					noify <- closeConnection
				case msg.SendLeageNodeList:
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
				case msg.MakeJoinTx:
					// JoinTx 생성
					err := t.makeJoinTx(t.manger.GetBlockChain().Config().ChainID, t.manger.GetOtprnWithSig().Otprn, t.manger.GetOtprnWithSig().Sig)
					if err != nil {
						log.Println("Error[andus] : ", err)
					}
				case msg.MakeBlock:
					t.manger.BlockMakeStart() <- struct{}{}
				case msg.SendFinalBlock:
					var received fairtypes.TransferFinalBlock

					if err := fromFaionodeMsg.Decode(&received); err != nil {
						log.Println("Error[andus] : ", err)
					}

					if len(received.EncodedBlock) != 0 {

						stream := rlp.NewStream(bytes.NewReader(received.EncodedBlock), 0)

						block := &gethTypes.Block{}

						if err := block.DecodeRLP(stream); err != nil {
							log.Println("Error[andus] : ", err)
						}

						if len(block.FairNodeSig) != 0 {
							fmt.Println("----파이널 블록 수신됨----", common.BytesToHash(block.FairNodeSig).String())

							t.manger.FinalBlock() <- fairtypes.FinalBlock{block, received.Receipts}
							noify <- closeConnection
						}

					} else {
						noify <- closeConnection
					}
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
			if io.EOF == err {
				fmt.Println("tcp connection dropped message", err)
				conn.Close()
				break Exit
			} else if "close" == err.Error() {
				conn.Close()
			} else if _, ok := err.(*net.OpError); ok {
				fmt.Println("tcp connection dropped message", err)
				break Exit
			}
			log.Println("Error[andus] : ", err)
		case <-exit:
			conn.Close()
		case winingBlock := <-t.manger.VoteBlock():

			fmt.Println("----tx len----", winingBlock.Block.Transactions().Len(), len(winingBlock.Receipts))

			var b bytes.Buffer
			err := winingBlock.Block.EncodeRLP(&b)
			if err != nil {
				fmt.Println("-------인코딩 테스트 에러 ----------", err)
			}

			tsfBlock := &fairtypes.TransferVoteBlock{
				EncodedBlock: b.Bytes(),
				HeaderHash:   winingBlock.HeaderHash,
				Sig:          winingBlock.Sig,
				Voter:        winingBlock.Voter,
				OtprnHash:    winingBlock.OtprnHash,
				Receipts:     winingBlock.Receipts,
			}
			fmt.Println("----블록 투표 번호 -----", winingBlock.Block.NumberU64(), winingBlock.Block.Coinbase().String())
			msg.Send(msg.SendBlockForVote, tsfBlock, conn)
		}
	}

	defer func() {
		t.IsRuning = false
		fmt.Println("tcpLoop kill")
	}()
}

func (t *Tcp) makeJoinTx(chanID *big.Int, otprn *otprn.Otprn, sig []byte) error {
	// TODO : andus >> JoinTx 생성 ( fairnode를 수신자로 하는 tx, 참가비 보냄...)
	// TODO : andus >> 잔액 조사 ( 임시 : 100 * 10^18 wei ) : 참가비 ( 수수료가 없는 tx )
	// TODO : andus >> joinNonce 현재 상태 조회
	currentBalance := t.manger.GetCurrentBalance()

	if currentBalance.Cmp(config.Price) > 0 {
		currentJoinNonce := t.manger.GetCurrentJoinNonce()
		signer := gethTypes.NewEIP155Signer(chanID)

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
				config.Price, 90000, big.NewInt(1000000000), joinTxData), signer, t.manger.GetCoinbsePrivKey())

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

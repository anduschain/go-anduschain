package fairnodeclient

//
//import (
//	"fmt"
//	"github.com/anduschain/go-anduschain/common"
//	"github.com/anduschain/go-anduschain/common/math"
//	"github.com/anduschain/go-anduschain/core/types"
//	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
//	"github.com/anduschain/go-anduschain/fairnode/fairtypes/msg"
//	"github.com/anduschain/go-anduschain/p2p/discover"
//	"github.com/pkg/errors"
//	"log"
//	"math/big"
//	"net"
//	"time"
//)
//
//var (
//	ErrorMakeJoinTx = errors.New("JoinTx 서명 에러")
//	ErrorLeakCoin   = errors.New("잔액 부족으로 마이닝을 할 수 없음")
//	ErrorAddTxPool  = errors.New("TxPool 추가 에러")
//)
//
//func (fc *FairnodeClient) TCPtoFairNode() {
//	log.Println("Info[andus] : TCPtoFairNode 시작")
//
//	fc.wg.Add(1)
//
//	// 연결이 종료 되었을때
//	tcpDisconnectCh := make(chan bool)
//
//	defer log.Println("Info[andus] : TCPtoFairNode 종료됨")
//
//Exit:
//	for {
//		select {
//		case <-fc.TcpConnStartCh:
//			// TODO : andus >> OTPRN이 수신되어 커넥션 만듬
//			if conn, err := net.DialTCP("tcp", nil, fc.SAddrTCP); err == nil {
//				fc.TcpDialer = conn
//				fc.tcpRunning = true
//
//				tsf := fairtypes.TransferCheck{*fc.Otprn, fc.Coinbase, fc.Srv.NodeInfo().Enode}
//				msg.Send(msg.ReqLeagueJoinOK, tsf, conn)
//
//				go fc.tcpLoop(tcpDisconnectCh)
//
//			} else {
//				log.Println("Error[andus] : GETH DialTCP 에러", err)
//				continue
//			}
//		case <-tcpDisconnectCh:
//			// TODO : andus >> 패어노드와 커낵션이 끊어 졌을때
//			fc.TcpDialer.Close()
//			fc.tcpRunning = false
//			log.Println("Error[andus] : TCPtoFairNode 패어노드와 연결이 끊어짐")
//		case <-fc.tcptoFairNodeExitCh:
//			fc.wg.Done()
//			break Exit
//		}
//	}
//
//	fc.wg.Wait()
//
//	//// TODO : andus >> 2. 각 enode값을 이용해서 피어 접속
//	//
//	////enodes := []string{"enode://12121@111.111.111:3303"}
//	////for _ := range enodes {
//	////	old, _ := disco.ParseNode(boot)
//	////	srv.AddPeer(old)
//	////}
//	//
//	//select {
//	//// type : types.TransferBlock
//	//case signedBlock := <-fc.WinningBlockCh:
//	//	// TODO : andus >> 위닝블록 TCP전송
//	//	fmt.Println("위닝 블록을 받아서 페어노드에게 보내요", signedBlock)
//	//}
//
//	// TODO : andus >> 페어노드가 전송한 최종 블록을 받는다.
//	// TODO : andus >> 받은 블록을 검증한다
//	// TODO : andus >> worker에 블록이 등록 되게 한다
//
//	//fc.FinalBlockCh <- &types.TransferBlock{}
//}
//
//func (fc *FairnodeClient) tcpLoop(tcpDisconnectCh chan bool) {
//	defer func() {
//		log.Println("Info[andus] : FairnodeClient tcpLoop 죽음")
//		fc.tcpRunning = false
//	}()
//
//	go func() {
//		defer fmt.Println("--------tcpLoop 내부 go 죽음----------")
//		for {
//			select {
//			case winingBlock := <-fc.WinningBlockCh:
//				fmt.Println("---------------블록 투표----------", winingBlock.Block.Hash().String())
//				msg.Send(msg.SendBlockForVote, winingBlock, fc.TcpDialer)
//			case <-fc.tcpConnStopCh:
//				fc.wg.Done()
//				return
//			default:
//				continue
//			}
//		}
//	}()
//
//	data := make([]byte, 4096)
//	for {
//		if n, err := fc.TcpDialer.Read(data); err == nil {
//			var str string
//
//			if n > 0 {
//				fromFaionodeMsg := msg.ReadMsg(data)
//				switch fromFaionodeMsg.Code {
//				case msg.ResLeagueJoinFalse:
//					// 참여 불가, Dial Close
//					fromFaionodeMsg.Decode(&str)
//					log.Println("Debug[andus] : ", str)
//					tcpDisconnectCh <- true
//					return
//				case msg.MinerLeageStop:
//					// 종료됨
//					fromFaionodeMsg.Decode(&str)
//					log.Println("Debug[andus] : ", str)
//					tcpDisconnectCh <- true
//					return
//				case msg.ResLeagueJoinTrue:
//					// 참여 가능???
//					log.Println("Info[andus] : ", str)
//				case msg.SendLeageNodeList:
//					// Add peer
//					var nodeList []string
//					fromFaionodeMsg.Decode(&nodeList)
//					log.Println("Info[andus] : SendLeageNodeList 수신", len(nodeList))
//
//					for index := range nodeList {
//						// addPeer 실행
//						node, err := discover.ParseNode(nodeList[index])
//						if err != nil {
//							fmt.Println("Error[andus] : 노드 url 파싱에러 : ", err)
//						}
//						fc.Srv.AddPeer(node)
//						log.Println("Info[andus] : 마이너 노드 추가")
//					}
//
//					// JoinTx 생성
//					if err := fc.makeJoinTx(tcpDisconnectCh, fc.BlockChain.Config().ChainID); err != nil {
//						log.Println("Error[andus] : ", err)
//						//return
//					}
//
//					go func() {
//						t := time.NewTicker(10 * time.Second)
//						select {
//						case <-t.C:
//							fc.StartCh <- struct{}{}
//							log.Println("Info[andus] : 블록 생성 시작")
//						}
//					}()
//
//				}
//
//			}
//		} else {
//			if err.Error() == "EOF" {
//				log.Println("INFO : TcpConn ----> EOF")
//				tcpDisconnectCh <- true
//				return
//			} else {
//				log.Println("Error[andus] : readLoop 에러!!!!", err.Error())
//				continue
//			}
//		}
//
//	}
//}
//
//func (fc *FairnodeClient) makeJoinTx(tcpDisconnectCh chan bool, chanID *big.Int) error {
//	// TODO : andus >> JoinTx 생성 ( fairnode를 수신자로 하는 tx, 참가비 보냄...)
//	// TODO : andus >> 잔액 조사 ( 임시 : 100 * 10^18 wei ) : 참가비 ( 수수료가 없는 tx )
//	// TODO : andus >> joinNonce 현재 상태 조회
//	currentBalance := fc.GetCurrentBalance()
//	coin := big.NewInt(TICKET_PRICE)
//	price := coin.Mul(coin, math.BigPow(10, 18))
//	if currentBalance.Cmp(price) > 0 {
//		currentJoinNonce := fc.GetCurrentJoinNonce()
//		log.Println("Info[andus] : JOIN_TX", currentBalance, currentJoinNonce)
//		signer := types.NewEIP155Signer(chanID)
//
//		// TODO : andus >> joinNonce Fairnode에게 보내는 Tx
//		tx, err := types.SignTx(types.NewTransaction(currentJoinNonce, common.HexToAddress(FAIRNODE_ADDRESS), price, 0, big.NewInt(0), []byte("JOIN_TX")), signer, &fc.CoinBasePrivateKey)
//		if err != nil {
//			tcpDisconnectCh <- true
//			return ErrorMakeJoinTx
//		}
//		log.Println("Info[andus] : JoinTx 생성 Success", tx)
//		// TODO : andus >> txpool에 추가.. 알아서 이더리움 프로세스 타고 날라감....
//		if err := fc.txPool.AddLocal(tx); err != nil {
//			log.Println("Error[andus] fc.txPool.AddLocal: ", err)
//			return ErrorAddTxPool
//		}
//	} else {
//		// 잔액이 부족한 경우
//		// 마이닝을 하지 못함..참여 불가, Dial Close
//		tcpDisconnectCh <- true
//		return ErrorLeakCoin
//	}
//
//	return nil
//}

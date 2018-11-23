package fairnodeclient

import (
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/common/math"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes/msg"
	"log"
	"math/big"
	"net"
	"time"
)

func (fc *FairnodeClient) TCPtoFairNode() {
	fmt.Println("andus >> TCPtoFairNode Start")

	fc.wg.Add(1)

	// 연결이 종료 되었을때
	tcpDisconnectCh := make(chan struct{})

	defer func() {
		fc.tcpRunning = false
		fmt.Println("andus >> TCPtoFairNode 죽음")
	}()

Exit:
	for {
		select {
		case <-fc.TcpConnStartCh:
			// TODO : andus >> OTPRN이 수신되어 커넥션 만들
			if conn, err := net.DialTCP("tcp", nil, fc.SAddrTCP); err == nil {
				fc.TcpDialer = conn
				fc.tcpRunning = true
				tsf := fairtypes.TransferCheck{*fc.Otprn, *fc.Coinbase, fc.Srv.NodeInfo().Enode}
				msg.Send(msg.ReqLeagueJoinOK, tsf, conn)

				go fc.tcpLoop(tcpDisconnectCh)

			} else {
				fmt.Println("andus >> GETH DialTCP 에러", err)
				continue
			}
		case <-tcpDisconnectCh:
			// TODO : andus >> 패어노드와 커낵션이 끊어 졌을때
			fc.TcpDialer.Close()
			fc.tcpRunning = false
			fmt.Println("andus >> TCPtoFairNode 패어노드와 연결이 끊어짐")
		case <-fc.tcptoFairNodeExitCh:
			fc.wg.Done()
			break Exit
		}
	}

	fc.wg.Wait()

	//// TODO : andus >> 2. 각 enode값을 이용해서 피어 접속
	//
	////enodes := []string{"enode://12121@111.111.111:3303"}
	////for _ := range enodes {
	////	old, _ := disco.ParseNode(boot)
	////	srv.AddPeer(old)
	////}
	//
	//select {
	//// type : types.TransferBlock
	//case signedBlock := <-fc.WinningBlockCh:
	//	// TODO : andus >> 위닝블록 TCP전송
	//	fmt.Println("위닝 블록을 받아서 페어노드에게 보내요", signedBlock)
	//}

	// TODO : andus >> 페어노드가 전송한 최종 블록을 받는다.
	// TODO : andus >> 받은 블록을 검증한다
	// TODO : andus >> worker에 블록이 등록 되게 한다

	//fc.FinalBlockCh <- &types.TransferBlock{}
}

func (fc *FairnodeClient) tcpLoop(tcpDisconnectCh chan struct{}) {
	defer func() {
		fmt.Println("andus >> FairnodeClient tcpLoop 죽음")

	}()

	data := make([]byte, 4096)
	for {
		select {
		case <-fc.tcpConnStopCh:
			fc.wg.Done()
			return
		default:
			fc.TcpDialer.SetDeadline(time.Now().Add(3 * time.Second))
			n, err := fc.TcpDialer.Read(data)
			if err != nil {
				fmt.Println("andus >> Read 에러!!", err.Error())
				if err.Error() != "EOF" {
					if err.(net.Error).Timeout() {
						continue
					}
				} else {
					tcpDisconnectCh <- struct{}{}
					return
				}
			}

			if n > 0 {
				fromFaionodeMsg := msg.ReadMsg(data)
				switch fromFaionodeMsg.Code {
				case msg.ResLeagueJoinFalse:
					// 참여 불가, Dial Close
					tcpDisconnectCh <- struct{}{}
					return
				case msg.ResLeagueJoinTrue:
					// 참여 가능
					// TODO : andus >> JoinTx 생성 ( fairnode를 수신자로 하는 tx, 참가비 보냄...)
					// TODO : andus >> 잔액 조사 ( 임시 : 100 * 10^18 wei ) : 참가비 ( 수수료가 없는 tx )

					// TODO : andus >> joinNonce 현재 상태 조회
					currentBalance := fc.GetCurrentBalance()

					coin := big.NewInt(TICKET_PRICE)
					price := coin.Mul(coin, math.BigPow(10, 18))
					if currentBalance.Cmp(price) > 0 {
						currentJoinNonce := fc.GetCurrentJoinNonce()

						fmt.Println("andus >> JOIN_TX", currentBalance, currentJoinNonce)

						signer := types.NewEIP155Signer(big.NewInt(100)) // chainID 변경해야함..

						// TODO : andus >> joinNonce Fairnode에게 보내는 Tx
						tx, err := types.SignTx(types.NewTransaction(currentJoinNonce, common.HexToAddress(FAIRNODE_ADDRESS), price, 0, big.NewInt(0), []byte("JOIN_TX")), signer, fc.CoinBasePrivateKey)
						if err != nil {
							log.Println("andus >> JoinTx 서명 에러")
						}

						log.Println("andus >> JoinTx 생성 Success", tx)

						// TODO : andus >> txpool에 추가.. 알아서 이더리움 프로세스 타고 날라감....
						fc.txPool.AddLocal(tx)
					} else {
						// 잔액이 부족한 경우
						// 마이닝을 하지 못함..참여 불가, Dial Close
						tcpDisconnectCh <- struct{}{}
						fmt.Println("andus >> 잔액 부족으로 마이닝을 할 수 없음")
						return
					}

				}

			}
		}

	}
}

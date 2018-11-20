package fairnodeclient

import (
	"fmt"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/rlp"
	"net"
	"time"
)

func (fc *FairnodeClient) TCPtoFairNode() {
	fmt.Println("andus >> TCPtoFairNode Start")

	fc.wg.Add(2)
	// 리스너가 종료 되었을때
	tcpDisconnectCh := make(chan struct{})
	// 리스너가 종료되고 다음 연결을 기다릴때
	TCPtoFairNodePendingCh := make(chan struct{})
	// 외부에서 마이닝 miner.stop() 실행시
	fromExitCh := make(chan struct{})

	defer func() {
		fc.tcpRunning = false
		fmt.Println("andus >> TCPtoFairNode 죽음")
	}()

EXIT:
	for {
		select {
		case <-fc.TcpConnStartCh:
			// TODO : andus >> OTPRN이 수신되어 커넥션 만들
			if conn, err := net.DialTCP("tcp", nil, fc.SAddrTCP); err == nil {
				fc.TcpDialer = conn
				fc.tcpRunning = true
				fmt.Println("andus >> TCPtoFairNode 채녈 들어옴")

				go fc.readLoop(tcpDisconnectCh)
				go fc.writeLoop(tcpDisconnectCh, TCPtoFairNodePendingCh, fromExitCh)

			} else {
				fmt.Println("andus >> GETH DialTCP 에러", err)
				continue
			}
		case <-TCPtoFairNodePendingCh:
			// TODO : andus >> 패어노드와 커낵션이 끊어 졌을때
			fc.TcpDialer.Close()
			fc.tcpRunning = false
			fmt.Println("andus >> TCPtoFairNode 패어노드와 연결이 끊어짐")

		case <-fc.tcptoFairNodeExitCh:
			// TODO : andus >> 패어노드와 커낵션이 없을때
			return
		case <-fromExitCh:
			break EXIT
		}
	}

	fc.wg.Wait()

	////TODO : andus >> TCP 통신 to FairNode
	////TODO : andus >> 1. fair Node에 TCP 연결
	////TODO : andus >> 2. OTPRN, enode값 전달
	//
	//// TODO : andus >> 1. 채굴 리스 리스트와 총 채굴리그 해시 수신
	//
	//// TODO : andus >> 1.1 추후 서명값 검증 해야함...
	//
	//// TODO : andus >> 4. JoinTx 생성 ( fairnode를 수신자로 하는 tx, 참가비 보냄...)
	//
	//var fairNodeAddr common.Address // TODO : andus >> 보내는 fairNode의 Address(주소)
	//
	//// TODO : andus >> joinNonce 현재 상태 조회
	//
	//currentJoinNonce := fc.GetCurrentJoinNonce()
	//
	//signer := types.NewEIP155Signer(big.NewInt(18))
	//
	//// TODO : andus >> joinNonce Fairnode에게 보내는 Tx
	//tx, err := types.SignTx(types.NewTransaction(currentJoinNonce, fairNodeAddr, new(big.Int), 0, new(big.Int), nil), signer, fc.CoinBasePrivateKey)
	//if err != nil {
	//	log.Println("andus >> JoinTx 서명 에러")
	//}
	//
	//log.Println("andus >> JoinTx 생성 Success", tx)
	//
	//// TODO : andus >> txpool에 추가.. 알아서 이더리움 프로세스 타고 날라감....
	//fc.txPool.AddLocal(tx)
	//
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

func (fc *FairnodeClient) readLoop(tcpDisconnectCh chan struct{}) {
	defer func() {
		fmt.Println("andus >> FairnodeClient ReadLoop 죽음")
	}()

	data := make([]byte, 4096)
	for {
		select {
		case <-fc.readLoopStopCh:
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
				fmt.Println("andus >> sendFairnodeData 수신")
				fmt.Println(string(data[:n]))
			}
		}

	}
}

func (fc *FairnodeClient) writeLoop(tcpDisconnectCh chan struct{}, pendingCh chan struct{}, exitCh chan struct{}) {

	defer fmt.Println("andus >> FairnodeClient writeLoop 죽음")

	for {
		select {
		case <-tcpDisconnectCh:
			pendingCh <- struct{}{}
			return
		case <-fc.writeLoopStopCh:
			exitCh <- struct{}{}
			fc.wg.Done()
			return
		default:
			fmt.Println("andus >> sendFairnodeData 전송")
			tsf := fairtypes.TransferCheck{*fc.Otprn, *fc.Coinbase, *fc.Enode}

			sendFairnodeData, err := rlp.EncodeToBytes(tsf)
			if err != nil {
				fmt.Println("andus >> sendFairnodeData 에러", err)
			}
			fc.TcpDialer.Write(sendFairnodeData)
			time.Sleep(1 * time.Second)
		}
	}
}

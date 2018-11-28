package server

import (
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes/msg"
	"github.com/anduschain/go-anduschain/fairnode/fairutil"
	"log"
	"net"
	"time"
)

func (f *FairNode) ListenTCP() {

	log.Println("ListenTCP 리슨TCP")

	go f.sendLeague()

	for {
		conn, err := f.TcpConn.AcceptTCP()
		if err != nil {
			log.Println("Error : f.TcpConn.Accept 에러!!", err)
		}

		go f.tcpLoop(conn)
	}

	//
	//// TODO : andus >> 위닝블록이 수신되는 곳 >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//// TODO : andus >> 1. 수신 블록 검증 ( sig, hash )
	//// TODO : andus >> 2. 검증된 블록을 MongoDB에 저장 ( coinbase, RAND, 보낸 놈, 블록 번호 )
	//// TODO : andus >> 3. 기다림........
	//// TODO : andus >> 3-1 채굴참여자 수 만큼 투표가 진행 되거나, 아니면 15초후에 투표 종료
	//
	//count := 0
	//leagueCount := 10 // TODO : andus >> 리그 채굴 참여자
	//endOfLeagueCh := make(chan interface{})
	//
	//if count == leagueCount {
	//	endOfLeagueCh <- "보내.."
	//}
	//
	//t := time.NewTicker(15 * time.Second)
	//log.Println(" @ go func() START !! ")
	//
	//go func() {
	//	for {
	//		select {
	//		case <-t.C:
	//			// TODO : andus >> 투표 결과 서명해서, TCP로 보내준다
	//			// TODO : andus >> types.TransferBlock{}의 타입으로 전송할것..
	//			// TODO : andus >> 받은 블록의 블록헤더의 해시를 이용해서 서명후, FairNodeSig에 넣어서 보낼것.
	//			// TODO : andus >> 새로운 리그 시작
	//			f.LeagueRunningOK = false
	//
	//		case <-endOfLeagueCh:
	//			// TODO : andus >> 투표 결과 서명해서, TCP로 보내준다
	//			// TODO : andus >> types.TransferBlock{}의 타입으로 전송할것..
	//			// TODO : andus >> 받은 블록의 블록헤더의 해시를 이용해서 서명후, FairNodeSig에 넣어서 보낼것.
	//			f.LeagueRunningOK = false
	//
	//		}
	//	}
	//}()
}

func (f *FairNode) tcpLoop(conn *net.TCPConn) {
	buf := make([]byte, 4096)
	for {
		select {
		case nodeList := <-f.sendLeagueCh:
			log.Println("INFO : 리드에 참여한 노드 리스트 전송", len(nodeList))
			msg.Send(msg.SendLeageNodeList, nodeList, conn)
		default:
			if n, err := conn.Read(buf); err == nil {
				if n > 0 {
					fromGethMsg := msg.ReadMsg(buf)
					switch fromGethMsg.Code {
					case msg.ReqLeagueJoinOK:
						var tsf fairtypes.TransferCheck
						fromGethMsg.Decode(&tsf)
						log.Println("INFO : CheckEnodeAndCoinbse", tsf.Enode, tsf.Coinbase.String())
						if f.Db.CheckEnodeAndCoinbse(tsf.Enode, tsf.Coinbase.String()) {
							// TODO : andus >> 1. Enode가 맞는지 확인 ( 조회 되지 않으면 팅김 )
							// TODO : andus >> 2. 해당하는 Enode가 이전에 보낸 코인베이스와 일치하는지

							if fairutil.IsJoinOK(tsf.Otprn, tsf.Coinbase) {
								// TODO : 채굴 리그 생성
								// TODO : 1. 채굴자 저장 ( key otprn num, Enode의 ID를 저장....)
								f.Db.SaveMinerNode(tsf.Otprn.HashOtprn().String(), tsf.Enode)
								msg.Send(msg.ResLeagueJoinTrue, "리그참여 대상자가 맞습니다", conn)
								f.sendLeagueStartCh <- tsf.Otprn.HashOtprn().String()
							} else {
								// TODO : andus >> 참여 대상자가 아니다
								msg.Send(msg.ResLeagueJoinFalse, "리그참여 대상자가 아님", conn)
								conn.Close() // 커넥션 종료
								return
							}
						} else {
							// TODO : andus >> 리그 참여 정보가 다르다
							msg.Send(msg.ResLeagueJoinFalse, "리그참여 대상자가 아님", conn)
							conn.Close() // 커넥션 종료
							return
						}
					}

				}
			} else {
				if err.Error() == "EOF" {
					conn.Close() // 커넥션 종료
					return
				} else {
					log.Println("Error : readLoop 에러!!!!", err.Error())
					continue
				}
			}
		}
	}

}

func (f *FairNode) sendLeague() {
	var temOtprnHash string
	var tick *time.Ticker

	for {
		select {
		case <-tick.C:
			//15 간격으로 호출
			nodeList := f.Db.GetMinerNode(temOtprnHash)
			if len(nodeList) >= 3 {
				f.sendLeagueCh <- nodeList
				tick.Stop()
			} else {
				continue
			}
		case otprnHash := <-f.sendLeagueStartCh:
			if temOtprnHash != otprnHash {
				temOtprnHash = otprnHash
				tick = time.NewTicker(15 * time.Second)
			} else {
				continue
			}
		}
	}
}

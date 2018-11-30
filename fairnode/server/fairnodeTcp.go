package server

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes/msg"
	"github.com/anduschain/go-anduschain/fairnode/fairutil"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"log"
	"net"
	"time"
)

func (f *FairNode) ListenTCP() {

	log.Println("ListenTCP 리슨TCP")

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
	//log.Println(" @ go func() START !! ")                                                                                                                                        ㅁ
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
	log.Println("INFO[andus] : tcpLoop 시작", conn.RemoteAddr().String())

	defer conn.Close()
	defer log.Println("INFO[andus] : tcpLoop 종료", conn.RemoteAddr().String())

	buf := make([]byte, 4096)
	var coinbase string

Exit:
	for {
		if n, err := conn.Read(buf); err == nil {
			if n > 0 {
				fromGethMsg := msg.ReadMsg(buf)
				switch fromGethMsg.Code {
				case msg.ReqLeagueJoinOK:
					var tsf fairtypes.TransferCheck
					fromGethMsg.Decode(&tsf)
					log.Println("INFO[andus] : 접속한 COINBASE", tsf.Coinbase.String())

					// Tcp 접속 pool에 저장
					f.LeagueConPool[tsf.Coinbase.String()] = conn
					coinbase = tsf.Coinbase.String()

					if f.Db.CheckEnodeAndCoinbse(tsf.Enode, tsf.Coinbase.String()) {
						// TODO : andus >> 1. Enode가 맞는지 확인 ( 조회 되지 않으면 팅김 )
						// TODO : andus >> 2. 해당하는 Enode가 이전에 보낸 코인베이스와 일치하는지

						if fairutil.IsJoinOK(tsf.Otprn, tsf.Coinbase) {
							// TODO : 채굴 리그 생성
							// TODO : 1. 채굴자 저장 ( key otprn num, Enode의 ID를 저장....)
							otprnHash := tsf.Otprn.HashOtprn().String()
							nodes, _ := f.GetLeaguePool(otprnHash)

							if otprn.Mminer > uint64(len(nodes)) {
								msg.Send(msg.ResLeagueJoinTrue, "리그참여 대상자가 맞습니다", conn)
								f.Db.SaveMinerNode(otprnHash, tsf.Enode)
								f.LeaguePoolInsert(otprnHash, tsf.Enode)
								//log.Println("INFO : 리그 참여자 TCP 연결 후 저장됨", tsf.Enode)
							} else {
								// TODO : 참여 인원수 오버된 케이스
								f.closeTcpConn(conn, coinbase)
								log.Println("INFO : 참여 인원수 오버된 케이스", tsf.Enode)
								break Exit
							}
						} else {
							// TODO : andus >> 참여 대상자가 아니다
							f.closeTcpConn(conn, coinbase)
							log.Println("INFO : 참여 대상자가 아니다", tsf.Enode)
							break Exit
						}
					} else {
						// TODO : andus >> 리그 참여 정보가 다르다
						f.closeTcpConn(conn, coinbase)
						log.Println("INFO : 리그 참여 정보가 다르다", tsf.Enode)
						break Exit
					}
				}

			}
		} else {
			if err.Error() == "EOF" {
				f.closeTcpConn(conn, coinbase) // 커넥션 종료
				log.Println("INFO : TcpConn ----> EOF")
				break Exit
			} else {
				log.Println("Error : readLoop 에러!!!!", err.Error())
				continue
			}
		}
	}

	log.Println("INFO[andus] : tcpLoop Exit")

}

// Tcp 접속 pool에 삭제 후 커넥션 종료
func (f *FairNode) closeTcpConn(conn *net.TCPConn, coinbase string) {
	msg.Send(msg.ResLeagueJoinFalse, "리그참여 대상자가 아님", conn)
	if _, ok := f.LeagueConPool[coinbase]; ok {
		delete(f.LeagueConPool, coinbase)
	}
	log.Println("Info[andus] : Tcp 접속 pool에 삭제 후 커넥션 종료")
}

func (f *FairNode) sendLeague(otprnHash string) {
	defer log.Println("Debug[andus] : sendLeague 죽음")
	t := time.NewTicker(15 * time.Second)
	for {
		select {
		case <-t.C:
			nodeList, index := f.GetLeaguePool(otprnHash)
			// 가능한 사람의 30%이상일때 접속할 채굴 리그를 전송해줌
			if len(nodeList) >= f.JoinTotalNum(30) {
				for coinbase, conn := range f.LeagueConPool {
					msg.Send(msg.SendLeageNodeList, nodeList, conn)
					log.Println("Debug : 노드 리스트 보냄 : ", coinbase)
				}
				f.DeleteLeaguePool(index)
				return
			} else {
				log.Println("Debug : 리그가 성립 안됨 연결 새로운 리그 시작 : ", len(nodeList))

				for coinbase, conn := range f.LeagueConPool {
					msg.Send(msg.MinerLeageStop, "리그가 종료 되었습니다", conn)
					log.Println("Debug : 노드 리스트 보냄 : ", coinbase)
				}

				f.LeagueRunningOK = false
				f.DeleteLeaguePool(index)
				return
			}
		}
	}
}

func (f *FairNode) JoinTotalNum(persent float64) int {
	f.lock.Lock()
	defer f.lock.Unlock()
	aciveNode := f.Db.GetActiveNodeList()
	var count float64 = 0
	for i := range aciveNode {
		if fairutil.IsJoinOK(*f.otprn, common.HexToAddress(aciveNode[i].Coinbase)) {
			count += 1
		}
	}

	log.Println("Info[andus] : JoinTotalNum의 30프로 이상일때 가능", count, count*(persent/100), int(count*(persent/100)))
	return int(count)
}

func (f *FairNode) LeaguePoolInit(otprnHash string) {
	m := make(connNode)
	m[otprnHash] = []string{}
	f.ConnNodePool = append(f.ConnNodePool, m)
}

func (f *FairNode) LeaguePoolInsert(otprnHash string, enode string) {
	_, index := f.GetLeaguePool(otprnHash)
	f.ConnNodePool[index][otprnHash] = append(f.ConnNodePool[index][otprnHash], enode)
}

func (f *FairNode) DeleteLeaguePool(index int) {
	m := f.ConnNodePool
	m = append(m[:index], m[index+1:]...)
}

func (f *FairNode) GetLeaguePool(otprnHash string) ([]string, int) {
	for index := range f.ConnNodePool {
		if _, ok := f.ConnNodePool[index][otprnHash]; ok {
			return f.ConnNodePool[index][otprnHash], index
		}
	}
	return []string{}, -1
}

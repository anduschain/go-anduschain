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

		f.LeagueConList = append(f.LeagueConList, conn)
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
	defer log.Println("INFO[andus] : tcpLoop 종료", conn.RemoteAddr().String())
	buf := make([]byte, 4096)
	for {
		if n, err := conn.Read(buf); err == nil {
			if n > 0 {
				fromGethMsg := msg.ReadMsg(buf)
				switch fromGethMsg.Code {
				case msg.ReqLeagueJoinOK:
					var tsf fairtypes.TransferCheck
					fromGethMsg.Decode(&tsf)
					log.Println("INFO[andus] : 접속한 COINBASE", tsf.Coinbase.String())
					if f.Db.CheckEnodeAndCoinbse(tsf.Enode, tsf.Coinbase.String()) {
						// TODO : andus >> 1. Enode가 맞는지 확인 ( 조회 되지 않으면 팅김 )
						// TODO : andus >> 2. 해당하는 Enode가 이전에 보낸 코인베이스와 일치하는지

						if fairutil.IsJoinOK(tsf.Otprn, tsf.Coinbase) {
							// TODO : 채굴 리그 생성
							// TODO : 1. 채굴자 저장 ( key otprn num, Enode의 ID를 저장....)
							otprnHash := tsf.Otprn.HashOtprn().String()
							if otprn.Mminer > f.Db.GetMinerNodeNum(otprnHash) {
								f.Db.SaveMinerNode(otprnHash, tsf.Enode)
								msg.Send(msg.ResLeagueJoinTrue, "리그참여 대상자가 맞습니다", conn)
								log.Println("INFO : 리그 참여자 TCP 연결 후 저장됨", tsf.Enode)
							} else {
								// TODO : 참여 인원수 오버된 케이스
								msg.Send(msg.ResLeagueJoinFalse, "리그참여 대상자가 아님", conn)
								conn.Close() // 커넥션 종료
								return
							}
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
				//log.Println("Error : readLoop 에러!!!!", err.Error())
				continue
			}
		}
	}
}

func (f *FairNode) sendLeague(otprnHash string) {
	defer log.Println("Debug[andus] : sendLeague 죽음")
	t := time.NewTicker(15 * time.Second)
	for {
		select {
		case <-t.C:
			nodeList := f.Db.GetMinerNode(otprnHash)
			// 가능한 사람의 30%이상일때 접속할 채굴 리그를 전송해줌
			if len(nodeList) >= f.JoinTotalNum(30) {
				for i := range f.LeagueConList {
					if f.LeagueConList[i] != nil {
						msg.Send(msg.SendLeageNodeList, nodeList, f.LeagueConList[i])
						log.Println("Debug : 노드 리스트 보냄 : ", len(nodeList))
					}
				}
				return
			} else {
				log.Println("Debug : 노드 리스트 PASS : ", len(nodeList))
				f.LeagueRunningOK = false
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

//func (f *FairNode) LeagueInsert(otprnHash string, enode string) {
//	nodeList, index := f.GetLeague(otprnHash)
//	if len(nodeList) > 0 {
//		f.LeagueList[index][otprnHash] = append(f.LeagueList[index][otprnHash], enode)
//	} else {
//		m := make(map[string][]string)
//		m[otprnHash] = []string{enode}
//		f.LeagueList = append(f.LeagueList, m)
//	}
//}
//
//func (f *FairNode) DeleteLeague(index int) {
//	m := f.LeagueList
//	m = append(m[:index], m[index+1:]...)
//}
//
//func (f *FairNode) GetLeague(otprnHash string) ([]string, int) {
//	for index := range f.LeagueList {
//		if _, ok := f.LeagueList[index][otprnHash]; ok {
//			return f.LeagueList[index][otprnHash], index
//		}
//	}
//	return []string{}, -1
//}

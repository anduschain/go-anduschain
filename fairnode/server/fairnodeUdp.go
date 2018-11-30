package server

import (
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes/msg"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"log"
	"net"
	"time"
)

func (f *FairNode) ListenUDP() {

	go f.manageActiveNode()
	// TODO : andus >> otprn 생성, 서명, 전송
	go f.startLeague()
	//go f.makeLeague()
}

// 활성 노드 관리 ( upd enode 수신, 저장, 업데이트 )
func (f *FairNode) manageActiveNode() {
	// TODO : andus >> Geth node Heart beat update ( Active node 관리 )
	// TODO : enode값 수신
	buf := make([]byte, 4096)
	t := time.NewTicker(3 * time.Minute)
	for {
		select {
		case <-t.C:
			// TODO : andus >> 3분이상 들어오지 않은 enode 지우기 (mongodb)
			f.Db.JobCheckActiveNode()
		default:
			f.UdpConn.SetReadDeadline(time.Now().Add(3 * time.Second))
			n, _, err := f.UdpConn.ReadFromUDP(buf)
			if err != nil {
				//log.Println("ReadFromUDP 에러", err)
				if err.(net.Error).Timeout() {
					continue
				}
			}

			if n > 0 {
				fromGethMsg := msg.ReadMsg(buf)
				switch fromGethMsg.Code {
				case msg.SendEnode:
					var fromGeth fairtypes.EnodeCoinbase
					fromGethMsg.Decode(&fromGeth)
					f.Db.SaveActiveNode(fromGeth.Enode, fromGeth.Coinbase, fromGeth.Port)
				}
			}
		}
	}
}

// otprn 생성, 서명, 전송 ( 3초 반복, active node >= 3, LeagueRunningOK == false // 고루틴 )
func (f *FairNode) startLeague() {
	t := time.NewTicker(3 * time.Second)
	for {
		select {
		case <-t.C:
			actNum := f.Db.GetActiveNodeNum()
			if !f.LeagueRunningOK && actNum >= 3 {

				activeNodeNum := uint64(f.Db.GetActiveNodeNum())
				f.otprn = otprn.New(activeNodeNum)

				// TODO : andus >> otprn을 서명
				sig, err := f.otprn.SignOtprn(f.Account, f.otprn.HashOtprn(), f.Keystore)
				if err != nil {
					log.Println("Otprn 서명 에러", err)
				}

				tsOtp := fairtypes.TransferOtprn{
					Otp:  *f.otprn,
					Sig:  sig,
					Hash: f.otprn.HashOtprn(),
				}
				// andus >> OTPRN DB 저장
				f.Db.SaveOtprn(tsOtp)

				if activeNodeNum > 0 {
					f.LeagueRunningOK = true
					activeNodeList := f.Db.GetActiveNodeList()

					go f.sendLeague(tsOtp.Hash.String())

					for index := range activeNodeList {
						url := activeNodeList[index].Ip + ":" + activeNodeList[index].Port
						ServerAddr, err := net.ResolveUDPAddr("udp", url)
						if err != nil {
							log.Println("ResolveUDPAddr", err)
						}
						Conn, err := net.DialUDP("udp", nil, ServerAddr)
						if err != nil {
							log.Println("DialUDP", err)
						}
						msg.Send(msg.SendOTPRN, tsOtp, Conn)
						Conn.Close()
					}
				}
			}
		}
	}
}

//func (f *FairNode) makeLeague() {
//	t := time.NewTicker(1 * time.Second)
//	for {
//		select {
//		case <-t.C:
//
//			log.Println(" @ in makeLeague() ")
//		}
//		// <- chan Start singnal // 레그 스타트
//
//		// TODO : andus >> 리그 스타트 ( 엑티브 노드 조회 ) ->
//
//		// TODO : andus >> 1. OTPRN 생성
//		// TODO : andus >> 2. OTPRN Hash
//		// TODO : andus >> 3. Fair Node 개인키로 암호화
//		// TODO : andus >> 4. OTPRN 값 + 전자서명값 을 전송
//		// TODO : andus >> 5. UDP 전송
//		// TODO : andus >> 6. UDP 전송 후 참여 요청 받을 때 까지 기다릴 시간( 3s )후
//		// TODO : andus >> 7. 리스 시작 채널에 메세지 전송
//		//bb <- "리그시작"
//
//		// close(Start singnal)
//	}
//}

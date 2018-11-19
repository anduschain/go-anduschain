package server

import (
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/fairnode/client"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/rlp"
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
	for {
		n, addr, err := f.UdpConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("andus >>", err)
		}

		if n > 0 {
			// TODO : andus >> rlp enode 디코드
			var fromGeth fairnodeclient.EnodeCoinbase
			rlp.DecodeBytes(buf, &fromGeth)
			f.Db.SaveActiveNode(fromGeth.Node, addr, fromGeth.Coinbase)
		}

	}
}

// otprn 생성, 서명, 전송 ( 3초 반복, active node >= 3, LeagueRunningOK == false // 고루틴 )
func (f *FairNode) startLeague() {
	var otp *otprn.Otprn
	var err error
	t := time.NewTicker(3 * time.Second)
	for {
		select {
		case <-t.C:
			actNum := f.Db.GetActiveNodeNum()
			if !f.LeagueRunningOK && actNum >= 3 {
				// TODO : andus >> otprn을 생성

				log.Println("andus >> otprn 생성")

				otp, err = otprn.New(11)
				if err != nil {
					log.Println("andus >> Otprn 생성 에러", err)
				}

				//f.LeagueRunningOK = true

				// TODO : andus >> otprn을 서명
				sig, err := otp.SignOtprn(f.Account, otp.HashOtprn(), f.Keystore)
				if err != nil {
					log.Println("andus >> Otprn 서명 에러", err)
				}

				fmt.Println("andus >> sig 값", common.BytesToHash(sig).String())

				tsOtp := otprn.TransferOtprn{
					Otp:  *otp,
					Sig:  sig,
					Hash: otp.HashOtprn(),
				}

				ts, err := rlp.EncodeToBytes(tsOtp)
				if err != nil {
					log.Println("andus >> Otprn rlp 인코딩 에러", err)
				}

				activeNodeList := f.Db.GetActiveNodeList()
				for index := range activeNodeList {
					ServerAddr, err := net.ResolveUDPAddr("udp", activeNodeList[index])
					if err != nil {
						log.Println("andus >>", err)
					}
					Conn, err := net.DialUDP("udp", nil, ServerAddr)
					if err != nil {
						log.Println("andus >>", err)
					}

					Conn.Write(ts)
					Conn.Close()
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

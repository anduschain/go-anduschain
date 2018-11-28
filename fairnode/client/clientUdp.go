package fairnodeclient

import (
	"fmt"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes/msg"
	"github.com/anduschain/go-anduschain/fairnode/fairutil"
	"github.com/anduschain/go-anduschain/p2p/nat"
	"log"
	"net"
	"time"
)

func (fc *FairnodeClient) UDPtoFairNode() {
	fc.wg.Add(2)

	defer fmt.Println("andus >> UDPtoFairNode 죽음 >>>>>")
	//TODO : andus >> udp 통신 to FairNode
	go fc.submitEnode()
	go fc.receiveOtprn()

	fc.wg.Wait()
}

func (fc *FairnodeClient) submitEnode() {
	// TODO : andus >> FairNode IP : localhost UDP Listener 11/06 -- start --
	Conn, err := net.DialUDP("udp", nil, fc.SAddrUDP)
	if err != nil {
		log.Println("andus >> UDPtoFairNode, DialUDP", err)
	}

	defer Conn.Close()
	defer fc.wg.Done()

	// TODO : andus >> FairNode IP : localhost UDP Listener 11/06 -- end --
	t := time.NewTicker(60 * time.Second)
	ts := fairtypes.EnodeCoinbase{fc.Srv.NodeInfo().Enode, *fc.Coinbase, DefaultConfig.ClientPort}

	if err := msg.Send(msg.SendEnode, ts, Conn); err != nil {
		fmt.Println("andus >>>>>>", err)
	}

	for {
		select {
		case <-t.C:
			//TODO : andus >> FairNode에게 enode값 전송 ( 1분단위)
			// TODO : andus >> enode Sender -- start --
			fmt.Println("andus >> Enode 전송")
			msg.Send(msg.SendEnode, ts, Conn)
		case <-fc.submitEnodeExitCh:
			fmt.Println("andus >> submitEnode 종료됨")
			return
		}
	}
}

func (fc *FairnodeClient) receiveOtprn() {

	//TODO : andus >> 1. OTPRN 수신

	localServerConn, err := net.ListenUDP("udp", fc.LAddrUDP)
	if err != nil {
		log.Println("Udp Server", err)
	}

	// TODO : andus >> NAT 추가 --- start ---

	natm, err := nat.Parse(fc.NAT)
	if err != nil {
		log.Fatalf("-nat: %v", err)
	}

	realaddr := localServerConn.LocalAddr().(*net.UDPAddr)
	if natm != nil {
		if !realaddr.IP.IsLoopback() {
			go nat.Map(natm, nil, "udp", realaddr.Port, realaddr.Port, "andus fairnode discovery")
		}
		// TODO: react to external IP changes over time.
		if ext, err := natm.ExternalIP(); err == nil {
			realaddr = &net.UDPAddr{IP: ext, Port: realaddr.Port}
		}
	}

	// TODO : andus >> NAT 추가 --- end ---

	defer localServerConn.Close()
	defer fc.wg.Done()

	tsOtprnByte := make([]byte, 4096)

	for {
		select {
		case <-fc.receiveOtprnExitCh:
			fmt.Println("andus >> receiveOtprn 종료됨")
			return
		default:
			localServerConn.SetReadDeadline(time.Now().Add(3 * time.Second))
			n, _, err := localServerConn.ReadFromUDP(tsOtprnByte)
			if err != nil {
				log.Println("Debug : Connect Error", err)
				if err.(net.Error).Timeout() {
					continue
				}
			}

			if n > 0 {
				// TODO : andus >> 수신된 otprn디코딩

				fromFairnodeMsg := msg.ReadMsg(tsOtprnByte)
				switch fromFairnodeMsg.Code {
				case msg.SendOTPRN:
					var tsOtprn fairtypes.TransferOtprn
					fromFairnodeMsg.Decode(&tsOtprn)

					log.Println("Debug : OTPRN 수신됨", tsOtprn.Hash.String())

					//TODO : andus >> 2. OTRRN 검증
					fairPubKey, err := crypto.SigToPub(tsOtprn.Hash.Bytes(), tsOtprn.Sig)
					if err != nil {
						log.Println("andus >> OTPRN 공개키 로드 에러")
					}

					if crypto.VerifySignature(crypto.FromECDSAPub(fairPubKey), tsOtprn.Hash.Bytes(), tsOtprn.Sig[:64]) {
						otprnHash := tsOtprn.Otp.HashOtprn()
						if otprnHash == tsOtprn.Hash {
							// TODO: andus >> 검증완료, Otprn 저장
							fc.Otprn = &tsOtprn.Otp
							//TODO : andus >> 3. 참여여부 확인

							//fmt.Println("andus >> OTPRN 검증 완료")

							if ok := fairutil.IsJoinOK(*fc.Otprn, *fc.Coinbase); ok {
								//TODO : andus >> 참가 가능할 때 처리
								//TODO : andus >> 6. TCP 연결 채널에 메세지 보내기
								//fmt.Println("andus >> 채굴 참여 대상자 확인")

								if !fc.tcpRunning {
									fc.TcpConnStartCh <- struct{}{}
								}

							}

						} else {
							// TODO: andus >> 검증실패..
							log.Println("andus >> OTPRN 검증 실패")

						}
					} else {
						// TODO: andus >> 서명 검증실패..
						log.Println("andus >> OTPRN 공개키 검증 실패")
					}
				}

			}
		}
	}
}

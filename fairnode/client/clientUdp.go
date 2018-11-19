package fairnodeclient

import (
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/fairnode/fairutil"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/p2p/discv5"
	"github.com/anduschain/go-anduschain/p2p/nat"
	"github.com/anduschain/go-anduschain/rlp"
	"log"
	"net"
	"time"
)

type EnodeCoinbase struct {
	Node     discv5.Node
	Coinbase common.Address
}

func (fc *FairnodeClient) UDPtoFairNode() {
	//TODO : andus >> udp 통신 to FairNode
	go fc.submitEnode()
	go fc.receiveOtprn()
}

func (fc *FairnodeClient) submitEnode() {
	// TODO : andus >> FairNode IP : localhost UDP Listener 11/06 -- start --
	Conn, err := net.DialUDP("udp", nil, fc.SAddrUDP)
	if err != nil {
		log.Println("andus >> UDPtoFairNode, DialUDP", err)
	}

	defer Conn.Close()

	// TODO : andus >> FairNode IP : localhost UDP Listener 11/06 -- end --
	t := time.NewTicker(60 * time.Second)

	fc.Enode = discv5.NewTable(discv5.PubkeyID(&fc.NodeKey.PublicKey), fc.LAddrUDP)

	ts := EnodeCoinbase{*fc.Enode, *fc.Coinbase}
	tsByte, err := rlp.EncodeToBytes(ts) // TODO : andus >> enode to byte
	if err != nil {
		log.Fatal("andus >> EncodeToBytes", err)
	}

	writeData := func() {
		_, err = Conn.Write(tsByte) // TODO : andus >> enode url 전송
		fmt.Println("andus >> enode 전송")
		if err != nil {
			log.Println("andus >> Write", err)
		}
	}

	writeData()

	for {
		select {
		case <-t.C:
			//TODO : andus >> FairNode에게 enode값 전송 ( 1분단위)
			// TODO : andus >> enode Sender -- start --
			// TODO : andus >> rlp encode -> byte ( enode type )
			writeData()
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

	natm, err := nat.Parse("any")
	if err != nil {
		log.Fatalf("-nat: %v", err)
	}

	realaddr := localServerConn.LocalAddr().(*net.UDPAddr)
	if true {
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
				log.Println("andus >> otprn 수신 에러", err)
				if err.(net.Error).Timeout() {
					continue
				}
				return
			}

			if n > 0 {
				// TODO : andus >> 수신된 otprn디코딩
				var tsOtprn otprn.TransferOtprn
				rlp.DecodeBytes(tsOtprnByte, &tsOtprn)

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

						fmt.Println("andus >> OTPRN 검증 완료")

						if ok := fairutil.IsJoinOK(fc.Otprn, fc.Coinbase); ok {
							//TODO : andus >> 참가 가능할 때 처리
							//TODO : andus >> 6. TCP 연결 채널에 메세지 보내기
							fc.TcpConnStartCh <- struct{}{}
							fmt.Println("andus >> 채굴 참여 대상자 확인")

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

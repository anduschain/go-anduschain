package udp

import (
	"fmt"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/fairnode/client/config"
	"github.com/anduschain/go-anduschain/fairnode/client/tcp"
	"github.com/anduschain/go-anduschain/fairnode/client/types"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes/msg"
	"github.com/anduschain/go-anduschain/fairnode/fairutil"
	"github.com/anduschain/go-anduschain/p2p/nat"
	"io"
	"log"
	"net"
	"time"
)

type Udp struct {
	SAddrUDP   *net.UDPAddr
	LAddrUDP   *net.UDPAddr
	services   map[string]types.Goroutine
	manger     types.Client
	tcpService *tcp.Tcp
}

func New(faiorServerString string, clientString string, manger types.Client, tcpService *tcp.Tcp) (*Udp, error) {

	SAddrUDP, err := net.ResolveUDPAddr("udp", faiorServerString)
	if err != nil {
		return nil, err
	}

	LAddrUDP, err := net.ResolveUDPAddr("udp", clientString)
	if err != nil {
		return nil, err
	}

	udp := &Udp{
		SAddrUDP: SAddrUDP,
		LAddrUDP: LAddrUDP,
		services: make(map[string]types.Goroutine),
		manger:   manger,
	}

	udp.services["submitEnode"] = types.Goroutine{udp.submitEnode, make(chan struct{})}
	udp.services["receiveOtprn"] = types.Goroutine{udp.receiveOtprn, make(chan struct{})}

	return udp, nil

}

func (u *Udp) Start() error {
	for name, serv := range u.services {
		log.Println(fmt.Sprintf("Info[andus] : %s Running", name))
		go serv.Fn(serv.Exit)
	}
	return nil
}

func (u *Udp) Stop() error {
	for _, srv := range u.services {
		srv.Exit <- struct{}{}
	}

	u.tcpService.Stop()

	return nil
}

func (u *Udp) submitEnode(exit chan struct{}) {
	// TODO : andus >> FairNode IP : localhost UDP Listener 11/06 -- start --
	Conn, err := net.DialUDP("udp", nil, u.SAddrUDP)
	if err != nil {
		log.Println("andus >> UDPtoFairNode, DialUDP", err)
	}

	// TODO : andus >> FairNode IP : localhost UDP Listener 11/06 -- end --
	t := time.NewTicker(60 * time.Second)
	ts := fairtypes.EnodeCoinbase{
		Enode:    u.manger.GetP2PServer().NodeInfo().Enode,
		Coinbase: u.manger.GetCoinbase(),
		Port:     config.DefaultConfig.ClientPort,
	}

	if err := msg.Send(msg.SendEnode, ts, Conn); err != nil {
		fmt.Println("andus >>>>>>", err)
	}

Exit:
	for {
		select {
		case <-t.C:
			//TODO : andus >> FairNode에게 enode값 전송 ( 1분단위)
			// TODO : andus >> enode Sender -- start --
			fmt.Println("andus >> Enode 전송")
			msg.Send(msg.SendEnode, ts, Conn)
		case <-exit:
			break Exit
		}
	}
}

func (u *Udp) receiveOtprn(exit chan struct{}) {

	//TODO : andus >> 1. OTPRN 수신

	localServerConn, err := net.ListenUDP("udp", u.LAddrUDP)
	if err != nil {
		log.Println("Udp Server", err)
	}

	defer localServerConn.Close()

	// TODO : andus >> NAT 추가 --- start ---

	natm, err := nat.Parse(config.DefaultConfig.NAT)
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

	notify := make(chan error)

	go func() {
		tsOtprnByte := make([]byte, 4096)
		for {
			n, err := localServerConn.Read(tsOtprnByte)
			if err != nil {
				notify <- err
				return
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
							u.manger.SetOtprn(&tsOtprn.Otp)
							//TODO : andus >> 3. 참여여부 확인

							if ok := fairutil.IsJoinOK(tsOtprn.Otp, u.manger.GetCoinbase()); ok {
								//TODO : andus >> 참가 가능할 때 처리
								//TODO : andus >> 6. TCP 연결 채널에 메세지 보내기

								log.Println("Debug[andus] : 참여대상이 맞음")
								u.tcpService.Start()

							} else {
								log.Println("Debug[andus] : 참여대상이 아님")
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
	}()

Exit:
	for {
		select {
		case err := <-notify:
			if io.EOF == err {
				fmt.Println("udp connection dropped message", err)
				return
			}
		case <-time.After(time.Second * 1):
			fmt.Println("UDP timeout, still alive")
		case <-exit:
			break Exit
		}
	}
}

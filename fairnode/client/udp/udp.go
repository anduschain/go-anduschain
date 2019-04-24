package udp

import (
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/fairnode/client/config"
	"github.com/anduschain/go-anduschain/fairnode/client/interface"
	"github.com/anduschain/go-anduschain/fairnode/client/tcp"
	"github.com/anduschain/go-anduschain/fairnode/client/types"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/fairnode/fairutil"
	"github.com/anduschain/go-anduschain/fairnode/transport"
	logger "github.com/anduschain/go-anduschain/log"
	"github.com/anduschain/go-anduschain/p2p/discover"
	"github.com/anduschain/go-anduschain/p2p/nat"
	"github.com/anduschain/go-anduschain/params"
	"net"
	"time"
)

const (
	SendEnodeInfo = 10 // minning을 하기 위해 fairnode에 enode를 보내는 주기(sec)
)

type Udp struct {
	SAddrUDP   *net.UDPAddr
	LAddrUDP   *net.UDPAddr
	services   map[string]types.Goroutine
	manger     _interface.Client
	tcpService *tcp.Tcp
	isRuning   bool
	RealAddr   *net.UDPAddr
	logger     logger.Logger
	nat        nat.Interface
}

func New(faiorServerString string, clientString string, manger _interface.Client, tcpService *tcp.Tcp) (*Udp, error) {

	SAddrUDP, err := net.ResolveUDPAddr("udp", faiorServerString)
	if err != nil {
		return nil, err
	}

	LAddrUDP, err := net.ResolveUDPAddr("udp", clientString)
	if err != nil {
		return nil, err
	}

	udp := &Udp{
		SAddrUDP:   SAddrUDP,
		LAddrUDP:   LAddrUDP,
		services:   make(map[string]types.Goroutine),
		manger:     manger,
		tcpService: tcpService,
		isRuning:   false,
		logger:     logger.New("fairclient", "UDP"),
	}

	udp.services["submitEnode"] = types.Goroutine{udp.submitEnode, make(chan struct{})}
	udp.services["receiveOtprn"] = types.Goroutine{udp.receiveOtprn, make(chan struct{})}

	return udp, nil

}

func (u *Udp) Start() error {
	if !u.isRuning {
		for name, serv := range u.services {
			u.logger.Info("Service Start", "Service : ", name)
			go serv.Fn(serv.Exit, nil)
		}

		u.isRuning = true
	}

	return nil
}

func (u *Udp) Stop() error {
	if u.isRuning {
		for _, srv := range u.services {
			srv.Exit <- struct{}{}
		}

		for i := range u.manger.GetSavedOtprnHashs() {
			u.tcpService.Stop(u.manger.GetSavedOtprnHashs()[i])
		}

		u.isRuning = false
	}

	return nil
}

func (u *Udp) NatStart(conn net.Conn) *net.UDPAddr {
	// TODO : andus >> NAT 추가 --- start ---
	u.nat = u.manger.GetNat()
	realaddr := conn.LocalAddr().(*net.UDPAddr)
	if u.nat != nil {
		if !realaddr.IP.IsLoopback() {
			go nat.Map(u.nat, nil, "udp", realaddr.Port, realaddr.Port, "andus fairnode discovery")
		}
		// TODO: react to external IP changes over time.
		if ext, err := u.nat.ExternalIP(); err == nil {
			realaddr = &net.UDPAddr{IP: ext, Port: realaddr.Port}
		}
	}
	// TODO : andus >> NAT 추가 --- end ---

	return realaddr
}

func (u *Udp) submitEnode(exit chan struct{}, v interface{}) {
	// TODO : andus >> FairNode IP : localhost UDP Listener 11/06 -- start --
	Conn, err := net.DialUDP("udp", nil, u.SAddrUDP)
	if err != nil {
		u.logger.Error("Dial", "error", err)
	}

	defer Conn.Close()

	// TODO : andus >> FairNode IP : localhost UDP Listener 11/06 -- end --
	t := time.NewTicker(SendEnodeInfo * time.Second)

Exit:
	for {
		select {
		case <-t.C:
			//TODO : andus >> FairNode에게 enode값 전송 ( 10초단위)
			// TODO : andus >> enode Sender -- start --
			ts := fairtypes.EnodeCoinbase{
				Enode:    u.manger.GetP2PServer().NodeInfo().ID,
				Coinbase: u.manger.GetCoinbase(),
				Port:     config.DefaultConfig.ClientPort,
				Version:  params.Version,
				ChainID:  u.manger.GetBlockChain().Config().ChainID.Uint64(),
			}

			node, err := discover.ParseNode(u.manger.GetP2PServer().NodeInfo().Enode)
			if err != nil {
				return
			}

			if node.IP.To4() == nil {
				addr, err := net.ResolveUDPAddr("udp", Conn.LocalAddr().String())
				if err != nil {
					return
				}
				node.IP = addr.IP
				ts.IP = addr.IP.String()
			} else {
				ts.IP = u.manger.GetP2PServer().NodeInfo().IP
			}

			u.logger.Info("Enode 전송", "enodeID", u.manger.GetP2PServer().NodeInfo().ID, "IP", ts.IP)
			err = transport.SendUDP(transport.SendEnode, ts, Conn)
			if err != nil {
				u.logger.Error("transport.SendUDP", "error", err)
			}
		case <-exit:
			break Exit
		}
	}

	defer u.logger.Debug("submitEnode killed")
}

func (u *Udp) receiveOtprn(exit chan struct{}, v interface{}) {

	//TODO : andus >> 1. OTPRN 수신

	localServerConn, err := net.ListenUDP("udp", u.LAddrUDP)
	if err != nil {
		u.logger.Error("ListenUDP", "error", err)
	}

	u.NatStart(localServerConn)

	notify := make(chan error)

	go func() {
		tsOtprnByte := make([]byte, 4096)
		for {
			n, err := localServerConn.Read(tsOtprnByte)
			if err != nil {
				notify <- err
				if _, ok := err.(*net.OpError); ok {
					return
				}
			}

			if n > 0 {
				// TODO : andus >> 수신된 otprn디코딩
				fromFairnodeMsg := transport.ReadUDP(tsOtprnByte)
				if fromFairnodeMsg == nil {
					continue
				}
				switch fromFairnodeMsg.Code {
				case transport.SendOTPRN:
					var tsOtprn fairtypes.TransferOtprn
					fromFairnodeMsg.Decode(&tsOtprn)
					u.logger.Info("OTPRN 수신됨", "hash", tsOtprn.Hash, "miner", tsOtprn.Otp.Mminer, "epoch", tsOtprn.Otp.Epoch, "fee", tsOtprn.Otp.Fee)

					//TODO : andus >> 2. OTRRN 검증
					fairPubKey, err := crypto.SigToPub(tsOtprn.Hash.Bytes(), tsOtprn.Sig)
					if err != nil {
						u.logger.Error("OTPRN 공개키 로드 에러", "error", err)
					}

					if crypto.VerifySignature(crypto.FromECDSAPub(fairPubKey), tsOtprn.Hash.Bytes(), tsOtprn.Sig[:64]) {
						otprnHash := tsOtprn.Otp.HashOtprn()
						if otprnHash == tsOtprn.Hash {

							// TODO: andus >> 검증완료, Otprn 저장
							u.manger.StoreOtprnWidthSig(&tsOtprn.Otp, tsOtprn.Sig)

							if !u.manger.GetBlockMine() {
								//큐에 있는 이전 otprn을 없애기
								u.logger.Info("SendOTPRN")
								u.manger.GetStoreOtprnWidthSig()
							}
							//TODO : andus >> 3. 참여여부 확인
							if ok := fairutil.IsJoinOK(&tsOtprn.Otp, u.manger.GetCoinbase()); ok {
								//TODO : andus >> 참가 가능할 때 처리
								//TODO : andus >> 6. TCP 연결 채널에 메세지 보내기
								u.logger.Debug("참여대상이 맞음")
								u.tcpService.Start(otprnHash)
							} else {
								u.logger.Debug("참여대상이 아님")
							}
						} else {
							// TODO: andus >> 검증실패..
							u.logger.Debug("OTPRN 검증 실패")

						}
					} else {
						// TODO: andus >> 서명 검증실패..
					}
				}
			}
		}
	}()

Exit:
	for {
		select {
		case err := <-notify:
			if _, ok := err.(*net.OpError); ok {
				u.logger.Debug("udp connection dropped message", "error", err)
				break Exit
			}
		case <-time.After(time.Second * 1):
			//log.Println("Debug[andus] : UDP timeout, still alive")
		case <-exit:
			localServerConn.Close()
		}
	}

	defer u.logger.Debug("ReceiveOtprn Killed")
}

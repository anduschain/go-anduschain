package fairudp

import (
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/fairnode/server/backend"
	"github.com/anduschain/go-anduschain/fairnode/server/db"
	"github.com/anduschain/go-anduschain/fairnode/server/manager/pool"
	"github.com/anduschain/go-anduschain/fairnode/transport"
	"github.com/anduschain/go-anduschain/p2p/nat"
	log "gopkg.in/inconshreveable/log15.v2"
	"io"
	"math/big"
	"net"
	"strings"
	"time"
)

const (
	MinActiveNum = 3
)

var (
	errNat     = errors.New("NAT 설정에 문제가 있습니다")
	errUdpConn = errors.New("UDP 커넥션 설정에 에러가 있음")
)

type tcpInterface interface {
	StartLeague(leagueChange bool)
}

type SendOtprn struct {
	LeagueChange bool
	TsOtprn      fairtypes.TransferOtprn
}

type FairUdp struct {
	LAddrUDP    *net.UDPAddr
	natm        nat.Interface
	udpConn     *net.UDPConn
	services    map[string]backend.Goroutine
	db          *db.FairNodeDB
	manager     backend.Manager
	ftcp        tcpInterface
	sendOtprnCH chan SendOtprn

	checkconn chan SendOtprn
	logger    log.Logger
}

func New(db *db.FairNodeDB, fm backend.Manager, tcp tcpInterface) (*FairUdp, error) {

	addr := fmt.Sprintf(":%s", backend.DefaultConfig.Port)

	laddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	natm, err := nat.Parse(backend.DefaultConfig.NAT)
	if err != nil {
		return nil, errNat
	}

	fu := &FairUdp{
		LAddrUDP:    laddr,
		natm:        natm,
		services:    make(map[string]backend.Goroutine),
		db:          db,
		manager:     fm,
		ftcp:        tcp,
		sendOtprnCH: make(chan SendOtprn),
		checkconn:   make(chan SendOtprn),
		logger:      log.New("fairnode", "udp"),
	}

	fu.services["manageActiveNode"] = backend.Goroutine{fu.manageActiveNode, make(chan struct{}, 1)}
	fu.services["manageOtprn"] = backend.Goroutine{fu.manageOtprn, make(chan struct{}, 1)}
	fu.services["JobActiveNode"] = backend.Goroutine{fu.JobActiveNode, make(chan struct{}, 1)}
	fu.services["broadcast"] = backend.Goroutine{fu.broadcast, make(chan struct{}, 1)}

	return fu, nil
}

func (fu *FairUdp) Start() error {
	var err error
	fu.udpConn, err = net.ListenUDP("udp", fu.LAddrUDP)
	if err != nil {
		return errUdpConn
	}

	if fu.natm != nil {
		realaddr := fu.udpConn.LocalAddr().(*net.UDPAddr)
		if fu.natm != nil {
			if !realaddr.IP.IsLoopback() {
				go nat.Map(fu.natm, nil, "udp", realaddr.Port, realaddr.Port, "andus fairnode discovery")
			}
			// TODO: react to external IP changes over time.
			if ext, err := fu.natm.ExternalIP(); err == nil {
				realaddr = &net.UDPAddr{IP: ext, Port: realaddr.Port}
			}
		}
	}

	for name, srv := range fu.services {
		fu.logger.Info(fmt.Sprintf("UDP 서비스 %s 실행됨", name))
		go srv.Fn(srv.Exit)
	}

	return nil
}

func (fu *FairUdp) Stop() error {
	err := fu.udpConn.Close()
	if err != nil {
		return err
	}

	time.Sleep(1 * time.Second)

	for _, srv := range fu.services {
		srv.Exit <- struct{}{}
	}

	return nil
}

// Geth node Heart beat update ( Active node 관리 )
// enode값 수신
func (fu *FairUdp) manageActiveNode(exit chan struct{}) {
	defer fu.logger.Debug("manageActiveNode kill")
	notify := make(chan error)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, fromAddr, err := fu.udpConn.ReadFrom(buf)
			if err != nil {
				notify <- err
				return
			}
			if n > 0 {
				m := transport.ReadUDP(buf[:n])
				if m == nil {
					return
				}
				switch m.Code {
				case transport.SendEnode:
					var fromGeth fairtypes.EnodeCoinbase
					m.Decode(&fromGeth)
					addr, err := net.ResolveIPAddr("", strings.Split(fromAddr.String(), ":")[0])
					if err != nil {
						return
					}
					if !addr.IP.Equal(net.IPv4zero) {
						fu.db.SaveActiveNode(fromGeth.Enode, fromGeth.Coinbase, fromGeth.Port, fromAddr.String())
					}

				default:
					fu.logger.Error("unKnown udp 메시지 코드", "manageActiveNode", m.Code)
				}
			}
		}
	}()

Exit:
	for {
		select {
		case err := <-notify:
			if io.EOF == err {
				fu.logger.Error("udp connection dropped message", "manageActiveNode", err)
				return
			}
		case <-exit:
			break Exit
		}
	}

}

//UDP로 받은 ActiveNode의 마지막 수신 시간을 확인하여 3분이 지나갔을시 DB에서 삭제
func (fu *FairUdp) JobActiveNode(exit chan struct{}) {
	defer fu.logger.Debug("JobActiveNode kill")
	t := time.NewTicker(3 * time.Minute)
Exit:
	for {
		select {
		case <-t.C:
			// 3분이상 들어오지 않은 enode 정리 (mongodb)
			fu.db.JobCheckActiveNode()
		case <-exit:
			break Exit
		}
	}
}

// OTPRN을 발행
// OTPRN 발행조건, 활성 노드수가 MinActiveNum 이상일때, 블록 번호가 에폭의 반 이상일때 % == 0
func (fu *FairUdp) manageOtprn(exit chan struct{}) {
	defer fu.logger.Debug("manageOtprn kill")
	t := time.NewTicker(3 * time.Second)

	sendOtprn := func(leaguechange bool) {
		actNum := fu.db.GetActiveNodeNum()
		if actNum >= MinActiveNum {
			// OTPRN 생성
			tsOtp, err := fu.makeOTPRN(uint64(actNum))
			if err != nil {
				fu.logger.Error("make otprn", "manageOtprn", err)
			}
			// 리그 전송 tcp
			fu.manager.StoreOtprn(&tsOtp.Otp) // otprn push
			fu.sendOtprnCH <- SendOtprn{leaguechange, tsOtp}

		} else {
			fu.manager.DeleteStoreOtprn()
		}
	}

Exit:
	for {
		select {
		case <-t.C:
			if fu.manager.GetUsingOtprn() == nil {
				sendOtprn(true)
			}
		case <-fu.manager.GetManagerOtprnCh():
			// 파이널 블록을 보낸 후 otprn을 전송 할 것인지 결졍하는 부분
			// 현재 블록 번호를 에폭으로 나누어서 0인 경우
			// half of epoch
			epoch := fu.manager.GetEpoch()
			if epoch.Int64() == 0 {
				continue
			}

			if fu.manager.GetLastBlockNum().Mod(fu.manager.GetLastBlockNum(), epoch).Int64() == 0 {
				// 리그 교체
				fu.manager.GetStopLeagueCh() <- struct{}{}
			} else {
				// joinTx 생성
				fu.manager.GetMakeJoinTxCh() <- struct{}{}
			}

			if fu.manager.GetLastBlockNum().Mod(fu.manager.GetLastBlockNum(), big.NewInt(int64(epoch.Int64()/2))).Int64() == 0 &&
				fu.manager.GetLastBlockNum().Mod(fu.manager.GetLastBlockNum(), epoch).Int64() != 0 {
				sendOtprn(false)
			}

		case otprnHash := <-fu.manager.GetReSendOtprn():

			leaguePool := fu.manager.GetLeaguePool()
			leaguePool.DeleteCh <- pool.OtprnHash(otprnHash)

			sendOtprn(true)
		case <-exit:
			break Exit
		}
	}
}

//모든 활성노드들에게 OTPRN과 서명을 보냄
func (fu *FairUdp) broadcast(exit chan struct{}) {
Exit:
	for {
		select {
		case OtprnStr := <-fu.sendOtprnCH:
			fu.sendUdpAll(transport.SendOTPRN, OtprnStr.TsOtprn) // UDP OTPRN submit
			// 리그 시작
			//fu.manager.SetLeagueRunning(true)

			fu.ftcp.StartLeague(OtprnStr.LeagueChange)

		case <-exit:
			break Exit
		}
	}
}

// 활성 node 전체에게 메시지 보냄
func (fu *FairUdp) sendUdpAll(msgcode uint32, data interface{}) {
	activeNodeList := fu.db.GetActiveNodeList()
	for index := range activeNodeList {
		url := activeNodeList[index].Ip + ":" + activeNodeList[index].Port
		ServerAddr, err := net.ResolveUDPAddr("udp", url)
		if err != nil {
			fu.logger.Error("ResolveUDPAddr", "error", err)
			continue
		}
		Conn, err := net.DialUDP("udp", nil, ServerAddr)
		if err != nil {
			fu.logger.Error("DialUDP", "error", err)
			continue
		}
		err = transport.SendUDP(msgcode, data, Conn)
		if err != nil {
			fu.logger.Error("transport.SendUDP", "error", err)
		}
		Conn.Close()
	}
}

// OTPRN을 생성후 서명, DB에 저장
func (fu *FairUdp) makeOTPRN(activeNodeNum uint64) (fairtypes.TransferOtprn, error) {
	otp := otprn.New(activeNodeNum)
	acc := fu.manager.GetServerKey()
	sig, err := otp.SignOtprn(acc.ServerAcc, otp.HashOtprn(), acc.KeyStore)
	if err != nil {
		return fairtypes.TransferOtprn{}, err
	}

	tsOtp := fairtypes.TransferOtprn{
		Otp:  *otp,
		Sig:  sig,
		Hash: otp.HashOtprn(),
	}

	// OTPRN DB 저장
	fu.db.SaveOtprn(tsOtp)

	return tsOtp, nil
}

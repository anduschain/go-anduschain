package fairtcp

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes/msg"
	"github.com/anduschain/go-anduschain/fairnode/fairutil"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/fairnode/server/backend"
	"github.com/anduschain/go-anduschain/fairnode/server/db"
	"github.com/anduschain/go-anduschain/fairnode/server/manager/pool"
	"github.com/anduschain/go-anduschain/p2p/nat"
	"github.com/anduschain/go-anduschain/rlp"
	"io"
	"log"
	"net"
	"time"
)

var (
	errNat          = errors.New("NAT 설정에 문제가 있습니다")
	errTcpListen    = errors.New("TCP Lister 설정에 문제가 있습니다")
	closeConnection = errors.New("close")
)

type FairTcp struct {
	LAddrTCP *net.TCPAddr
	natm     nat.Interface
	listener *net.TCPListener
	Db       *db.FairNodeDB
	manager  backend.Manager
	services map[string]backend.Goroutine
}

func New(db *db.FairNodeDB, fm backend.Manager) (*FairTcp, error) {
	addr := fmt.Sprintf(":%s", backend.DefaultConfig.Port)

	LAddrTCP, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}

	natm, err := nat.Parse(backend.DefaultConfig.NAT)
	if err != nil {
		return nil, errNat
	}

	ft := &FairTcp{
		LAddrTCP: LAddrTCP,
		natm:     natm,
		Db:       db,
		manager:  fm,
		services: make(map[string]backend.Goroutine),
	}

	ft.services["accepter"] = backend.Goroutine{ft.accepter, make(chan struct{}, 1)}

	return ft, nil
}

func (ft *FairTcp) Start() error {

	var err error

	ft.listener, err = net.ListenTCP("tcp", ft.LAddrTCP)
	if err != nil {
		return errTcpListen
	}

	if ft.natm != nil {
		laddr := ft.listener.Addr().(*net.TCPAddr)
		// Map the TCP listening port if NAT is configured.
		if !laddr.IP.IsLoopback() {
			go func() {
				nat.Map(ft.natm, nil, "tcp", laddr.Port, laddr.Port, "andus fairnode discovery")
			}()
		}
	}

	for name, srv := range ft.services {
		log.Printf("Info[andus] : TCP 서비스 %s 실행됨", name)
		go srv.Fn(srv.Exit)
	}

	return nil

}

func (ft *FairTcp) Stop() error {
	err := ft.listener.Close()
	if err != nil {
		return err
	}

	time.Sleep(1 * time.Second)

	for _, srv := range ft.services {
		srv.Exit <- struct{}{}
	}

	return nil
}

func (ft *FairTcp) accepter(exit chan struct{}) {
	defer log.Println("Info[andus] : tcp accepter kill")

	noify := make(chan error)
	accept := make(chan net.Conn)

	go func() {
		for {
			conn, err := ft.listener.Accept()
			if err != nil {
				noify <- err
				return
			}

			accept <- conn
		}
	}()

Exit:
	for {
		select {
		case <-exit:
			break Exit
		case <-time.After(time.Second * 1):
			//fmt.Println("TCP timeout, still alive")
		case err := <-noify:
			log.Println("Error[andus] : ", err)
		case conn := <-accept:
			log.Println("Info[andus] : tcp 접속 함")
			go ft.handeler(conn)
		}

	}
}

func (ft *FairTcp) handeler(conn net.Conn) {
	defer log.Println("Info[andus] : hander kill")

	noify := make(chan error)

	poolUpdate := func(leaguePool *pool.LeaguePool, otprn string, tsf fairtypes.TransferCheck) {
		leaguePool.UpdateCh <- pool.PoolIn{
			Hash: pool.StringToOtprn(otprn),
			Node: pool.Node{Enode: tsf.Enode, Coinbase: tsf.Coinbase, Conn: nil},
		}
	}

	go func() {
		defer log.Println("Info[andus] : ReadMsg Loop 죽음")
		buf := make([]byte, 4096)
		leaguePool := ft.manager.GetLeaguePool()
		votePool := ft.manager.GetVotePool()
		for {
			n, err := conn.Read(buf)
			if err != nil {
				noify <- err
				if err == io.EOF {
					return
				}
				if _, ok := err.(*net.OpError); ok {
					return
				}
			}

			if n > 0 {
				fromGethMsg := msg.ReadMsg(buf)
				switch fromGethMsg.Code {
				case msg.ReqLeagueJoinOK:
					var tsf fairtypes.TransferCheck
					fromGethMsg.Decode(&tsf)

					otprnHash := tsf.Otprn.HashOtprn().String()
					if ft.Db.CheckEnodeAndCoinbse(tsf.Enode, tsf.Coinbase.String()) {
						// TODO : andus >> 1. Enode가 맞는지 확인 ( 조회 되지 않으면 팅김 )
						// TODO : andus >> 2. 해당하는 Enode가 이전에 보낸 코인베이스와 일치하는지
						if fairutil.IsJoinOK(tsf.Otprn, tsf.Coinbase) {
							// TODO : 채굴 리그 생성
							// TODO : 1. 채굴자 저장 ( key otprn num, Enode의 ID를 저장....)

							_, n, _ := leaguePool.GetLeagueList(pool.StringToOtprn(otprnHash))
							if otprn.Mminer > n {
								log.Println("INFO : 참여 가능자 저장됨", tsf.Coinbase.String())
								leaguePool.InsertCh <- pool.PoolIn{
									Hash: pool.StringToOtprn(otprnHash),
									Node: pool.Node{Enode: tsf.Enode, Coinbase: tsf.Coinbase, Conn: conn},
								}
							} else {
								// TODO : 참여 인원수 오버된 케이스
								log.Println("INFO : 참여 인원수 오버된 케이스", tsf.Enode)
								noify <- closeConnection
								poolUpdate(leaguePool, otprnHash, tsf)

							}
						} else {
							// TODO : andus >> 참여 대상자가 아니다
							log.Println("INFO : 참여 대상자가 아니다", tsf.Enode)
							noify <- closeConnection
							poolUpdate(leaguePool, otprnHash, tsf)

						}
					} else {
						// TODO : andus >> 리그 참여 정보가 다르다
						log.Println("INFO : 리그 참여 정보가 다르다", tsf.Enode)
						noify <- closeConnection
						poolUpdate(leaguePool, otprnHash, tsf)

					}
				case msg.SendBlockForVote:
					var voteBlock fairtypes.TransferVoteBlock

					if err := fromGethMsg.Decode(&voteBlock); err != nil {
						fmt.Println("------SendBlockForVote------", err)
					}

					stream := rlp.NewStream(bytes.NewReader(voteBlock.EncodedBlock), 0)

					block := &types.Block{}

					if err := block.DecodeRLP(stream); err != nil {
						fmt.Println("-------디코딩 테스트 에러 ----------", err)
					}

					otp := ft.manager.GetOtprn()
					lastNum := ft.manager.GetLastBlockNum()

					if otp.HashOtprn() == voteBlock.OtprnHash && lastNum+1 == block.NumberU64() {
						votePool.InsertCh <- pool.Vote{
							Hash:     pool.StringToOtprn(voteBlock.OtprnHash.String()),
							Block:    block,
							Coinbase: voteBlock.Voter,
							Receipts: voteBlock.Receipts,
						}

						fmt.Println("-----블록 투표 됨-----", voteBlock.Voter.String(), block.NumberU64())
					} else {
						fmt.Println("-----다른 OTPRN으로 투표 또는 숫자가 맞지 않아 거절됨-----", lastNum, otp.HashOtprn() == voteBlock.OtprnHash, block.Coinbase().String(), block.NumberU64())
						noify <- closeConnection
					}

				}
			}
		}

	}()

Exit:
	for {
		select {
		case <-time.After(time.Second * 1):
			//fmt.Println("tcp timeout 1, still alive")
		case err := <-noify:
			if io.EOF == err {
				fmt.Println("tcp connection dropped message", err)
				conn.Close()
				break Exit
			} else if "close" == err.Error() {
				conn.Close()
			} else if _, ok := err.(*net.OpError); ok {
				fmt.Println("tcp connection dropped message", err)
				break Exit
			}
			log.Println("Error[andus] : ", err)
		}

	}
}

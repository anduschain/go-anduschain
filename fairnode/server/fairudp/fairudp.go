package fairudp

import (
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/accounts"
	"github.com/anduschain/go-anduschain/accounts/keystore"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes/msg"
	"github.com/anduschain/go-anduschain/fairnode/fairutil"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/fairnode/server/backend"
	"github.com/anduschain/go-anduschain/fairnode/server/db"
	"github.com/anduschain/go-anduschain/fairnode/server/manager/pool"
	"github.com/anduschain/go-anduschain/p2p/nat"
	"io"
	"log"
	"net"
	"time"
)

var (
	errNat     = errors.New("NAT 설정에 문제가 있습니다")
	errUdpConn = errors.New("UDP 커넥션 설정에 에러가 있음")
)

type FairUdp struct {
	LAddrUDP *net.UDPAddr
	natm     nat.Interface
	udpConn  *net.UDPConn
	services map[string]backend.Goroutine
	db       *db.FairNodeDB
	fm       backend.Manager
}

func New(db *db.FairNodeDB, fm backend.Manager) (*FairUdp, error) {

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
		LAddrUDP: laddr,
		natm:     natm,
		services: make(map[string]backend.Goroutine),
		db:       db,
		fm:       fm,
	}

	fu.services["manageActiveNode"] = backend.Goroutine{fu.manageActiveNode, make(chan struct{}, 1)}
	fu.services["manageOtprn"] = backend.Goroutine{fu.manageOtprn, make(chan struct{}, 1)}
	fu.services["JobActiveNode"] = backend.Goroutine{fu.JobActiveNode, make(chan struct{}, 1)}

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
		log.Printf("Info[andus] : UDP 서비스 %s 실행됨", name)
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

func (fu *FairUdp) manageActiveNode(exit chan struct{}) {
	// TODO : andus >> Geth node Heart beat update ( Active node 관리 )
	// TODO : enode값 수신

	defer log.Printf("Info[andus] : manageActiveNode kill")
	notify := make(chan error)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := fu.udpConn.Read(buf)
			if err != nil {
				notify <- err
				return
			}
			if n > 0 {
				fromGethMsg := msg.ReadMsg(buf)
				switch fromGethMsg.Code {
				case msg.SendEnode:
					var fromGeth fairtypes.EnodeCoinbase
					fromGethMsg.Decode(&fromGeth)
					fu.db.SaveActiveNode(fromGeth.Enode, fromGeth.Coinbase, fromGeth.Port)
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
			//fmt.Println("UDP timeout, still alive")
		case <-exit:
			break Exit
		}
	}

}

func (fu *FairUdp) JobActiveNode(exit chan struct{}) {
	defer log.Printf("Info[andus] : JobActiveNode kill")

	t := time.NewTicker(3 * time.Minute)
Exit:
	for {
		select {
		case <-t.C:
			// TODO : andus >> 3분이상 들어오지 않은 enode 지우기 (mongodb)
			fu.db.JobCheckActiveNode()
		case <-exit:
			break Exit
		}
	}
}

// OTPRN을 발행
func (fu *FairUdp) manageOtprn(exit chan struct{}) {
	defer log.Printf("Info[andus] : manageOtprn kill")

	t := time.NewTicker(3 * time.Second)

Exit:
	for {
		select {
		case <-t.C:
			actNum := fu.db.GetActiveNodeNum()
			if !fu.fm.GetLeagueRunning() && actNum >= 3 {

				activeNodeNum := uint64(fu.db.GetActiveNodeNum())

				otp := otprn.New(activeNodeNum)

				// TODO : andus >> otprn을 서명
				acc := fu.fm.GetServerKey()
				sig, err := otp.SignOtprn(acc.ServerAcc, otp.HashOtprn(), acc.KeyStore)
				if err != nil {
					log.Println("Otprn 서명 에러", err)
				}

				tsOtp := fairtypes.TransferOtprn{
					Otp:  *otp,
					Sig:  sig,
					Hash: otp.HashOtprn(),
				}
				// andus >> OTPRN DB 저장
				fu.db.SaveOtprn(tsOtp)

				if activeNodeNum > 0 {
					fu.fm.SetLeagueRunning(true)
					activeNodeList := fu.db.GetActiveNodeList()

					go func() {
						t := time.NewTicker(10 * time.Second)
						for {
							select {
							case <-t.C:
								fu.fm.SetOtprn(otp)
								go fu.sendLeague(tsOtp.Hash.String())
								return
							}
						}

					}()

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
		case <-exit:
			break Exit
		}
	}
}

func (fu *FairUdp) sendLeague(otprnHash string) {
	t := time.NewTicker(5 * time.Second)
	leaguePool := fu.fm.GetLeaguePool()
	var percent float64 = 30
	for {
		select {
		case <-t.C:
			nodes, num, enodes := leaguePool.GetLeagueList(pool.StringToOtprn(otprnHash))
			// 가능한 사람의 30%이상일때 접속할 채굴 리그를 전송해줌
			if num >= fu.JoinTotalNum(percent) && num > 0 {
				for index := range nodes {
					if nodes[index].Conn != nil {
						msg.Send(msg.SendLeageNodeList, enodes, nodes[index].Conn)
					}
				}

				fmt.Println("-------리그 전송---------")

				time.Sleep(5 * time.Second)

				for index := range nodes {
					if nodes[index].Conn != nil {
						msg.Send(msg.MakeJoinTx, otprnHash, nodes[index].Conn)
					}
				}

				fmt.Println("-------조인 tx 생성--------")

				time.Sleep(5 * time.Second)

				for index := range nodes {
					if nodes[index].Conn != nil {
						msg.Send(msg.MakeBlock, otprnHash, nodes[index].Conn)
					}
				}

				fmt.Println("-------블록 생성--------", otprnHash)

				// peer list 전송후 30초
				go fu.sendFinalBlock(otprnHash)
				return
			}

			//if num >= fu.JoinTotalNum(percent) {
			//	for index := range nodes {
			//		if nodes[index].Conn != nil {
			//			msg.Send(msg.SendLeageNodeList, enodes, nodes[index].Conn)
			//		}
			//	}
			//
			//	// peer list 전송후 30초
			//	go fu.sendWiningBlock(otprnHash)
			//
			//	return
			//} else {
			//log.Println("Info[andus] : 리그가 성립 안됨 연결 새로운 리그 시작 : ", num)
			//for i := range nodes {
			//	if nodes[i].Conn == nil {
			//		msg.Send(msg.MinerLeageStop, "리그가 종료 되었습니다", nodes[i].Conn)
			//		nodes[i].Conn.Close()
			//	}
			//}
			//leaguePool.SnapShot <- pool.StringToOtprn(otprnHash)
			//leaguePool.DeleteCh <- pool.StringToOtprn(otprnHash)
			//fu.fm.SetLeagueRunning(false)
			//return
			//}
		}
	}
}

func (fu *FairUdp) sendFinalBlock(otprnHash string) {
	t := time.NewTicker(20 * time.Second)
	leaguePool := fu.fm.GetLeaguePool()
	votePool := fu.fm.GetVotePool()
	nodes, _, _ := leaguePool.GetLeagueList(pool.StringToOtprn(otprnHash))

	notify := make(chan *fairtypes.FinalBlock)

	go func() {
		t := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-t.C:
				fb := fu.GetFinalBlock(otprnHash, votePool)
				if fb == nil {
					continue
				} else {
					notify <- fb
					return
				}
			}
		}
	}()

	for {
		select {
		case <-t.C:
			n := <-notify

			for index := range nodes {
				if nodes[index].Conn != nil {
					msg.Send(msg.SendFinalBlock, n.GetTsFinalBlock(), nodes[index].Conn)
				}
			}

			fmt.Println("----파이널 블록 전송-----", n.Block.NumberU64(), n.Block.Coinbase().String())

			fu.fm.SetLeagueRunning(false)

			leaguePool.SnapShot <- pool.StringToOtprn(otprnHash)
			leaguePool.DeleteCh <- pool.StringToOtprn(otprnHash)

			// DB에 블록 저장
			votePool.SnapShot <- n.Block
			votePool.DeleteCh <- pool.StringToOtprn(otprnHash)
			return
		}
	}
}

func (fu *FairUdp) JoinTotalNum(persent float64) uint64 {
	aciveNode := fu.db.GetActiveNodeList()
	var count float64 = 0
	for i := range aciveNode {
		if fairutil.IsJoinOK(fu.fm.GetOtprn(), common.HexToAddress(aciveNode[i].Coinbase)) {
			count += 1
		}
	}

	return uint64(count * (persent / 100))
}

func (fu *FairUdp) GetFinalBlock(otprnHash string, votePool *pool.VotePool) *fairtypes.FinalBlock {
	voteBlocks := votePool.GetVoteBlocks(pool.StringToOtprn(otprnHash))
	acc := fu.fm.GetServerKey()

	var fb fairtypes.FinalBlock

	if len(voteBlocks) == 0 {
		return nil
	} else if len(voteBlocks) == 1 {
		fmt.Println("--------------count == 1----------")
		fb.Block = voteBlocks[0].Block
		fb.Receipts = voteBlocks[0].Receipts
		SignFairNode(fb.Block, voteBlocks[0], acc.ServerAcc, acc.KeyStore)
	} else {
		var cnt uint64 = 0
		var pvBlock pool.VoteBlock
		for i := range voteBlocks {
			// 1. count가 높은 블록
			// 2. Rand == diffcult 값이 높은 블록
			// 3. joinNunce	== nonce 값이 놓은 블록
			// 4. 블록이 홀수 이면 - 주소값이 작은사람 , 블록이 짝수이면 - 주소값이 큰사람
			if cnt < voteBlocks[i].Count {
				fb.Block = voteBlocks[i].Block
				fb.Receipts = voteBlocks[i].Receipts
				pvBlock = voteBlocks[i]
				cnt = voteBlocks[i].Count
			} else if cnt == voteBlocks[i].Count {
				// 동수인 투표일때
				if voteBlocks[i].Block.Difficulty().Cmp(pvBlock.Block.Difficulty()) == 1 {
					// diffcult 값이 높은 블록
					fb.Block = voteBlocks[i].Block
					fb.Receipts = voteBlocks[i].Receipts
					pvBlock = voteBlocks[i]
				} else if voteBlocks[i].Block.Difficulty().Cmp(pvBlock.Block.Difficulty()) == 0 {
					// diffcult 값이 같을때
					if voteBlocks[i].Block.Nonce() > pvBlock.Block.Nonce() {
						// nonce 값이 큰 블록
						fb.Block = voteBlocks[i].Block
						fb.Receipts = voteBlocks[i].Receipts
						pvBlock = voteBlocks[i]
					} else if voteBlocks[i].Block.Nonce() == pvBlock.Block.Nonce() {
						// nonce 값이 같을 때
						if voteBlocks[i].Block.Number().Uint64()%2 == 0 {
							// 블록 번호가 짝수 일때
							if voteBlocks[i].Block.Coinbase().Big().Cmp(pvBlock.Block.Coinbase().Big()) == 1 {
								// 주소값이 큰 블록
								fb.Block = voteBlocks[i].Block
								fb.Receipts = voteBlocks[i].Receipts
								pvBlock = voteBlocks[i]
							}
						} else {
							// 블록 번호가 홀수 일때
							if voteBlocks[i].Block.Coinbase().Big().Cmp(pvBlock.Block.Coinbase().Big()) == -1 {
								// 주소값이 작은 블록
								fb.Block = voteBlocks[i].Block
								fb.Receipts = voteBlocks[i].Receipts
								pvBlock = voteBlocks[i]
							}
						}

					}
				}
			}
		}

		SignFairNode(fb.Block, pvBlock, acc.ServerAcc, acc.KeyStore)
	}

	return &fb
}

func SignFairNode(block *types.Block, vBlock pool.VoteBlock, account accounts.Account, ks *keystore.KeyStore) {
	sig, err := ks.SignHash(account, vBlock.Block.Hash().Bytes())
	if err != nil {
		log.Println("Error[andus] : SignFairNode 서명에러", err)
	}

	block.Voter = vBlock.Voters
	block.FairNodeSig = sig
}

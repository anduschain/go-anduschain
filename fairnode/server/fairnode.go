package server

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/accounts"
	"github.com/anduschain/go-anduschain/accounts/keystore"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/fairnode/fairutil"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/fairnode/server/db"
	"github.com/anduschain/go-anduschain/p2p/discv5"
	"github.com/anduschain/go-anduschain/p2p/nat"
	"github.com/anduschain/go-anduschain/rlp"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TODO : andus >> timezone 셋팅

var (
	makeOtprnError = errors.New("OTPRN 구조체 생성 오류")
)

type FairNode struct {
	Privkey  *ecdsa.PrivateKey
	LaddrTcp *net.TCPAddr
	LaddrUdp *net.UDPAddr
	keypath  string
	Account  accounts.Account
	UdpConn  *net.UDPConn
	TcpConn  *net.TCPListener
	Db       *db.FairNodeDB

	otprn           *otprn.Otprn
	SingedOtprn     *string // 전자서명값
	startSignalCh   chan struct{}
	startMakeLeague chan string
	Wg              sync.WaitGroup
	lock            sync.RWMutex
	StopCh          chan struct{} // TODO : andus >> 죽을때 처리..
	Running         bool

	Keystore        *keystore.KeyStore
	ctx             *cli.Context
	LeagueRunningOK bool
	natm            nat.Interface
}

func New(c *cli.Context) (*FairNode, error) {
	// TODO : andus >> account, passphrase
	keypath := c.String("keypath")                                //$HOME/.fairnode/key
	keyfile := filepath.Join(c.String("keypath"), "fairkey.json") //$HOME/.fairnode/key/fairkey.json
	pass := c.String("password")
	natdesc := c.String("nat")

	natm, err := nat.Parse(natdesc)
	if err != nil {
		log.Fatalf("-nat: %v", err)
	}

	if _, err := os.Stat(keypath); err != nil {
		log.Fatalf("Keyfile not exists at %s.", keypath)
		return nil, err
	}

	LAddrUDP, err := net.ResolveUDPAddr("udp", ":60002") //60002
	if err != nil {
		log.Println("andus >> ResolveUDPAddr, LocalAddr", err)
		return nil, err
	}

	LAddrTCP, err := net.ResolveTCPAddr("tcp", ":"+c.String("tcp")) //60001
	if err != nil {
		log.Println("andus >> ResolveTCPAddr, LocalAddr", err)
		return nil, err
	}

	fnNode := &FairNode{
		ctx:             c,
		LaddrTcp:        LAddrTCP,
		LaddrUdp:        LAddrUDP,
		keypath:         c.String("keypath"),
		startSignalCh:   make(chan struct{}),
		LeagueRunningOK: false,
		Db:              db.New(c.String("dbhost"), c.String("dbport"), "11111"),
		natm:            natm,
	}

	fnNode.Keystore = keystore.NewKeyStore(keypath, keystore.StandardScryptN, keystore.StandardScryptP)
	blob, err := ioutil.ReadFile(keyfile)
	if err != nil {
		log.Println("Failed to read account key contents %s , %s", keypath, err)
	}
	acc, err := fnNode.Keystore.Import(blob, pass, pass)
	if err != nil {
		log.Println("Failed to import faucet signer account : %s ", err)
	}

	fnNode.Keystore.Unlock(acc, pass)

	fnNode.Account = acc

	if privkey := fnNode.Keystore.GetUnlockedPrivKey(acc.Address); privkey == nil {
		return nil, errors.New("andus >> 패어노드키가 언락 되지 않았습니다")
	} else {
		fnNode.Privkey = privkey
	}

	return fnNode, nil
}

func (f *FairNode) Start() error {
	f.Running = true

	udpConn, err := net.ListenUDP("udp", f.LaddrUdp)
	if err != nil {
		log.Println("Udp Server", err)
	}

	realaddr := udpConn.LocalAddr().(*net.UDPAddr)
	if f.natm != nil {
		if !realaddr.IP.IsLoopback() {
			go nat.Map(f.natm, nil, "udp", realaddr.Port, realaddr.Port, "andus fairnode discovery")
		}
		// TODO: react to external IP changes over time.
		if ext, err := f.natm.ExternalIP(); err == nil {
			realaddr = &net.UDPAddr{IP: ext, Port: realaddr.Port}
		}
	}

	f.UdpConn = udpConn

	go f.ListenUDP()
	//go fairNode.ListenTCP()

	return nil
}

func (f *FairNode) ListenUDP() {

	go f.manageActiveNode()
	// TODO : andus >> otprn 생성, 서명, 전송
	go f.startLeague()
	//go f.makeLeague(f.startSignalCh, f.startMakeLeague)
}

func (f *FairNode) ListenTCP() error {
	// TODO : andus >> 1. 접속한 GETH노드의 채굴 참여 가능 여부 확인 ( 참여 검증 )
	//
	//
	//
	// _ := fairutil.IsJoinOK()
	// TODO : andus >> 참여자 별로 가능여부 체크 후, 불가한 노드는 disconnect

	// TODO : andus >> 2. 채굴 가능 노드들의 enode값 저장

	log.Println(" @ go func() sendLeague START !! ")
	go f.sendLeague(f.startMakeLeague)

	// TODO : andus >> 위닝블록이 수신되는 곳 >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	// TODO : andus >> 1. 수신 블록 검증 ( sig, hash )
	// TODO : andus >> 2. 검증된 블록을 MongoDB에 저장 ( coinbase, RAND, 보낸 놈, 블록 번호 )
	// TODO : andus >> 3. 기다림........
	// TODO : andus >> 3-1 채굴참여자 수 만큼 투표가 진행 되거나, 아니면 15초후에 투표 종료

	count := 0
	leagueCount := 10 // TODO : andus >> 리그 채굴 참여자
	endOfLeagueCh := make(chan interface{})

	if count == leagueCount {
		endOfLeagueCh <- "보내.."
	}

	t := time.NewTicker(15 * time.Second)
	log.Println(" @ go func() START !! ")
	go func() {
		for {
			select {
			case <-t.C:
				// TODO : andus >> 투표 결과 서명해서, TCP로 보내준다
				// TODO : andus >> types.TransferBlock{}의 타입으로 전송할것..
				// TODO : andus >> 받은 블록의 블록헤더의 해시를 이용해서 서명후, FairNodeSig에 넣어서 보낼것.
				// TODO : andus >> 새로운 리그 시작
				f.LeagueRunningOK = false

			case <-endOfLeagueCh:
				// TODO : andus >> 투표 결과 서명해서, TCP로 보내준다
				// TODO : andus >> types.TransferBlock{}의 타입으로 전송할것..
				// TODO : andus >> 받은 블록의 블록헤더의 해시를 이용해서 서명후, FairNodeSig에 넣어서 보낼것.
				f.LeagueRunningOK = false

			}
		}
	}()

	return nil
}

// 활성 노드 관리 ( upd enode 수신, 저장, 업데이트 )
func (f *FairNode) manageActiveNode() {
	// TODO : andus >> Geth node Heart beat update ( Active node 관리 )
	// TODO : enode값 수신
	buf := make([]byte, 4096)
	for {
		n, addr, err := f.UdpConn.ReadFromUDP(buf)
		log.Println("andus >> enode값 수신", string(buf[:n]), " from ", addr)

		if err != nil {
			fmt.Println("andus >>", err)
		}

		// TODO : andus >> rlp enode 디코드
		var enodeUrl string
		rlp.DecodeBytes(buf, &enodeUrl)
		node, err := discv5.ParseNode(enodeUrl)
		log.Println(enodeUrl, node)

		// TODO : andus >> DB에 insert Or Update
		f.Db.SaveActiveNode()
		if err != nil {
			log.Println("andus >> Udp enode 수신중 에러", err)
		}

		f.UdpConn.Write([]byte("OTPRN 보냅니다"))
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
			// TODO : andus >> active node 조회 3개이상
			log.Println("andus >> 디비에서 엑티브 노드 조회")

			actNum := 3
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

				fmt.Println("andus >> tsOtp", tsOtp.Hash.String())

				ts, err := rlp.EncodeToBytes(tsOtp)
				if err != nil {
					log.Println("andus >> Otprn rlp 인코딩 에러", err)
				}

				// TODO : andus >> DB에서 Active node 리스트를 조회
				activeNodeList := []string{":50002"}
				for index := range activeNodeList {
					ServerAddr, err := net.ResolveUDPAddr("udp", activeNodeList[index])
					if err != nil {
						log.Println("andus >>", err)
					}
					Conn, err := net.DialUDP("udp", nil, ServerAddr)
					if err != nil {
						log.Println("andus >>", err)
					}

					// TODO : andus >> Active node 노드에게 OTPRN 전송
					Conn.Write(ts)
					Conn.Close()
				}

			}
		}
	}
}

func (f *FairNode) makeLeague(startCh chan struct{}, bb chan string) {
	log.Println(" @ run makeLeague() ")
	t := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-t.C:
			log.Println(" @ in makeLeague() ")
		}
		// <- chan Start singnal // 레그 스타트

		// TODO : andus >> 리그 스타트 ( 엑티브 노드 조회 ) ->

		// TODO : andus >> 1. OTPRN 생성
		// TODO : andus >> 2. OTPRN Hash
		// TODO : andus >> 3. Fair Node 개인키로 암호화
		// TODO : andus >> 4. OTPRN 값 + 전자서명값 을 전송
		// TODO : andus >> 5. UDP 전송
		// TODO : andus >> 6. UDP 전송 후 참여 요청 받을 때 까지 기다릴 시간( 3s )후
		// TODO : andus >> 7. 리스 시작 채널에 메세지 전송
		//bb <- "리그시작"

		// close(Start singnal)
	}
}

func (f *FairNode) sendLeague(aa chan string) {
	for {
		<-aa
		// TODO : andus >> 1. 채굴참여자 조회 ( from DB )
		// TODO : andus >> 2. 채굴 리그 구성

		var league []map[string]string

		leagueHash := f.makeHash(league) // TODO : andsu >> 전체 채굴리그의 해시값

		for key, value := range fairutil.GetPeerList() {
			//key = to,
			//value = 접속할 peer list

			fmt.Println(leagueHash, key, value)
			// TODO : andus >> 각 GETH 노드에게 연결할 peer 리스트 전달 + 전체 채굴리그의 해시값 ( leagueHash )
			// TODO : andus >> 추후 서명 예정....
		}
	}
}

func (f *FairNode) makeHash(list []map[string]string) common.Hash {

	return common.Hash{}
}

func (f *FairNode) Stop() {
	f.Running = false
	f.UdpConn.Close()
	f.TcpConn.Close()
}

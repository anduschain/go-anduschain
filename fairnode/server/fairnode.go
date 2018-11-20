package server

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/accounts"
	"github.com/anduschain/go-anduschain/accounts/keystore"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/fairnode/server/db"
	"github.com/anduschain/go-anduschain/p2p/nat"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
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

	UdpConn *net.UDPConn
	TcpConn *net.TCPListener

	Db *db.FairNodeDB

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

	LAddrUDP, err := net.ResolveUDPAddr("udp", ":60003") //60002
	if err != nil {
		log.Println("andus >> ResolveUDPAddr, LocalAddr", err)
		return nil, err
	}

	LAddrTCP, err := net.ResolveTCPAddr("tcp", ":60003") //60002
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

	tcpConn, err := net.ListenTCP("tcp", f.LaddrTcp)
	if err != nil {
		fmt.Println("andus >> ListenTCP 에러!!", err)
	}

	laddr := tcpConn.Addr().(*net.TCPAddr)
	// Map the TCP listening port if NAT is configured.
	if !laddr.IP.IsLoopback() {
		go func() {
			nat.Map(f.natm, nil, "tcp", laddr.Port, laddr.Port, "andus fairnode discovery")
		}()
	}

	f.UdpConn = udpConn
	f.TcpConn = tcpConn

	go f.ListenUDP()
	go f.ListenTCP()

	return nil
}

func (f *FairNode) Stop() {
	f.Running = false
	f.UdpConn.Close()
	f.TcpConn.Close()
	f.Db.Mongo.Close()
}

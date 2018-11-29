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
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
)

// TODO : andus >> timezone 셋팅

var (
	FairnodeKeyError     = errors.New("패어노드키가 언락 되지 않았습니다")
	KeyFileNotExistError = errors.New("페어노드의 키가 생성되지 않았습니다. 생성해 주세요.")
	NATinitError         = errors.New("NAT 설정에 문제가 있습니다")
	KeyPassError         = errors.New("패어노드 키가 입력되지 않았습니다.")
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

	otprn             *otprn.Otprn
	SingedOtprn       *string // 전자서명값
	sendLeagueStartCh chan string
	Wg                sync.WaitGroup
	lock              sync.RWMutex
	StopCh            chan struct{} // TODO : andus >> 죽을때 처리..
	Running           bool

	Keystore        *keystore.KeyStore
	LeagueRunningOK bool
	natm            nat.Interface

	//LeagueList []map[string][]string

	// 리그가 시작되었을때 커넥션 관리
	LeagueConList []*net.TCPConn
}

func New() (*FairNode, error) {
	keypath := DefaultConfig.KeyPath                  // for UNIX $HOME/.fairnode/key
	keyfile := filepath.Join(keypath, "fairkey.json") // for UNIX $HOME/.fairnode/key/fairkey.json
	keyPass := DefaultConfig.KeyPass

	if keyPass == "" {
		return nil, KeyPassError
	}

	natm, err := nat.Parse(DefaultConfig.NAT)
	if err != nil {
		return nil, NATinitError
	}

	if _, err := os.Stat(DefaultConfig.KeyPath); err != nil {
		return nil, KeyFileNotExistError
	}

	addr := fmt.Sprintf(":%s", DefaultConfig.Port)

	LAddrUDP, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	LAddrTCP, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}

	mongoDB, err := db.New(DefaultConfig.DBhost, DefaultConfig.DBport, DefaultConfig.DBpass, DefaultConfig.DBuser)
	if err != nil {
		return nil, err
	}

	fnNode := &FairNode{
		LaddrTcp:          LAddrTCP,
		LaddrUdp:          LAddrUDP,
		keypath:           DefaultConfig.KeyPath,
		sendLeagueStartCh: make(chan string),
		LeagueRunningOK:   false,
		Db:                mongoDB,
		natm:              natm,
	}

	fnNode.Keystore = keystore.NewKeyStore(keypath, keystore.StandardScryptN, keystore.StandardScryptP)
	blob, err := ioutil.ReadFile(keyfile)
	if err != nil {
		return nil, err
	}
	acc, err := fnNode.Keystore.Import(blob, DefaultConfig.KeyPass, DefaultConfig.KeyPass)
	if err != nil {
		return nil, err
	}

	if err := fnNode.Keystore.Unlock(acc, DefaultConfig.KeyPass); err == nil {
		fnNode.Account = acc

		if privkey, ok := fnNode.Keystore.GetUnlockedPrivKey(acc.Address); ok {
			fnNode.Privkey = privkey
		} else {
			return nil, FairnodeKeyError
		}
	} else {
		return nil, FairnodeKeyError
	}

	return fnNode, nil
}

func (f *FairNode) Start() error {
	var err error

	f.Running = true
	f.UdpConn, err = net.ListenUDP("udp", f.LaddrUdp)
	if err != nil {
		log.Println("Error : Udp Server", err)
		return err
	}

	if f.natm != nil {
		realaddr := f.UdpConn.LocalAddr().(*net.UDPAddr)
		if f.natm != nil {
			if !realaddr.IP.IsLoopback() {
				go nat.Map(f.natm, nil, "udp", realaddr.Port, realaddr.Port, "andus fairnode discovery")
			}
			// TODO: react to external IP changes over time.
			if ext, err := f.natm.ExternalIP(); err == nil {
				realaddr = &net.UDPAddr{IP: ext, Port: realaddr.Port}
			}
		}
	}

	f.TcpConn, err = net.ListenTCP("tcp", f.LaddrTcp)
	if err != nil {
		log.Println("Error : ListenTCP 에러!!", err)
		return err
	}

	if f.natm != nil {
		laddr := f.TcpConn.Addr().(*net.TCPAddr)
		// Map the TCP listening port if NAT is configured.
		if !laddr.IP.IsLoopback() {
			go func() {
				nat.Map(f.natm, nil, "tcp", laddr.Port, laddr.Port, "andus fairnode discovery")
			}()
		}
	}

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

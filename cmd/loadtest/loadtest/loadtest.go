package loadtest

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/anduschain/go-anduschain/accounts/keystore"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/ethclient"
	"github.com/anduschain/go-anduschain/params"
	"github.com/anduschain/go-anduschain/rlp"
	"github.com/anduschain/go-anduschain/rpc"
	"github.com/pkg/errors"
	"io/ioutil"
	"math/big"
	"net/url"
	"strings"
	"time"
)

var (
	price = new(big.Int).Mul(big.NewInt(1), big.NewInt(params.Daon))
	fee   = 0.00000001 * params.Daon

	chainID = big.NewInt(1315)
	signer  = types.NewEIP155Signer(chainID)
)

type Account struct {
	Address string
	Url     string
}

type Keystore struct {
	Accounts []Account
	Status   string
	Url      string
}

type LoadTest struct {
	rc    *rpc.Client
	ec    *ethclient.Client
	ks    *ecdsa.PrivateKey
	addr  common.Address
	pwd   string
	nonce uint64
}

func NewLoadTestModule(client *rpc.Client, addr, pwd string) *LoadTest {
	return &LoadTest{
		rc:   client,
		ec:   ethclient.NewClient(client),
		addr: common.HexToAddress(addr),
		pwd:  pwd,
	}
}

func (l *LoadTest) GetPrivateKey() error {
	res := []Keystore{}
	var rawurl string
	err := l.rc.CallContext(context.Background(), &res, "personal_listWallets")
	if err != nil {
		fmt.Println("GetPrivateKey", err)
		return nil
	}

	for i := range res {
		if strings.Compare(res[i].Accounts[0].Address, strings.ToLower(l.addr.String())) == 0 {
			rawurl = res[i].Accounts[0].Url
			break
		}
	}

	u, err := url.Parse(rawurl)
	if err != nil {
		return errors.New(fmt.Sprintf("GetPrivateKey", err))
	}

	// Open the account key file
	keyJson, readErr := ioutil.ReadFile(u.Path)
	if readErr != nil {
		return errors.New(fmt.Sprintf("key json read error"))
	}

	// Get the private key
	keyWrapper, keyErr := keystore.DecryptKey(keyJson, l.pwd)
	if keyErr != nil {
		return errors.New(fmt.Sprintf("key decrypt error"))
	}

	l.ks = keyWrapper.PrivateKey
	return nil
}

func (l *LoadTest) UnlockAccount() bool {
	var result bool
	err := l.rc.CallContext(context.Background(), &result, "personal_unlockAccount", l.addr, l.pwd)
	if err != nil {
		fmt.Println("rpcClient.CallContext", err)
		return false
	}
	return result
}

func (l *LoadTest) CheckBalance() bool {
	balance, err := l.ec.BalanceAt(context.Background(), l.addr, nil)
	if err != nil {
		return false
	}

	if balance.Cmp(price) > 0 {
		return true
	}

	return false
}

func (l *LoadTest) GetNonce() error {
	nonce, err := l.ec.PendingNonceAt(context.Background(), l.addr)
	if err != nil {
		return err
	}

	l.nonce = nonce
	return nil
}

func (l *LoadTest) SendTransaction() error {
	fmt.Println("nonce", l.nonce)

	value := price
	gasLimit := uint64(100000000)
	gasPrice := big.NewInt(40 * params.GWei)
	toAddress := l.addr

	data := struct {
		Payload string
		Time    time.Time
	}{"loadTest transaction", time.Now()}

	bData, err := rlp.EncodeToBytes(data)
	if err != nil {
		return err
	}

	tx := types.NewTransaction(l.nonce, toAddress, value, gasLimit, gasPrice, bData)
	signedTx, err := types.SignTx(tx, signer, l.ks)
	if err != nil {
		return err
	}

	fmt.Println("signedTx.Hash()", signedTx.Hash().String())

	err = l.ec.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return err
	}

	l.nonce++

	fmt.Println("tx sent", signedTx.Hash().Hex(), l.nonce)

	return nil
}

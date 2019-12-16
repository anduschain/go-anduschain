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
	"log"
	"math/big"
	"net/url"
	"strings"
	"time"
)

var (
	price = new(big.Int).Mul(big.NewInt(1), big.NewInt(params.Daon))
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
	rc     *rpc.Client
	ec     *ethclient.Client
	ks     *ecdsa.PrivateKey
	addr   common.Address
	pwd    string
	nonce  uint64
	signer types.EIP155Signer
}

func NewLoadTestModule(client *rpc.Client, addr, pwd string, chainID int64) *LoadTest {
	return &LoadTest{
		rc:     client,
		ec:     ethclient.NewClient(client),
		addr:   common.HexToAddress(addr),
		pwd:    pwd,
		signer: types.NewEIP155Signer(big.NewInt(chainID)),
	}
}

func (l *LoadTest) GetPrivateKey() error {
	res := []Keystore{}
	var rawurl string
	err := l.rc.CallContext(context.Background(), &res, "personal_listWallets")
	if err != nil {
		return err
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

	if strings.Compare(u.Path, "") == 0 {
		return errors.New("path is nil")
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

func (l *LoadTest) UnlockAccount() error {
	var result bool
	err := l.rc.CallContext(context.Background(), &result, "personal_unlockAccount", l.addr, l.pwd)
	if err != nil {
		return err
	}
	return nil
}

func (l *LoadTest) CheckBalance() error {
	balance, err := l.ec.BalanceAt(context.Background(), l.addr, nil)
	if err != nil {
		return err
	}

	if balance.Cmp(price) < 0 {
		return errors.New("잔액이 부족합니다")
	}

	return nil
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
	value := price
	gasLimit := 10 * params.TxGas
	gasPrice, _ := l.ec.SuggestGasPrice(context.Background())
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
	signedTx, err := types.SignTx(tx, l.signer, l.ks)
	if err != nil {
		return err
	}

	err = l.ec.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return err
	}

	l.nonce++
	log.Println("tx sent", signedTx.Hash().Hex())
	return nil
}

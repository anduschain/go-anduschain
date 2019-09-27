// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"github.com/anduschain/go-anduschain/params"
	"math/big"
	"testing"

	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/rlp"
)

// The values in those tests are from the Transaction Tests
// at github.com/ethereum/tests.
var (
	emptyTx = NewTransaction(
		0,
		common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"),
		big.NewInt(0), 0, big.NewInt(0),
		nil,
	)

	rightvrsTx, _ = NewTransaction(
		3,
		common.HexToAddress("b94f5374fce5edbc8e2a8697c15331677e6ebf0b"),
		big.NewInt(10),
		2000,
		big.NewInt(1),
		common.FromHex("5544"),
	).WithSignature(
		HomesteadSigner{},
		common.Hex2Bytes("98ff921201554726367d2be8c804a7ff89ccf285ebc57dff8ae4c44b9c19ac4a8887321be575c8095f789dd4c743dfe42c1820f9231f98a962b210e3ac2452a301"),
	)
)

func TestNewJoinTransaction(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("could not generate key: %v", err)
	}
	signer := NewEIP155Signer(common.Big1)

	otprn := NewOtprn(100, fairaddress, chainConfig)
	t.Log("origin otprn", otprn.HashOtprn().String())

	bOtrpn, err := otprn.EncodeOtprn()
	if err != nil {
		t.Error("otprn encode err", err)
	}

	jtx := NewJoinTransaction(0, 0, bOtrpn, common.Address{})
	t.Log("Join transaction", jtx.Hash().String())

	jnonce, err := jtx.JoinNonce()
	if err != nil {
		t.Error("join transaction jnonce", err)
	}

	t.Log("join nonce", jnonce)

	totprn, err := jtx.Otprn()
	if err != nil {
		t.Error("join transaction jnonce", err)
	}

	dotp, err := DecodeOtprn(totprn)
	if err != nil {
		t.Error("join transaction DecodeOtprn", err)
	}

	t.Log("join transaction otprn", dotp.HashOtprn().String())
	t.Log("++++++++++++++++++++++++++++++++++")

	hash, _ := jtx.PayloadHash()
	rHash := rlpHash([]interface{}{jnonce, bOtrpn, common.Address{}})

	if bytes.Compare(hash, rHash.Bytes()) != 0 {
		t.Error("join transaction payload recover")
	}

	t.Log("++++++++++++++++++++++++++++++++++")

	sjtx, err := SignTx(jtx, signer, key)
	if err != nil {
		t.Error("join transaction SignTx", err)
	}

	t.Log("signed Join transaction", sjtx.Hash().String())
	t.Log("signed Join transaction to", sjtx.To().String())
	v, r, s := sjtx.RawSignatureValues()
	t.Log("signed signature", "v :", v, "r :", r, "s :", s)
	t.Log("signed signature", "transactionid", "isJointx", sjtx.TransactionId() == JoinTx)

	encodedJointx, err := rlp.EncodeToBytes(sjtx)
	if err != nil {
		t.Error("join transaction EncodeToBytes", err)
	}
	t.Log("encodedJointx", common.BytesToHash(encodedJointx).String())

	t.Log("++++++++++++++++++++++++++++++++++")

	var decodedJtx *Transaction
	err = rlp.DecodeBytes(encodedJointx, &decodedJtx)
	if err != nil {
		t.Error("join transaction DecodeBytes", err)
	}
	t.Log("decoded Join transaction to", sjtx.To().String())
	t.Log("decoded Join transaction", decodedJtx.Hash().String())
	dv, dr, ds := decodedJtx.RawSignatureValues()
	t.Log("decoded signature", "v :", dv, "r :", dr, "s :", ds)
	t.Log("decoded signature", "transactionid", "isJointx", decodedJtx.TransactionId() == JoinTx)

}

func TestTransaction_MarshalJSON(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("could not generate key: %v", err)
	}
	signer := NewEIP155Signer(common.Big1)

	otprn := NewOtprn(100, fairaddress, chainConfig)
	bOtrpn, err := otprn.EncodeOtprn()
	if err != nil {
		t.Error("otprn encode err", err)
	}
	jtx := NewJoinTransaction(0, 0, bOtrpn, common.Address{})
	sjtx, err := SignTx(jtx, signer, key)
	if err != nil {
		t.Error("join transaction SignTx", err)
	}
	t.Log("Join transaction", sjtx.Hash().String())
	jsonJtx, err := sjtx.MarshalJSON()
	if err != nil {
		t.Error("join transaction MarshalJSON", err)
	}
	newTx := new(Transaction)
	err = newTx.UnmarshalJSON(jsonJtx)
	if err != nil {
		t.Error("join transaction UnmarshalJSON", err)
	}
	t.Log("Join UnmarshalJSON transaction", newTx.Hash().String())

	t.Log("++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")
	gentx := NewTransaction(1, common.HexToAddress("b94f5374fce5edbc8e2a8697c15331677e6ebf0b"), big.NewInt(10), 2000, big.NewInt(1), common.FromHex("5544"))
	signedGentx, err := SignTx(gentx, signer, key)
	if err != nil {
		t.Error("gentx signedGentx", err)
	}
	t.Log("rightvrsTx transaction", signedGentx.Hash().String())
	btx, err := signedGentx.MarshalJSON()
	if err != nil {
		t.Error("signedGentx transaction MarshalJSON", err)
	}

	fmt.Println(string(btx))

	newTx2 := new(Transaction)
	err = newTx2.UnmarshalJSON(btx)
	if err != nil {
		t.Error("signedGentx transaction UnmarshalJSON", err)
	}
	t.Log("signedGentx UnmarshalJSON transaction", newTx2.Hash().String())
}

func TestTransactionSigHash(t *testing.T) {
	var homestead HomesteadSigner
	if homestead.Hash(emptyTx) != common.HexToHash("c775b99e7ad12f50d819fcd602390467e28141316969f4b57f0626f74fe3b386") {
		t.Errorf("empty transaction hash mismatch, got %x", emptyTx.Hash())
	}
	if homestead.Hash(rightvrsTx) != common.HexToHash("fe7a79529ed5f7c3375d06b26b186a8644e0e16c373d7a12be41c62d6042b77a") {
		t.Errorf("RightVRS transaction hash mismatch, got %x", rightvrsTx.Hash())
	}
}

func TestTransacionEncodeDecode(t *testing.T) {
	pkey, err := crypto.HexToECDSA("abd1374002952e5f3b60fd16293d06521e1557d9dfb9996561568423a26a9cc9")
	if err != nil {
		t.Error(err)
	}
	signer := NewEIP155Signer(params.TEST_NETWORK)
	tx := NewTransaction(1, common.HexToAddress("0x160A2189e10Ec349e176d8Fa6Fccf4a19ED6d133"), big.NewInt(1), 200000000, big.NewInt(100), nil)
	sTx, err := SignTx(tx, signer, pkey)
	if err != nil {
		t.Error("tx signedGentx", err)
	}

	jsonTx, err := sTx.MarshalJSON()
	if err != nil {
		t.Error("tx MarshalJSON", err)
	}

	unTx := new(Transaction)
	if err := unTx.UnmarshalJSON(jsonTx); err != nil {
		t.Error("tx UnmarshalJSON", err)
	}

	t.Log("Signed Transastion", sTx.Hash().String())

	buf := bytes.Buffer{}
	if err := sTx.EncodeRLP(&buf); err != nil {
		t.Error("tx EncodeRLP", err)
	}

	t.Log(common.Bytes2Hex(buf.Bytes()))

	newTx := new(Transaction)
	if err := newTx.DecodeRLP(rlp.NewStream(&buf, uint64(buf.Len()))); err != nil {
		t.Error("tx DecodeRLP", err)
	}

	t.Log("Decoded Transastion", newTx.Hash().String())
}

func TestTransaction_DecodeRLP(t *testing.T) {
	encodedTx := "f866800164840bebc20094160a2189e10ec349e176d8fa6fccf4a19ed6d13301808401b40e25a07c91493ca956ccd46cb4b4960244a6e062f71a874157d6191315e40a0e2417d1a00928a8e437c8a37e9280d81601e41e1573e04ebb4fc7fc245913c54613e33073"
	b := common.Hex2Bytes(encodedTx)
	newTx := new(Transaction)
	if err := rlp.DecodeBytes(b, newTx); err != nil {
		t.Error("tx DecodeRLP", err)
	}

	jsonTx, err := newTx.MarshalJSON()
	if err != nil {
		t.Error("tx MarshalJSON", err)
	}

	addr, err := newTx.Sender(NewEIP155Signer(params.TEST_NETWORK))
	if err != nil {
		t.Error("tx Sender", err)
	}

	t.Log("Sender :", addr.String())
	t.Log("JSON : ", string(jsonTx))
	t.Log("Hash : ", newTx.Hash().String())
}

//func TestTransactionEncode(t *testing.T) {
//	txb, err := rlp.EncodeToBytes(rightvrsTx)
//	if err != nil {
//		t.Fatalf("encode error: %v", err)
//	}
//	should := common.FromHex("f86103018207d094b94f5374fce5edbc8e2a8697c15331677e6ebf0b0a8255441ca098ff921201554726367d2be8c804a7ff89ccf285ebc57dff8ae4c44b9c19ac4aa08887321be575c8095f789dd4c743dfe42c1820f9231f98a962b210e3ac2452a3")
//	if !bytes.Equal(txb, should) {
//		t.Errorf("encoded RLP mismatch, got %x", txb)
//	}
//}

func decodeTx(data []byte) (*Transaction, error) {
	var tx Transaction
	t, err := &tx, rlp.Decode(bytes.NewReader(data), &tx)

	return t, err
}

func defaultTestKey() (*ecdsa.PrivateKey, common.Address) {
	key, _ := crypto.HexToECDSA("45a915e4d060149eb4365960e6a7a45f334393093061116b197e3240065ff2d8")
	addr := crypto.PubkeyToAddress(key.PublicKey)
	return key, addr
}

//func TestRecipientEmpty(t *testing.T) {
//	_, addr := defaultTestKey()
//	tx, err := decodeTx(common.Hex2Bytes("f8498080808080011ca09b16de9d5bdee2cf56c28d16275a4da68cd30273e2525f3959f5d62557489921a0372ebd8fb3345f7db7b5a86d42e24d36e983e259b0664ceb8c227ec9af572f3d"))
//	if err != nil {
//		t.Error(err)
//		t.FailNow()
//	}
//
//	from, err := tx.Sender(HomesteadSigner{})
//	if err != nil {
//		t.Error(err)
//		t.FailNow()
//	}
//	if addr != from {
//		t.Error("derived address doesn't match")
//	}
//}
//
//func TestRecipientNormal(t *testing.T) {
//	_, addr := defaultTestKey()
//
//	tx, err := decodeTx(common.Hex2Bytes("f85d80808094000000000000000000000000000000000000000080011ca0527c0d8f5c63f7b9f41324a7c8a563ee1190bcbf0dac8ab446291bdbf32f5c79a0552c4ef0a09a04395074dab9ed34d3fbfb843c2f2546cc30fe89ec143ca94ca6"))
//	if err != nil {
//		t.Error(err)
//		t.FailNow()
//	}
//
//	from, err := tx.Sender(HomesteadSigner{})
//	if err != nil {
//		t.Error(err)
//		t.FailNow()
//	}
//
//	if addr != from {
//		t.Error("derived address doesn't match")
//	}
//}

// Tests that transactions can be correctly sorted according to their price in
// decreasing order, but at the same time with increasing nonces when issued by
// the same account.
func TestTransactionPriceNonceSort(t *testing.T) {
	// Generate a batch of accounts to start with
	keys := make([]*ecdsa.PrivateKey, 25)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
	}

	signer := HomesteadSigner{}
	// Generate a batch of transactions with overlapping values, but shifted nonces
	groups := map[common.Address]Transactions{}
	for start, key := range keys {
		addr := crypto.PubkeyToAddress(key.PublicKey)
		for i := 0; i < 25; i++ {
			tx, _ := SignTx(NewTransaction(uint64(start+i), common.Address{}, big.NewInt(100), 100, big.NewInt(int64(start+i)), nil), signer, key)
			groups[addr] = append(groups[addr], tx)
		}
	}
	// Sort the transactions and cross check the nonce ordering
	txset := NewTransactionsByPriceAndNonce(signer, groups)

	txs := Transactions{}
	for tx := txset.Peek(); tx != nil; tx = txset.Peek() {
		txs = append(txs, tx)
		txset.Shift()
	}
	if len(txs) != 25*25 {
		t.Errorf("expected %d transactions, found %d", 25*25, len(txs))
	}
	for i, txi := range txs {
		fromi, _ := txi.Sender(signer)

		// Make sure the nonce order is valid
		for j, txj := range txs[i+1:] {
			fromj, _ := txj.Sender(signer)

			if fromi == fromj && txi.Nonce() > txj.Nonce() {
				t.Errorf("invalid nonce ordering: tx #%d (A=%x N=%v) < tx #%d (A=%x N=%v)", i, fromi[:4], txi.Nonce(), i+j, fromj[:4], txj.Nonce())
			}
		}

		// If the next tx has different from account, the price must be lower than the current one
		if i+1 < len(txs) {
			next := txs[i+1]
			fromNext, _ := next.Sender(signer)
			if fromi != fromNext && txi.GasPrice().Cmp(next.GasPrice()) < 0 {
				t.Errorf("invalid gasprice ordering: tx #%d (A=%x P=%v) < tx #%d (A=%x P=%v)", i, fromi[:4], txi.GasPrice(), i+1, fromNext[:4], next.GasPrice())
			}
		}
	}
}

// TestTransactionJSON tests serializing/de-serializing to/from JSON.
func TestTransactionJSON(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("could not generate key: %v", err)
	}
	signer := NewEIP155Signer(common.Big1)

	for i := uint64(0); i < 25; i++ {
		var tx *Transaction
		switch i % 2 {
		case 0:
			tx = NewTransaction(i, common.Address{1}, common.Big0, 1, common.Big2, []byte("abcdef"))
		case 1:
			tx = NewContractCreation(i, common.Big0, 1, common.Big2, []byte("abcdef"))
		}

		tx, err := SignTx(tx, signer, key)
		if err != nil {
			t.Fatalf("could not sign transaction: %v", err)
		}

		data, err := json.Marshal(tx)
		if err != nil {
			t.Errorf("json.Marshal failed: %v", err)
		}

		var parsedTx *Transaction
		if err := json.Unmarshal(data, &parsedTx); err != nil {
			t.Errorf("json.Unmarshal failed: %v", err)
		}

		// compare nonce, price, gaslimit, recipient, amount, payload, V, R, S
		if tx.Hash() != parsedTx.Hash() {
			t.Errorf("parsed tx differs from original tx, want %v, got %v", tx, parsedTx)
		}
		if tx.ChainId().Cmp(parsedTx.ChainId()) != 0 {
			t.Errorf("invalid chain id, want %d, got %d", tx.ChainId(), parsedTx.ChainId())
		}
	}
}

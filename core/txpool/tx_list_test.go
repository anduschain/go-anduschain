package txpool

import (
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	txType "github.com/anduschain/go-anduschain/core/transaction"
	"testing"
)

func TestTxList_Remove(t *testing.T) {
	genTx := txType.NewGenTransaction(0, common.Address{1}, common.Big0, 1, common.Big2, []byte("abcdef"))

	txTypeCheck(genTx)

	t.Log(genTx.Nonce())
}

func txTypeCheck(tx interface{}) {
	switch v := tx.(type) {
	case *txType.GenTransaction:
		fmt.Print("**********", v.Nonce())
	}
}

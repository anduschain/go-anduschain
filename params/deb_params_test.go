package params

import (
	"fmt"
	"github.com/anduschain/go-anduschain/common/hexutil"
	"math/big"
	"testing"
)

func TestMinGasPrice(t *testing.T) {
	// cost == V + GP * GL

	cost := new(big.Int).Mul(big.NewInt(1), big.NewInt(Daon))
	GL := big.NewInt(21000)
	V := new(big.Int).Mul(big.NewInt(8), big.NewInt(1e17))
	sub := new(big.Int).Sub(cost, V) // cost - v
	GP := new(big.Int).Div(sub, GL)

	fmt.Println(GP.String())
}

func TestCalBnToHex(t *testing.T) {
	fmt.Println(hexutil.EncodeBig(new(big.Int).Mul(big.NewInt(94e8), big.NewInt(Daon))))
	fmt.Println(hexutil.EncodeBig(new(big.Int).Mul(big.NewInt(1e8), big.NewInt(Daon))))
}

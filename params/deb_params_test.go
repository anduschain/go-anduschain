package params

import (
	"fmt"
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

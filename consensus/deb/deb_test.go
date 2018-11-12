package deb

import (
	"fmt"
	"math/big"
	"testing"
)

func TestMakeRand(t *testing.T) {

	rand := big.NewInt(0)

	for i := 0; i <= int(10); i++ {
		newRand := big.NewInt(int64(i))
		if newRand.Cmp(rand) > 0 {
			rand = newRand
		}

		fmt.Println(rand.String(), newRand.String())
	}

}

package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

func main() {
	nBig, err := rand.Int(rand.Reader, big.NewInt(99999999999999999))
	if err != nil {
		panic(err)
	}
	n := nBig.Int64()
	fmt.Printf("Here is a random %T in [0,99999999999999999) : %d\n", n, n)
}

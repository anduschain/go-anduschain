package bls12381

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"os"
	"testing"
)

var fuz int

func TestMain(m *testing.M) {
	fmt.Println("Test Start")
	_fuz := flag.Int("fuzz", 10, "# of iterations")
	flag.Parse()
	fuz = *_fuz
	os.Exit(m.Run())
}

func randScalar(max *big.Int) *big.Int {
	fmt.Println("randScalar")
	a, err := rand.Int(rand.Reader, max)
	if err != nil {
		panic(errors.New(""))
	}
	return a
}

func fromHex(size int, hexStrs ...string) []byte {
	fmt.Println("fromHex")
	var out []byte
	if size > 0 {
		out = make([]byte, size*len(hexStrs))
	}
	for i := 0; i < len(hexStrs); i++ {
		hexStr := hexStrs[i]
		if hexStr[:2] == "0x" {
			hexStr = hexStr[2:]
		}
		if len(hexStr)%2 == 1 {
			hexStr = "0" + hexStr
		}
		bytes, err := hex.DecodeString(hexStr)
		if err != nil {
			return nil
		}
		if size <= 0 {
			out = append(out, bytes...)
		} else {
			if len(bytes) > size {
				return nil
			}
			offset := i*size + (size - len(bytes))
			copy(out[offset:], bytes)
		}
	}
	return out
}

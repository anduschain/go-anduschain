package main

import (
	"log"
	"math/big"
	"testing"
)

func TestNewWizardcalGasPrice(t *testing.T) {
	tCast := []struct {
		Fee   float64
		Price *big.Int
	}{
		{0.2, big.NewInt(9523809523809)},
		{0.3, big.NewInt(14285714285714)},
	}

	for _, c := range tCast {
		price := calGasPrice(c.Fee)
		log.Printf("Fee %f Daon GasPrice %s Result = %t", c.Fee, price.String(), price.Cmp(c.Price) == 0)
	}
}

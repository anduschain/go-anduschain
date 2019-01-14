package fairudp

import (
	"testing"
)

func TestNew(t *testing.T) {
	for i := 0; i < 1000; i++ {
		if i%200 == 0 {
			t.Log("Epoch -----", i)
		}
	}
}

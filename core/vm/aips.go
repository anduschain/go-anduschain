// Copyright 2019 The go-ethereum Authors
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

package vm

import (
	"fmt"
	"sort"
)

var activators = map[int]func(*JumpTable){
	1000: enable1000, // fork Pohang
}

// EnableAIP enables the given AIP on the config.
// This operation writes in-place, and callers need to ensure that the globally
// defined jump tables are not polluted.
func EnableAIP(eipNum int, jt *JumpTable) error {
	enablerFn, ok := activators[eipNum]
	if !ok {
		return fmt.Errorf("undefined eip %d", eipNum)
	}
	enablerFn(jt)
	return nil
}

func ValidAip(eipNum int) bool {
	_, ok := activators[eipNum]
	return ok
}
func ActivateableAips() []string {
	var nums []string
	for k := range activators {
		nums = append(nums, fmt.Sprintf("%d", k))
	}
	sort.Strings(nums)
	return nums
}

func enable1000(jt *JumpTable) {
	// Gas cost changes

	// New opcode
	jt[CHAINID] = &operation{
		execute:       opChainID,
		gasCost:       gasChainID,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jt[SELFBALANCE] = &operation{
		execute:       opSelfBalance,
		gasCost:       gasSelfBalance,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jt[BASEFEE] = &operation{
		execute:       opBaseFee,
		gasCost:       gasBaseFee,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	//jt[PUSH0] = &operation{
	//	execute:       opPush0,
	//	gasCost:       gasPush0,
	//	validateStack: makeStackFunc(0, 1),
	//	valid:         true,
	//}
}

// Copyright 2015 The go-ethereum Authors
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

package params

//AndusChainTestNode
var AndusChainBootnodes = []string{
	"enode://a0ca256bdde175df67b8092927bcb51be6643fc438156995b492a7ca8088f2e83514ec903a80b044321e0225f6d4ca78a070c536b15eea7a1f480b29cb0fdb98@13.209.167.161:30301",
}

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main Ethereum network.
var MainnetBootnodes = []string{
	//"enode://4ec9bb7dc72afbab998de71e4e349a5c06f9d0c2b76637a010b3f53650916a09fddc24ffe86788a2562a3bd9b58fea12d595c1a46f0f19acb6767f4906c1e74a@121.156.104.254:30301", //andus
}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Ropsten test network.
var TestnetBootnodes = []string{
	//"enode://4ec9bb7dc72afbab998de71e4e349a5c06f9d0c2b76637a010b3f53650916a09fddc24ffe86788a2562a3bd9b58fea12d595c1a46f0f19acb6767f4906c1e74a@121.156.104.254:30301", //andus
}

// RinkebyBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Rinkeby test network.
var RinkebyBootnodes = []string{}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
var DiscoveryV5Bootnodes = []string{
	//"enode://4ec9bb7dc72afbab998de71e4e349a5c06f9d0c2b76637a010b3f53650916a09fddc24ffe86788a2562a3bd9b58fea12d595c1a46f0f19acb6767f4906c1e74a@121.156.104.254:30301", //andus
}

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

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main Ethereum network.
var MainnetBootnodes = []string{
	"enode://4485245ef95c49f09ba9338207d18eb0d2de84beec7c42f7dd51da09e4b7cca850b0b66e842ddd31d6720459a6c078de829e3e63789fd006d2c1160f926ec84d@bootnode.mainnet.anduschain.io:50501",
}

//AndusChainTestNode
var TestnetBootnodes = []string{
	"enode://585232819253c97da804baefe8b0b015cd797dc784cd8b48888ff2e69f0e39815bdcdf81ea04e7823b663df7b49e075d5558fc3a0dfeea021c5147bb6540cd4c@bootnode.testnet.anduschain.io:50501",
}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
var DiscoveryV5Bootnodes = []string{}

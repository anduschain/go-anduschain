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
	"enode://dff0d4740dff20b2ee026cbe8e484f792fbc8b7c3b9a9af240a5bd579ba5d3003cba61be3303e385bfd292921423ebbbb2b8613bd0d7e34b62b773620035c11f@121.134.35.45:34500", //andus
}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Ropsten test network.
var TestnetBootnodes = []string{
	"enode://dff0d4740dff20b2ee026cbe8e484f792fbc8b7c3b9a9af240a5bd579ba5d3003cba61be3303e385bfd292921423ebbbb2b8613bd0d7e34b62b773620035c11f@121.134.35.45:34500", //andus
}

// RinkebyBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Rinkeby test network.
var RinkebyBootnodes = []string{}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
var DiscoveryV5Bootnodes = []string{
	//"enode://dff0d4740dff20b2ee026cbe8e484f792fbc8b7c3b9a9af240a5bd579ba5d3003cba61be3303e385bfd292921423ebbbb2b8613bd0d7e34b62b773620035c11f@121.134.35.45:34500",//andus
}

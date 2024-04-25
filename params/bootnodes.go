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
	"enode://b3db50144fefa4f06b0b897d735d1233fcb33bf0f3b60d3c1cdda6cd70aeac15908e7a9882d9ab7e090473d1d36cd75dcd34d68869825ec18c92cefa6ec20fc4@211.34.230.1:50501",
	"enode://f465600130c8d4d06fe41dbedb6d9d72aa213afb7d0e182232258eb108016f312f8786aaba9acd6aa3d6923308178b527e8a96139b0224e0cf3fe5227837210d@211.34.230.2:50501",
	"enode://328787decfb2e5848f3659a322f90db312d2f340c667ce4c2e3c5e66f4642dc249181385b51d7cf9dc3b8e0b81bd76ca00d80fccba3d8d10a834eda0856ededc@211.34.230.6:50501",
}

// AndusChainTestNode
var TestnetBootnodes = []string{
	"enode://585232819253c97da804baefe8b0b015cd797dc784cd8b48888ff2e69f0e39815bdcdf81ea04e7823b663df7b49e075d5558fc3a0dfeea021c5147bb6540cd4c@bootnode.testnet.anduschain.io:50501",
}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
var DiscoveryV5Bootnodes = []string{}

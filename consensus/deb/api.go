// Copyright 2018 The go-anduschain Authors
// Package clique implements the proof-of-deb consensus engine.

package deb

import (
	"github.com/anduschain/go-anduschain/consensus"
	"github.com/anduschain/go-anduschain/log"
)

// API is a user facing RPC API to allow controlling the signer and voting
// mechanisms of the proof-of-deb scheme.
type API struct {
	chain consensus.ChainReader
	deb   *Deb
}

func (api *API) GetJoinNonce() uint64 {
	current := api.chain.CurrentHeader()
	state, err := api.chain.StateAt(current.Hash())
	if err != nil {
		log.Error("GetJoinNonce", "error", err)
	}
	return state.GetJoinNonce(api.chain.CurrentHeader().Coinbase)
}

func (api *API) GetFairnodeAddress() string {
	return api.deb.config.FairPubKey
}

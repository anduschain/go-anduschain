// Copyright 2018 The go-anduschain Authors
// Package clique implements the proof-of-deb consensus engine.

package deb

import (
	"github.com/anduschain/go-anduschain/consensus"
)

// API is a user facing RPC API to allow controlling the signer and voting
// mechanisms of the proof-of-deb scheme.
type PrivateDebApi struct {
	chain consensus.ChainReader
	deb   *Deb
}

func NewPrivateDebApi(chain consensus.ChainReader, deb *Deb) *PrivateDebApi {
	return &PrivateDebApi{chain, deb}
}

func (api *PrivateDebApi) GetFairnodePubKey() string {
	return api.deb.config.FairPubKey
}

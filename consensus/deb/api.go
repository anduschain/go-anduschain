// Copyright 2018 The go-anduschain Authors
// Package clique implements the proof-of-deb consensus engine.

package deb

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
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

func (api *PrivateDebApi) FairnodePubKey() string {
	return api.deb.config.FairPubKey
}

func (api *PrivateDebApi) GenVrfKey() string {
	privatekey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return ""
	}
	x509Encoded, _ := x509.MarshalECPrivateKey(privatekey)
	priBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x509Encoded})
	return string(priBytes)
}

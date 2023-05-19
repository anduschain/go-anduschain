// Copyright 2018 The go-anduschain Authors
// Package clique implements the proof-of-deb consensus engine.

package deb

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"github.com/anduschain/go-anduschain/consensus"
	"github.com/anduschain/go-anduschain/crypto/custom"
	"math/big"
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

type vrfdata struct {
	Index string
	Proof string
	X     big.Int
	Y     big.Int
}

func (api *PrivateDebApi) VrfProof(m string) string {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err.Error()
	}

	// Get the public key from the private key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return err.Error()
	}

	// Get the private key in hex format
	//privateKeyHex := hexutil.Encode(crypto.FromECDSA(privateKey))

	index, proof := custom.Evaluate(privateKey, []byte(m))

	X := *publicKeyECDSA.X
	Y := *publicKeyECDSA.Y

	vrfData := &vrfdata{
		Index: hex.EncodeToString(index[:]),
		Proof: hex.EncodeToString(proof),
		X:     X,
		Y:     Y,
	}

	ret, err := json.Marshal(vrfData)
	if err != nil {
		return err.Error()
	}
	return string(ret)
}

func (api *PrivateDebApi) VrfVerify(m string, proofHex string, X big.Int, Y big.Int) string {
	pubKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     &X,
		Y:     &Y,
	}
	proof, err := hex.DecodeString(proofHex)
	if err != nil {
		return err.Error()
	}
	index, err := custom.ProofToHash(pubKey, []byte(m), proof)
	if err != nil {
		return err.Error()
	}

	return hex.EncodeToString(index[:])
}

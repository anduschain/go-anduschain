package vrf

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"filippo.io/edwards25519"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

var testVectors []testVector

type testVector struct {
	sk, pk, x, h, k, u, v, pi, alpha, beta []byte
}

func decodeHex(str string) []byte {
	b, err := hex.DecodeString(str)
	if err != nil {
		panic(err)
	}
	return b
}

func init() {
	testVectors = make([]testVector, 3)
	// B.3 ECVRF-EDWARDS25519-SHA512-TAI
	// B.3 Example 16
	testVectors[0] = testVector{
		decodeHex("9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60"),
		decodeHex("d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a"),
		decodeHex("307c83864f2833cb427a2ef1c00a013cfdff2768d980c0a3a520f006904de94f"),
		decodeHex("91bbed02a99461df1ad4c6564a5f5d829d0b90cfc7903e7a5797bd658abf3318"),
		decodeHex("7100f3d9eadb6dc4743b029736ff283f5be494128df128df2817106f345b8594b6d6da2d6fb0b4c0257eb337675d96eab49cf39e66cc2c9547c2bf8b2a6afae4"),
		decodeHex("aef27c725be964c6a9bf4c45ca8e35df258c1878b838f37d9975523f09034071"),
		decodeHex("5016572f71466c646c119443455d6cb9b952f07d060ec8286d678615d55f954f"),
		decodeHex("8657106690b5526245a92b003bb079ccd1a92130477671f6fc01ad16f26f723f5e8bd1839b414219e8626d393787a192241fc442e6569e96c462f62b8079b9ed83ff2ee21c90c7c398802fdeebea4001"),
		decodeHex(""),
		decodeHex("90cf1df3b703cce59e2a35b925d411164068269d7b2d29f3301c03dd757876ff66b71dda49d2de59d03450451af026798e8f81cd2e333de5cdf4f3e140fdd8ae"),
	}

	// B.3 Example 17
	testVectors[1] = testVector{
		decodeHex("4ccd089b28ff96da9db6c346ec114e0f5b8a319f35aba624da8cf6ed4fb8a6fb"),
		decodeHex("3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c"),
		decodeHex("68bd9ed75882d52815a97585caf4790a7f6c6b3b7f821c5e259a24b02e502e51"),
		decodeHex("5b659fc3d4e9263fd9a4ed1d022d75eaacc20df5e09f9ea937502396598dc551"),
		decodeHex("42589bbf0c485c3c91c1621bb4bfe04aed7be76ee48f9b00793b2342acb9c167cab856f9f9d4febc311330c20b0a8afd3743d05433e8be8d32522ecdc16cc5ce"),
		decodeHex("1dcb0a4821a2c48bf53548228b7f170962988f6d12f5439f31987ef41f034ab3"),
		decodeHex("fd03c0bf498c752161bae4719105a074630a2aa5f200ff7b3995f7bfb1513423"),
		decodeHex("f3141cd382dc42909d19ec5110469e4feae18300e94f304590abdced48aed593f7eaf3eb2f1a968cba3f6e23b386aeeaab7b1ea44a256e811892e13eeae7c9f6ea8992557453eac11c4d5476b1f35a08"),
		decodeHex("72"),
		decodeHex("eb4440665d3891d668e7e0fcaf587f1b4bd7fbfe99d0eb2211ccec90496310eb5e33821bc613efb94db5e5b54c70a848a0bef4553a41befc57663b56373a5031"),
	}

	// B.3 Example 18
	testVectors[2] = testVector{
		decodeHex("c5aa8df43f9f837bedb7442f31dcb7b166d38535076f094b85ce3a2e0b4458f7"),
		decodeHex("fc51cd8e6218a1a38da47ed00230f0580816ed13ba3303ac5deb911548908025"),
		decodeHex("909a8b755ed902849023a55b15c23d11ba4d7f4ec5c2f51b1325a181991ea95c"),
		decodeHex("bf4339376f5542811de615e3313d2b36f6f53c0acfebb482159711201192576a"),
		decodeHex("38b868c335ccda94a088428cbf3ec8bc7955bfaffe1f3bd2aa2c59fc31a0febc59d0e1af3715773ce11b3bbdd7aba8e3505d4b9de6f7e4a96e67e0d6bb6d6c3a"),
		decodeHex("2bae73e15a64042fcebf062abe7e432b2eca6744f3e8265bc38e009cd577ecd5"),
		decodeHex("88cba1cb0d4f9b649d9a86026b69de076724a93a65c349c988954f0961c5d506"),
		decodeHex("9bc0f79119cc5604bf02d23b4caede71393cedfbb191434dd016d30177ccbf80e29dc513c01c3a980e0e545bcd848222d08a6c3e3665ff5a4cab13a643bef812e284c6b2ee063a2cb4f456794723ad0a"),
		decodeHex("af82"),
		decodeHex("645427e5d00c62a23fb703732fa5d892940935942101e456ecca7bb217c61c452118fec1219202a0edcf038bb6373241578be7217ba85a2687f7a0310b2df19f"),
	}
}

func TestRFCVectors_GenerateKey(t *testing.T) {
	for _, vector := range testVectors {
		key, err := GenerateKey(bytes.NewReader(vector.sk))
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, vector.sk, key.sk)
		assert.Equal(t, vector.pk, key.pk)

		s, _ := edwards25519.NewScalar().SetBytesWithClamping(vector.x)
		assert.Equal(t, s, key.x)
	}
}

func TestRFCVectors_NewPrivateKey(t *testing.T) {
	for _, vector := range testVectors {
		key, err := NewPrivateKey(append(vector.sk, vector.pk...))
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, vector.sk, key.sk)
		assert.Equal(t, vector.pk, key.pk)

		s, _ := edwards25519.NewScalar().SetBytesWithClamping(vector.x)
		assert.Equal(t, s, key.x)
	}
}

func TestRFCVectors_hashToCurveTAI(t *testing.T) {
	for _, vector := range testVectors {
		h, err := hashToCurveTAI(vector.pk, vector.alpha)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, vector.h, h.Bytes())
	}
}

func TestRFCVectors_generateNonceHash(t *testing.T) {
	for _, vector := range testVectors {
		k := generateNonceHash(vector.sk, vector.h)
		assert.Equal(t, vector.k, k)
	}
}

func TestRFCVectors_Prove(t *testing.T) {
	for _, vector := range testVectors {
		key, err := NewPrivateKey(append(vector.sk, vector.pk...))
		if err != nil {
			t.Fatal(err)
		}

		beta, proof, err := key.Prove(vector.alpha)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, vector.pi, proof)
		assert.Equal(t, vector.beta, beta)
	}
}

func TestRFCVectors_Verify(t *testing.T) {
	for _, vector := range testVectors {
		key, _ := NewPublicKey(vector.pk)

		verified, beta, err := key.Verify(vector.alpha, vector.pi)
		if err != nil {
			t.Fatal(err)
		}

		assert.True(t, verified)
		assert.Equal(t, vector.beta, beta)
	}
}

func TestHonestComplete(t *testing.T) {
	sk, err := GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	pk, _ := sk.Public()
	alice := []byte("alice")
	aliceVRF, aliceProof, err := sk.Prove(alice)
	if err != nil {
		t.Fatal(err)
	}

	verified, aliceVRFFromVerification, err := pk.Verify(alice, aliceProof)
	if err != nil {
		t.Fatal(err)
	}
	if !verified {
		t.Error("Gen -> Compute -> Prove -> Verify -> FALSE")
	}

	assert.Equal(t, aliceVRF, aliceVRFFromVerification)
}

func TestConvertPrivateKeyToPublicKey(t *testing.T) {
	sk, err := GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	pk, _ := sk.Public()

	assert.Equal(t, sk.pk, pk.Bytes())

	edKey := ed25519.NewKeyFromSeed(sk.sk)
	assert.Equal(t, []byte(edKey[:scalarSize]), sk.sk)
	assert.Equal(t, []byte(edKey[scalarSize:]), sk.pk)
}

func TestFlipBitForgery(t *testing.T) {
	sk, err := GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	pk, _ := sk.Public()
	alice := []byte("alice")

	for i := 0; i < ProofSize; i++ {
		_, aliceProof, _ := sk.Prove(alice)
		for j := uint(0); j < 8; j++ {
			aliceProof[i] ^= 1 << j
			verified, _, _ := pk.Verify(alice, aliceProof)
			if verified {
				t.Fatalf("forged by using aliceVRF[%d]^%d:\n (sk=%X)", i, j, sk.sk)
			}
		}
	}
}

func BenchmarkProve(b *testing.B) {
	sk, err := GenerateKey(nil)
	if err != nil {
		b.Fatal(err)
	}
	alice := []byte("alice")
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, _, _ = sk.Prove(alice)
	}
}

func BenchmarkVerify(b *testing.B) {
	sk, err := GenerateKey(nil)
	if err != nil {
		b.Fatal(err)
	}
	alice := []byte("alice")
	_, proof, _ := sk.Prove(alice)
	pk, _ := sk.Public()

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_, _, _ = pk.Verify(alice, proof)
	}
}

func TestRatio(t *testing.T) {
	sk, err := GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	pk, _ := sk.Public()
	stringUuid := (uuid.New()).String()
	alice := []byte(stringUuid)
	aliceVRF, aliceProof, err := sk.Prove(alice)
	if err != nil {
		t.Fatal(err)
	}

	verified, aliceVRFFromVerification, err := pk.Verify(alice, aliceProof)
	if err != nil {
		t.Fatal(err)
	}
	if !verified {
		t.Error("Gen -> Compute -> Prove -> Verify -> FALSE")
	}

	assert.Equal(t, aliceVRF, aliceVRFFromVerification)
}

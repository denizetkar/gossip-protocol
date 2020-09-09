package securecomm

import (
	"github.com/monnand/dhkx"
	"golang.org/x/crypto/scrypt"
	"math/rand"
)

type KeyManagement struct {
	dh_group *dhkx.DHGroup
	dh_priv  *dhkx.DHKey
	dh_pub   []byte
}

func emptyKM() *KeyManagement {
	return &KeyManagement{}
}

func (km *KeyManagement) generateOwnDHKeys() {
	// Use default Group
	km.dh_group, _ = dhkx.GetGroup(0)
	// Use default random generator for private key generation
	km.dh_priv, _ = km.dh_group.GeneratePrivateKey(nil)
	km.dh_pub = km.dh_priv.Bytes()
}

// Compute received byte slice to a private DH Key
func (km *KeyManagement) computeExtDHKey(b []byte) []byte {
	// Recover public key
	pubKey := dhkx.NewPublicKey(b)

	// Compute the key
	k, _ := km.dh_group.ComputeKey(pubKey, km.dh_priv)
	return k.Bytes()
}

func proofOfWork(k int, pre_m []byte) ([]byte, error) {
	nonce := make([]byte, 64)
	rand.Read(nonce)
	m := append(pre_m, nonce...)
	// https://wizardforcel.gitbooks.io/practical-cryptography-for-developers-book/content/mac-and-key-derivation/scrypt.html
	// Memory required = 128 * N * r * p bytes
	hash, err := scrypt.Key(m, nonce, 16384, 8, 1, 128)
	if err != nil {
		return nil, err
	}
	for i := 0; i < k; i++ {
		if hash[i] != 0 {
			return proofOfWork(k, pre_m)
		}
	}
	return m, nil
}

package securecomm

import (
	"errors"
	"github.com/monnand/dhkx"
	"golang.org/x/crypto/scrypt"
	"math/rand"
)

const (
	SCRYPT_N           = 16384
	SCRYPT_r           = 8
	SCRYPT_p           = 1
	SCRYPT_HASH_LENGTH = 128
	SCRYPT_NONCE_SIZE  = 64
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

// Combine external public key and the local private key to compute the final key
func (km *KeyManagement) computeFinalKey(b []byte) ([]byte, error) {
	// Recover public key
	pubKey := dhkx.NewPublicKey(b)

	// Compute the key
	k, err := km.dh_group.ComputeKey(pubKey, km.dh_priv)
	return k.Bytes(), err
}
func (km *KeyManagement) returnNonceSize() int {
	return SCRYPT_NONCE_SIZE
}

// Tries to find right nonce to have k zeros at
func proofOfWork(k int, pre_m []byte) ([]byte, error) {
	// Length of the hash created by scrypt
	nonce := make([]byte, SCRYPT_NONCE_SIZE)
	rand.Read(nonce)
	m := append(pre_m, nonce...)
	// https://wizardforcel.gitbooks.io/practical-cryptography-for-developers-book/content/mac-and-key-derivation/scrypt.html
	// Memory required = 128 * N * r * p bytes
	hash, err := scrypt.Key(m, nonce, SCRYPT_N, SCRYPT_r, SCRYPT_p, SCRYPT_HASH_LENGTH)
	if err != nil {
		return nil, err
	}
	for i := SCRYPT_HASH_LENGTH - 1; i >= k; i-- {
		if hash[i] != 0 {
			return proofOfWork(k, pre_m)
		}
	}
	return m, nil
}

// Checks a message for validity
func checkProofOfWorkValidity(k int, m []byte) error {
	_, _, nonce := splitM(m, SCRYPT_NONCE_SIZE)
	hash, err := scrypt.Key(m, nonce, SCRYPT_N, SCRYPT_r, SCRYPT_p, SCRYPT_HASH_LENGTH)
	if err != nil {
		return err
	}
	for i := SCRYPT_HASH_LENGTH - 1; i >= k; i-- {
		if hash[i] != 0 {
			return errors.New("ProofOfWork is not valid")
		}
	}
	return nil
}

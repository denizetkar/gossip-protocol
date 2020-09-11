package securecomm

import (
	"errors"
	"math/big"
	"math/rand"

	"github.com/monnand/dhkx"
	"golang.org/x/crypto/scrypt"
)

// Parameters to be used in the scrypt command
const (
	ScryptN          = 16384
	ScryptR          = 8
	ScryptP          = 1
	ScryptHashlength = 128
	ScryptNonceSize  = 64
	ScryptRepetion   = 200
)

// KeyManagement manages own keys related to DH exchange
type KeyManagement struct {
	dhGroup *dhkx.DHGroup
	dhPriv  *dhkx.DHKey
	dhPub   []byte
}

func emptyKM() *KeyManagement {
	return &KeyManagement{}
}

func (km *KeyManagement) generateOwnDHKeys() {
	// Use default Group
	km.dhGroup, _ = dhkx.GetGroup(0)
	// Use default random generator for private key generation
	km.dhPriv, _ = km.dhGroup.GeneratePrivateKey(nil)
	km.dhPub = km.dhPriv.Bytes()
}

// Combine external public key and the local private key to compute the final key
func (km *KeyManagement) computeFinalKey(b []byte) ([]byte, error) {
	// Recover public key
	pubKey := dhkx.NewPublicKey(b)

	// Compute the key
	k, err := km.dhGroup.ComputeKey(pubKey, km.dhPriv)
	return k.Bytes(), err
}
func (km *KeyManagement) returnNonceSize() int {
	return ScryptNonceSize
}

// Tries to find right nonce to have k zeros at
func proofOfWork(k int, preM []byte) ([]byte, error) {
	// Length of the hash created by scrypt
	nonce := make([]byte, ScryptNonceSize)
	rand.Read(nonce)
	m := append(preM, nonce...)
	// https://wizardforcel.gitbooks.io/practical-cryptography-for-developers-book/content/mac-and-key-derivation/scrypt.html
	// Memory required = 128 * N * r * p bytes
	hash, err := scrypt.Key(m, nonce, ScryptN, ScryptR, ScryptP, ScryptHashlength)
	if err != nil {
		return nil, err
	}
	for i := ScryptHashlength - 1; i >= k; i-- {
		if hash[i] != 0 {
			return proofOfWork(k, preM)
		}
	}
	return nonce, nil
}

// checkProofOfWorkValidity expects k, and the handshake, where the nonce is seperated and the signatur is not included and checks the handshake for validity of the proof of work
func checkProofOfWorkValidity(k int, m []byte, nonce []byte) error {
	hash, err := scrypt.Key(m, nonce, ScryptN, ScryptR, ScryptP, ScryptHashlength)
	if err != nil {
		return err
	}
	for i := ScryptHashlength - 1; i >= ScryptHashlength-k; i-- {
		if hash[i] != 0 {
			return errors.New("ProofOfWork is not valid")
		}
	}
	return nil
}

// PoWThreshold returns the 'k' value for a given bit size and repetition.
func PoWThreshold(repetition, bits uint64) *big.Int {
	k := new(big.Int)
	// k = (2^bits - 1)/repetition
	k.Exp(new(big.Int).SetInt64(2), new(big.Int).SetUint64(bits), nil).
		Sub(k, new(big.Int).SetInt64(1)).
		Div(k, new(big.Int).SetUint64(repetition))
	return k
}

// Tries to find right nonce to have k zeros at
func proofOfWorkIT(k int, preM []byte) ([]byte, error) {
	// Threshold that must not be crossed to have a valid nonce
	threshold := PoWThreshold(ScryptRepetion, 256)
	nonce := make([]byte, ScryptNonceSize)
	rand.Read(nonce)
	m := append(preM, nonce...)
	// https://wizardforcel.gitbooks.io/practical-cryptography-for-developers-book/content/mac-and-key-derivation/scrypt.html
	// Memory required = 128 * N * r * p bytes
	for i := 0; i < 2*ScryptRepetion; i++ {
		hash, err := scrypt.Key(m, nonce, ScryptN, ScryptR, ScryptP, ScryptHashlength)
		if err != nil {
			return nil, err
		}
		hashVal := new(big.Int).SetBytes(hash)
		if hashVal.Cmp(threshold) <= 0 {
			return nonce, nil
		}
		rand.Read(nonce)
	}
	return nil, errors.New("securecomm: No suitable nonces found for PoW")
}

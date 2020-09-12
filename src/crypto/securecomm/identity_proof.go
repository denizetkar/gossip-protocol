package securecomm

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"math/big"
	"math/rand"
	"parser/identity"

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
	ScryptRepetition = 200
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

func (h *Handshake) hashVal() (*big.Int, error) {
	if h.Nonce == nil {
		return nil, errors.New("securecomm: Nonce should not be nil")
	}
	hash, err := scrypt.Key(h.concatIdentifiers(), h.Nonce, ScryptN, ScryptR, ScryptP, ScryptHashlength)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(hash), nil
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

// checkProofOfWorkValidity expects k, and the handshake, where the nonce is seperated and the signatur is not included and checks the handshake for validity of the proof of work
func checkProofOfWorkValidity(k int, h *Handshake) error {
	hashVal, err := h.hashVal()
	if err != nil {
		return err
	}
	threshold := PoWThreshold(ScryptRepetition, ScryptHashlength*8)

	if hashVal.Cmp(threshold) <= 0 {
		return nil
	}
	return errors.New("ProofOfWork is not valid")
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

// ProofOfWork tries to find right nonce to have k leading zeros
func ProofOfWork(k int, h *Handshake) error {
	// Threshold that must not be crossed to have a valid nonce
	threshold := PoWThreshold(ScryptRepetition, ScryptHashlength*8)
	h.Nonce = make([]byte, ScryptNonceSize)
	rand.Read(h.Nonce)
	// https://wizardforcel.gitbooks.io/practical-cryptography-for-developers-book/content/mac-and-key-derivation/scrypt.html
	// Memory required = 128 * N * r * p bytes
	for i := 0; i < 2*ScryptRepetition; i++ {
		hashVal, err := h.hashVal()
		if err != nil {
			return err
		}
		if hashVal.Cmp(threshold) <= 0 {
			return nil
		}
		rand.Read(h.Nonce)
	}
	h.Nonce = nil
	return errors.New("securecomm: No suitable nonces found for PoW")
}

// CheckIdentity ensures that the public key is trusted using the out-of-band shared identities
func CheckIdentity(pubKey *rsa.PublicKey, path string) error {
	pubKeyBytes := x509.MarshalPKCS1PublicKey(pubKey)
	shaKey := sha256.Sum256(pubKeyBytes)
	identities := identity.Parse(path)
	for _, v := range identities {
		if v == string(shaKey[:]) {
			return nil
		}
	}
	return errors.New("securecomm: Identity is not trusted")
}

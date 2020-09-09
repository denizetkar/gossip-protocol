package securecomm

import (
	"github.com/monnand/dhkx"
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

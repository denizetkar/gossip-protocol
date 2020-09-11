package securecomm

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"time"

	"golang.org/x/crypto/sha3"
)

func (c *SecureConn) serverHandshake() (err error) {
	if c.config == nil {
		return errors.New("Config is nil")
	}

	hs := &serverHandshakeState{
		c:  c,
		km: emptyKM(),
	}
	hs.km.generateOwnDHKeys()

	if err := hs.handshake(); err != nil {
		return err
	}

	return nil
}

type serverHandshakeState struct {
	c            *SecureConn
	km           *KeyManagement
	mClient      *Handshake
	mServer      *Handshake
	masterSecret []byte
}

func (hs *serverHandshakeState) handshake() error {
	if err := hs.doFullHandshake(); err != nil {
		return err
	}
	if err := hs.establishKey(); err != nil {
		return err
	}
	hs.c.masterKey = hs.masterSecret
	return nil
}

func (hs *serverHandshakeState) doFullHandshake() error {
	c := hs.c
	privKey := c.config.HostKey

	// Read and verify client handshake
	handshakeClient, err := c.read()
	if err != nil {
		return err
	}
	if handshakeClient.Data != nil || handshakeClient.Handshake.IsClient == true || handshakeClient.Handshake.isValid() {
		return messageError{}
	}
	hs.mClient = &handshakeClient.Handshake
	err = checkProofOfWorkValidity(hs.c.config.k, hs.mClient)
	if err != nil {
		return err
	}
	err = checkIdentity(&hs.mClient.RSAPub, hs.c.config.TrustedIdentitiesPath)
	if err != nil {
		return err
	}
	err = rsa.VerifyPKCS1v15(&hs.mClient.RSAPub, crypto.SHA3_256, hs.mClient.concatIdentifiersInclNonce(), hs.mClient.RSASig)
	if err != nil {
		return err
	}

	// Write handshake to Client
	handshake := Handshake{
		DHPub:    hs.km.dhPub,
		RSAPub:   hs.c.config.HostKey.PublicKey,
		Time:     time.Now().UTC(),
		Addr:     c.LocalAddr(),
		IsClient: false}

	nonce, err := proofOfWork(c.config.k, &handshake)
	if err != nil {
		return err
	}
	handshake.Nonce = nonce
	m := append(handshake.concatIdentifiers(), nonce...)
	shaM := sha3.Sum256(m)
	s, err := privKey.Sign(rand.Reader, shaM[:], crypto.SHA3_256)
	if err != nil {
		return err
	}
	handshake.RSASig = s
	hs.mServer = &handshake
	c.write(
		&Message{
			Data:      nil,
			Handshake: handshake})
	if err != nil {
		return err
	}
	return nil
}
func (hs *serverHandshakeState) establishKey() (err error) {
	hs.masterSecret, err = hs.km.computeFinalKey(hs.mClient.DHPub)
	return err
}

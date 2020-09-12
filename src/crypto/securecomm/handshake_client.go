package securecomm

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/sha3"
)

func (c *SecureConn) clientHandshake() (err error) {
	if c.config == nil {
		return errors.New("Config is nil")
	}

	hs := &clientHandshakeState{
		c:  c,
		km: emptyKM(),
	}
	hs.km.generateOwnDHKeys()

	if err := hs.handshake(); err != nil {
		return err
	}

	return nil
}

type clientHandshakeState struct {
	c            *SecureConn
	km           *KeyManagement
	mClient      *Handshake
	mServer      *Handshake
	masterSecret []byte
}

func (hs *clientHandshakeState) handshake() error {
	if err := hs.doFullHandshake(); err != nil {
		return err
	}
	if err := hs.establishKey(); err != nil {
		return err
	}
	hs.c.masterKey = hs.masterSecret
	atomic.StoreInt32(&hs.c.handShakeCompleted, 1)
	return nil
}

func (hs *clientHandshakeState) doFullHandshake() error {
	c := hs.c
	privKey := c.config.HostKey

	handshake := Handshake{
		DHPub:    hs.km.dhPub,
		RSAPub:   c.config.HostKey.PublicKey,
		Time:     time.Now().UTC(),
		Addr:     c.LocalAddr(),
		IsClient: true}

	err := ProofOfWork(c.config.k, &handshake)
	if err != nil {
		return err
	}

	// Sign message
	shaM := sha3.Sum256(handshake.concatIdentifiersInclNonce())
	s, err := rsa.SignPSS(rand.Reader, privKey, crypto.SHA3_256, shaM[:], nil)
	if err != nil {
		return err
	}
	handshake.RSASig = s

	// Write handshake to server
	c.write(
		&Message{
			Data:      nil,
			Handshake: handshake})
	hs.mClient = &handshake

	// Read and verify server handshake
	handshakeServer, err := c.read()
	if err != nil {
		return err
	}
	if handshakeServer.Data != nil || handshakeServer.Handshake.IsClient == true || handshakeServer.Handshake.isValid() {
		return messageError{}
	}
	hs.mServer = &handshakeServer.Handshake
	err = checkProofOfWorkValidity(hs.c.config.k, hs.mServer)
	if err != nil {
		return err
	}
	err = CheckIdentity(&hs.mServer.RSAPub, hs.c.config.TrustedIdentitiesPath)
	if err != nil {
		return err
	}
	err = rsa.VerifyPSS(&hs.mServer.RSAPub, crypto.SHA3_256, hs.mServer.concatIdentifiersInclNonce(), hs.mServer.RSASig, nil)
	if err != nil {
		return err
	}
	return nil
}
func (hs *clientHandshakeState) establishKey() (err error) {
	hs.masterSecret, err = hs.km.computeFinalKey(hs.mServer.DHPub)
	return err
}
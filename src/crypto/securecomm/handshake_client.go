package securecomm

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"errors"
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
	mClient      Handshake
	mServer      Handshake
	masterSecret []byte
}

func (hs *clientHandshakeState) handshake() error {
	if err := hs.doFullHandshake(); err != nil {
		return err
	}
	if err := hs.establishKey(); err != nil {
		return err
	}
	return nil
}

func (hs *clientHandshakeState) doFullHandshake() error {
	c := hs.c
	privKey := c.config.HostKey

	handshake := Handshake{
		DHPub:  hs.km.dhPub,
		RSAPub: hs.c.config.HostKey.PublicKey,
		Time:   time.Now().UTC(),
		Addr:   c.LocalAddr()}

	nonce, err := proofOfWork(c.config.k, handshake.concatIdentifiers())
	if err != nil {
		return err
	}
	hs.mClient.Nonce = nonce
	m := append(handshake.concatIdentifiers(), nonce...)
	shaM := sha3.Sum256(m)
	s, err := privKey.Sign(rand.Reader, shaM[:], crypto.SHA3_256)
	if err != nil {
		return err
	}
	hs.mClient.RSASig = s
	c.write(
		&Message{
			IsClient:  true,
			Data:      nil,
			Handshake: handshake})
	handshakeServer, err := c.read()
	if err != nil {
		return err
	}
	if handshakeServer.Data != nil || handshakeServer.IsClient == true || handshakeServer.Handshake.isEmpty() {
		return messageError{}
	}
	hs.mServer = handshakeServer.Handshake
	err = checkProofOfWorkValidity(hs.c.config.k, hs.mServer.concatIdentifiers(), hs.mServer.Nonce)
	if err != nil {
		return err
	}
	return nil
}
func (hs *clientHandshakeState) establishKey() (err error) {
	serverRSAPub := x509.MarshalPKCS1PublicKey(&hs.mServer.RSAPub)
	hs.masterSecret, err = hs.km.computeFinalKey(serverRSAPub)
	return err
}

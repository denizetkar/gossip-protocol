package securecomm

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
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
	mClient      Handshake
	mServer      Handshake
	masterSecret []byte
}

func (hs *serverHandshakeState) handshake() error {
	if err := hs.doFullHandshake(); err != nil {
		return err
	}
	if err := hs.establishKey(); err != nil {
		return err
	}
	return nil
}

func (hs *serverHandshakeState) doFullHandshake() error {
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
	hs.mServer.Nonce = nonce
	m := append(handshake.concatIdentifiers(), nonce...)
	shaM := sha3.Sum256(m)
	s, err := privKey.Sign(rand.Reader, shaM[:], crypto.SHA3_256)
	if err != nil {
		return err
	}
	hs.mServer.RSASig = s
	c.write(
		&Message{
			IsClient:  false,
			Data:      nil,
			Handshake: handshake})
	handshakeClient, err := c.read()
	if err != nil {
		return err
	}
	if handshakeClient.Data != nil || handshakeClient.IsClient == true || handshakeClient.Handshake.isEmpty() {
		return messageError{}
	}
	hs.mClient = handshakeClient.Handshake
	err = checkProofOfWorkValidity(hs.c.config.k, hs.mClient.concatIdentifiers(), hs.mClient.Nonce)
	if err != nil {
		return err
	}
	return nil
}
func (hs *serverHandshakeState) establishKey() (err error) {
	clientRSAPub := x509.MarshalPKCS1PublicKey(&hs.mClient.RSAPub)
	hs.masterSecret, err = hs.km.computeFinalKey(clientRSAPub)
	return err
}

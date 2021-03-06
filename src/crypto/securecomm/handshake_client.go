package securecomm

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"gossip/src/utils"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/sha3"
)

func (c *SecureConn) clientHandshake() (err error) {
	if c.config == nil {
		return fmt.Errorf("Config is nil")
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
		Addr:     c.RemoteAddr(),
		IsClient: true}

	err := ProofOfWork(c.config.k, &handshake)
	if err != nil {
		return err
	}

	// Sign message
	var opts rsa.PSSOptions
	opts.SaltLength = rsa.PSSSaltLengthAuto // for simple example
	shaM := sha3.Sum256(handshake.concatIdentifiersInclNonce())
	s, err := rsa.SignPSS(rand.Reader, privKey, crypto.SHA3_256, shaM[:], &opts)
	if err != nil {
		return err
	}
	handshake.RSASig = s

	// Write handshake to server
	err = c.write(
		&Message{
			Data:      make([]byte, 0),
			Handshake: handshake})
	if err != nil {
		return err
	}
	hs.mClient = &handshake

	// Read and verify server handshake
	handshakeServer, err := c.read()
	if err != nil {
		return err
	}
	if len(handshakeServer.Data) != 0 || handshakeServer.Handshake.IsClient == true || !handshakeServer.Handshake.isValid() {
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
	if !utils.TCPAddrCmp(c.conn.LocalAddr().String(), hs.mServer.Addr.String()) {
		return fmt.Errorf("securecomm: Handshake IP Address and Connection IP Address don't match")
	}
	shaM = sha3.Sum256(hs.mServer.concatIdentifiersInclNonce())
	err = rsa.VerifyPSS(&hs.mServer.RSAPub, crypto.SHA3_256, shaM[:], hs.mServer.RSASig, &opts)
	if err != nil {
		return err
	}
	return nil
}
func (hs *clientHandshakeState) establishKey() (err error) {
	hs.masterSecret, err = hs.km.computeFinalKey(hs.mServer.DHPub)
	return err
}

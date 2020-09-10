package securecomm

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"golang.org/x/crypto/sha3"
	"time"
)

func (c *SecureConn) clientHandshake() (err error) {
	if c.config == nil {
		c.config = defaultConfig()
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
	m_client     []byte
	m_server     []byte
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
	priv_key := c.config.HostKey

	// Seriallize Public Key
	rsa_pub := x509.MarshalPKCS1PublicKey(&priv_key.PublicKey)
	pre_m := append(hs.km.dh_pub, rsa_pub...)

	// Seriallize Time
	time_bytes := toByteArray(time.Now().UTC().Unix())
	pre_m = append(pre_m, time_bytes[:]...)

	// Seriallize IP adress
	addr_bytes, _ := hex.DecodeString(c.LocalAddr().String())
	pre_m = append(pre_m, addr_bytes...)

	m, err := proofOfWork(c.config.k, pre_m)
	if err != nil {
		return err
	}
	sha_m := sha3.Sum256(m)
	s, err := priv_key.Sign(rand.Reader, sha_m[:], crypto.SHA3_256)
	if err != nil {
		return err
	}
	hs.m_client = append(m, s...)
	c.write(
		&message{
			header_type: HANDSHAKE_MSG,
			isClient:    true,
			data:        hs.m_client})
	handshake_server, err := c.read()
	if err != nil {
		return err
	}
	if handshake_server.header_type != HANDSHAKE_MSG || handshake_server.isClient == true {
		return messageError{}
	}
	hs.m_server = handshake_server.data
	err = checkProofOfWorkValidity(hs.c.config.k, hs.m_server)
	if err != nil {
		return err
	}
	return nil
}
func (hs *clientHandshakeState) establishKey() (err error) {
	dhe_server, _, _ := splitM(hs.m_server, hs.km.returnNonceSize())
	hs.masterSecret, err = hs.km.computeFinalKey(dhe_server)
	return err
}

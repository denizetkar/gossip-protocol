package securecomm

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"golang.org/x/crypto/sha3"
	"time"
)

func (c *SecureConn) serverHandshake() (err error) {
	if c.config == nil {
		c.config = defaultConfig()
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
	m_client     []byte
	m_server     []byte
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
	hs.m_server = append(m, s...)

	handshake_client, err := c.read()
	if err != nil {
		return err
	}
	if handshake_client.header_type != HANDSHAKE_MSG || handshake_client.isClient == false {
		return messageError{}
	}
	hs.m_client = handshake_client.data
	err = checkProofOfWorkValidity(hs.c.config.k, hs.m_client)
	if err != nil {
		return err
	}

	c.write(
		&message{
			header_type: HANDSHAKE_MSG,
			isClient:    false,
			data:        hs.m_server})
	return nil
}
func (hs *serverHandshakeState) establishKey() (err error) {
	dhe_client, _, _ := splitM(hs.m_client, hs.km.returnNonceSize())
	hs.masterSecret, err = hs.km.computeFinalKey(dhe_client)
	return err
}

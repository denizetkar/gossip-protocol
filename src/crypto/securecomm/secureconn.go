package securecomm

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/gob"
	"encoding/hex"
	"net"
	"sync"
	"time"
)

// SecureConn is the secure communication connection.
type SecureConn struct {
	conn           *net.TCPConn
	config         *Config
	isClient       bool
	handshakeFn    func() error // (*SecureConn).clientHandshake or serverHandshake
	handshakeMutex sync.Mutex

	// handshakeStatus is 1 if the connection is currently transferring
	// application data (i.e. is not currently processing a handshake).
	// This field is only to be accessed with sync/atomic.
	handshakeStatus uint32
	input           *gob.Decoder
	output          *gob.Encoder
}

// Message that is seriallized and should be send or received
// Includes either Data or Handshake
type Message struct {
	IsClient  bool
	Data      []byte
	Handshake Handshake
}

// Handshake that can be included in a message
type Handshake struct {
	DHPub  []byte
	RSAPub rsa.PublicKey
	Time   time.Time
	Addr   net.Addr
	Nonce  []byte
	RSASig []byte
}

func (h *Handshake) isEmpty() bool {
	return h.DHPub == nil && h.RSAPub.Size() == 0 && h.Time.IsZero() && h.Addr.String() == "" && h.Nonce == nil && h.RSASig == nil
}

// concatIdentifiers returns a byte slice of every identity-realted field in the handshake (DHPub, RSAPub, Time, Addr)
func (h *Handshake) concatIdentifiers() (result []byte) {
	// Seriallize Public Key
	rsaPub := x509.MarshalPKCS1PublicKey(&h.RSAPub)
	result = append(h.DHPub, rsaPub...)

	// Seriallize Time
	timeBytes := toByteArray(h.Time.Unix())
	result = append(result, timeBytes[:]...)

	// Seriallize IP adress
	addrBytes, _ := hex.DecodeString(h.Addr.String())
	result = append(result, addrBytes...)
	return result
}

type messageError struct{}

func (messageError) Error() string { return "securecomm: Message format is incorrect" }
func (c *SecureConn) write(data *Message) error {
	if !(data.Data != nil || !data.Handshake.isEmpty()) {
		return messageError{}
	}
	err := c.output.Encode(data)
	return err
}
func (c *SecureConn) read() (data *Message, err error) {

	err = c.input.Decode(data)
	return data, err
}

// Handshake runs the client or server handshake
// protocol if it has not yet been run.
//
// Most uses of this package need not call Handshake explicitly: the
// first Read or Write will call it automatically.
//
// For control over canceling or setting a timeout on a handshake, use
// the Dialer's DialContext method.
func (c *SecureConn) Handshake() error {
	c.handshakeMutex.Lock()
	defer c.handshakeMutex.Unlock()

	handshakeErr := c.handshakeFn()

	return handshakeErr
}

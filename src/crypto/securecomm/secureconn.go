package securecomm

import (
	"encoding/gob"
	"net"
	"sync"
)

// SecureConn is the secure communication connection.
type SecureConn struct {
	conn           net.Conn
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

const (
	HANDSHAKE_MSG = "handshake_msg"
	DATA_MSG      = "data_msg"
)

// The message that is seriallized and should be send or received
type message struct {
	// Either HANDSHAKE_MSG or DATA_MSG
	header_type string
	isClient    bool
	data        []byte
}
type messageError struct{}

func (messageError) Error() string { return "securecomm: Message format is incorrect" }
func (c *SecureConn) write(data *message) error {
	if data.header_type != HANDSHAKE_MSG || data.header_type != DATA_MSG {
		return messageError{}
	}
	err := c.output.Encode(data)
	return err
}
func (c *SecureConn) read() (data *message, err error) {

	err = c.input.Decode(data)
	return data, err
}

// Extract components of handshake message
func splitM(m []byte, nonce_size int) (dhe []byte, rsa []byte, nonce []byte) {
	dhe = m[:len(m)-512-nonce_size-1]
	rsa = m[len(m)-512-nonce_size : len(m)-nonce_size-1]
	nonce = m[len(m)-nonce_size:]
	return dhe, rsa, nonce
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

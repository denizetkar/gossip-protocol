package securecomm

import (
	"bytes"
	"errors"
	"net"
	"sync"
	"sync/atomic"
)

// SecureConn is the secure communication connection.
type SecureConn struct {
	// TODO: fill here
	conn           net.Conn
	config         *Config
	isClient       bool
	handshakeFn    func() error // (*SecureConn).clientHandshake or serverHandshake
	handshakeMutex sync.Mutex
	handshakeErr   error // error resulting from handshake

	// handshakeStatus is 1 if the connection is currently transferring
	// application data (i.e. is not currently processing a handshake).
	// This field is only to be accessed with sync/atomic.
	handshakeStatus uint32

	// input/output
	in, out   halfConn
	rawInput  bytes.Buffer // raw input, starting with a record header
	input     bytes.Reader // application data waiting to be read, from rawInput.Next
	hand      bytes.Buffer // handshake data waiting to be read
	outBuf    []byte       // scratch buffer used by out.encrypt
	buffering bool         // whether records are buffered in sendBuf
	sendBuf   []byte       // a buffer of records waiting to be sent
}

var emptyConfig Config

func defaultConfig() *Config {
	return &emptyConfig
}

func (c *SecureConn) flush() (int, error) {
	if len(c.sendBuf) == 0 {
		return 0, nil
	}

	n, err := c.conn.Write(c.sendBuf)
	// c.bytesSent += int64(n)
	c.sendBuf = nil
	c.buffering = false
	return n, err
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

	if err := c.handshakeErr; err != nil {
		return err
	}
	if c.handshakeComplete() {
		return nil
	}

	c.in.Lock()
	defer c.in.Unlock()

	c.handshakeErr = c.handshakeFn()
	if c.handshakeErr != nil {
		// If an error occurred during the handshake try to flush the
		// alert that might be left in the buffer.
		c.flush()
	}

	if c.handshakeErr == nil && !c.handshakeComplete() {
		c.handshakeErr = errors.New("tls: internal error: handshake should have had a result")
	}

	return c.handshakeErr
}

func (c *SecureConn) handshakeComplete() bool {
	return atomic.LoadUint32(&c.handshakeStatus) == 1
}

// A halfConn represents one direction of the record layer
// connection, either sending or receiving.
type halfConn struct {
	sync.Mutex

	err            error    // first permanent error
	seq            [8]byte  // 64-bit sequence number
	additionalData [13]byte // to avoid allocs; interface method args escape
}

func (c *SecureConn) clientHandshake() (err error) {
	if c.config == nil {
		c.config = defaultConfig()
	}

	hs := &clientHandshakeState{
		c:           c,
		serverHello: serverHello,
		hello:       hello,
		km:          emptyKM(),
	}

	if err := hs.handshake(); err != nil {
		return err
	}

	return nil
}

type clientHandshakeState struct {
	c            *SecureConn
	serverHello  *serverHelloMsg
	hello        *clientHelloMsg
	km           *KeyManagement
	m            string
	masterSecret []byte
}

func (hs *clientHandshakeState) handshake() error {
	c := hs.c
	hs.km.generateOwnDHKeys()
	if err := hs.doFullHandshake(); err != nil {
		return err
	}
	if err := hs.establishKeys(); err != nil {
		return err
	}
	if _, err := c.flush(); err != nil {
		return err
	}

	return nil
}

func (hs *clientHandshakeState) doFullHandshake() error {
	c := hs.c

	// pre_m := append(hs.km.dh_pub, c.config.HostKey)
	return nil
}
func (hs *clientHandshakeState) establishKeys() error {
	c := hs.c

	return nil
}

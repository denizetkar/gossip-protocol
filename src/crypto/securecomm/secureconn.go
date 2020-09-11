package securecomm

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"encoding/gob"
	"encoding/hex"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// SecureConn is the secure communication connection.
type SecureConn struct {
	conn           *net.TCPConn
	config         *Config
	isClient       bool
	handshakeFn    func() error // (*SecureConn).clientHandshake or serverHandshake
	handshakeMutex sync.Mutex

	// handShakeCompleted is 1 if a handshake was established
	// This field is only to be accessed with sync/atomic.
	handShakeCompleted int32
	input              *gob.Decoder
	output             *gob.Encoder

	// Master Key which encrypts and decrypts communication between two peers
	masterKey []byte

	// Mutex for access to the communication input/outputs
	in  sync.Mutex
	out sync.Mutex
}

// Message that is serialized and should be send or received
// Includes either Data or Handshake
type Message struct {
	Data      []byte
	Handshake Handshake
}

// Handshake that can be included in a message
type Handshake struct {
	DHPub    []byte
	RSAPub   rsa.PublicKey
	Time     time.Time
	Addr     net.Addr
	Nonce    []byte
	RSASig   []byte
	IsClient bool
}

func (h *Handshake) isValid() bool {
	return len(h.DHPub) == 256 &&
		h.RSAPub.Size() == 512 &&
		!h.Time.IsZero() &&
		h.Addr != nil &&
		h.Addr.String() != "" &&
		len(h.Nonce) == ScryptNonceSize &&
		len(h.RSASig) == 512
}

// concatIdentifiers returns a byte slice of every identity-realted field in the handshake (DHPub, RSAPub, Time, Addr)
func (h *Handshake) concatIdentifiers() (result []byte) {
	// Serialize Public Key
	rsaPub := x509.MarshalPKCS1PublicKey(&h.RSAPub)
	result = append(h.DHPub, rsaPub...)

	// Serialize Time
	timeBytes := toByteArray(h.Time.Unix())
	result = append(result, timeBytes[:]...)

	// Serialize IP adress
	addrBytes, _ := hex.DecodeString(h.Addr.String())
	result = append(result, addrBytes...)
	return result
}

// concatIdentifiers returns a byte slice of every field in the handshake except RSASig(DHPub, RSAPub, Time, Addr, Nonce)
func (h *Handshake) concatIdentifiersInclNonce() (result []byte) {
	result = h.concatIdentifiers()
	result = append(result, h.Nonce...)
	return result
}

type messageError struct{}

func (messageError) Error() string { return "securecomm: Message format is incorrect" }

// Write a Message directly, should be used only internally
func (c *SecureConn) write(data *Message) error {
	c.out.Lock()
	defer c.out.Unlock()
	if !(data.Data != nil || !data.Handshake.isValid()) {
		return messageError{}
	}
	err := c.output.Encode(data)
	return err
}

// Read a Message directly, should be used only internally
func (c *SecureConn) read() (data *Message, err error) {
	c.in.Lock()
	defer c.in.Unlock()
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

	if atomic.LoadInt32(&c.handShakeCompleted) == 1 {
		return nil
	}
	handshakeErr := c.handshakeFn()

	return handshakeErr
}

func toByteArray(i int64) (arr [8]byte) {
	binary.BigEndian.PutUint64(arr[0:8], uint64(i))
	return
}

// Package securecomm package is the implementation of
// Gossip-6 layer4 secure communication protocol.
package securecomm

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"strings"
	"time"
)

// Config is the struct for configuration parameters
// of either a SecureListener or a SecureConn.
type Config struct {
	// TrustedIdentitiesPath is the path to the folder containing the
	// empty files whose names are hex encoded 'identity' of the trusted peers.
	// This folder HAS TO contain the identity of the 'bootstrapper' !!!
	TrustedIdentitiesPath string
	// HostKey is the variable containing 4096-bit RSA key.
	HostKey *rsa.PrivateKey
	// TODO: fill here
}

// SecureListener is the secure communication listener.
type SecureListener struct {
	// TODO: fill here
	ln     net.TCPListener
	config *Config
}

// SecureConn is the secure communication connection.
type SecureConn struct {
	// TODO: fill here
	conn net.TCPConn
}

// NewConfig is the constructor method for Config struct.
func NewConfig(trustedIdentitiesPath, hostKeyPath, pubKeyPath string) (*Config, error) {
	// Read and load the RSA private key.
	priv, err := ioutil.ReadFile(hostKeyPath)
	if err != nil {
		return nil, err
	}
	privPem, _ := pem.Decode(priv)
	if privPem == nil || !strings.Contains(privPem.Type, "PRIVATE KEY") {
		return nil, fmt.Errorf("RSA key is not a valid '.pem' type private key")
	}
	privPemBytes := privPem.Bytes
	var parsedKey interface{}
	if parsedKey, err = x509.ParsePKCS1PrivateKey(privPemBytes); err != nil {
		if parsedKey, err = x509.ParsePKCS8PrivateKey(privPemBytes); err != nil {
			return nil, fmt.Errorf("Unable to parse RSA private key")
		}
	}
	privateKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("Unable to parse RSA private key")
	}

	// Read and load the RSA public key.
	pub, err := ioutil.ReadFile(pubKeyPath)
	if err != nil {
		return nil, fmt.Errorf("No RSA public key found, generating temp one")
	}
	pubPem, _ := pem.Decode(pub)
	if pubPem == nil || !strings.Contains(pubPem.Type, "PUBLIC KEY") {
		return nil, fmt.Errorf("RSA key is not a valid '.pem' type public key")
	}

	if parsedKey, err = x509.ParsePKIXPublicKey(pubPem.Bytes); err != nil {
		return nil, fmt.Errorf("Unable to parse RSA public key")
	}
	var pubKey *rsa.PublicKey
	if pubKey, ok = parsedKey.(*rsa.PublicKey); !ok {
		return nil, fmt.Errorf("Unable to parse RSA public key")
	}

	privateKey.PublicKey = *pubKey
	return &Config{TrustedIdentitiesPath: trustedIdentitiesPath, HostKey: privateKey}, nil
}

// NewListener is the constructor function of SecureListener.
func NewListener() *SecureListener {
	// TODO: fill here
	return nil
}

// Listen is the function for creating a secure
// communication listener.
func Listen(network, laddr string, config *Config) (*SecureListener, error) {
	// TODO: fill here
	return nil, nil
}

// Dial is the function for creating a secure
// communication connection.
func Dial(network, addr string, config *Config) (*SecureConn, error) {
	// TODO: fill here
	return nil, nil
}

// DialWithDialer is the function for creating a secure
// communication connection with the given dialer.
func DialWithDialer(dialer *net.Dialer, network, addr string, config *Config) (*SecureConn, error) {
	// TODO: fill here
	return nil, nil
}

// Accept waits for and returns the next incoming secure connection.
// The returned connection is of type *SecureConn.
func (l *SecureListener) Accept() (net.Conn, error) {
	// TODO: fill here
	return nil, nil
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (l *SecureListener) Close() error {
	// TODO: fill here
	return nil
}

// Addr returns the listener's network address, a *TCPAddr.
// The Addr returned is shared by all invocations of Addr, so
// do not modify it.
func (l *SecureListener) Addr() net.Addr {
	return l.ln.Addr()
}

// Read can be made to time out and return a net.Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
func (sc *SecureConn) Read(b []byte) (int, error) {
	// TODO: fill here
	return 0, nil
}

// Write writes data to the connection.
func (sc *SecureConn) Write(b []byte) (int, error) {
	// TODO: fill here
	return 0, nil
}

// Close closes the secure connection properly.
func (sc *SecureConn) Close() error {
	// TODO: fill here
	return nil
}

// LocalAddr returns the local network address.
func (sc *SecureConn) LocalAddr() net.Addr {
	return sc.conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (sc *SecureConn) RemoteAddr() net.Addr {
	return sc.conn.RemoteAddr()
}

// SetDeadline sets the read and write deadlines associated with the connection.
// A zero value for t means Read and Write will not time out.
// After a Write has timed out, the TLS state is corrupt and all future writes will return the same error.
func (sc *SecureConn) SetDeadline(t time.Time) error {
	return sc.conn.SetDeadline(t)
}

// SetReadDeadline sets the read deadline on the underlying connection.
// A zero value for t means Read will not time out.
func (sc *SecureConn) SetReadDeadline(t time.Time) error {
	return sc.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline on the underlying connection.
// A zero value for t means Write will not time out.
// After a Write has timed out, the TLS state is corrupt and all future writes will return the same error.
func (sc *SecureConn) SetWriteDeadline(t time.Time) error {
	return sc.conn.SetWriteDeadline(t)
}

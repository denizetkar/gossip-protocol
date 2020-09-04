// Package securecomm package is the implementation of
// Gossip-6 layer4 secure communication protocol.
package securecomm

import (
	"crypto/rsa"
	"net"
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

// SecureServer returns a new secure server side connection
// using TCPConn as the underlying transport.
func SecureServer(conn *net.TCPConn) *SecureConn {
	c := &SecureConn{
		conn: *conn,
	}
	return c
}

// NewListener is the constructor function of SecureListener.
func NewListener(inner *net.TCPListener, config *Config) *SecureListener {
	return &SecureListener{
		ln:     *inner,
		config: config,
	}
}

// Listen is the function for creating a secure
// communication listener.
func Listen(network string, laddr *net.TCPAddr, config *Config) (*SecureListener, error) {
	//TODO: Check for prerequisitions
	// Construct a TCPListener
	ln, err := net.ListenTCP(network, laddr)
	if err != nil {
		return nil, err
	}
	return NewListener(ln, config), nil
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
	c, err := l.ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	return SecureServer(c), nil
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (l *SecureListener) Close() error {
	//TODO: Shutdown gracefully
	err := l.Close()
	return err
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
	err := sc.Close()
	// TODO: check if done properly
	return err
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

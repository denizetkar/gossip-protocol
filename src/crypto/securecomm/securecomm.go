// Package securecomm package is the implementation of
// Gossip-6 layer4 secure communication protocol.
package securecomm

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/gob"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
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
	// Number of zeros necessary in Proof Of Work hash
	k int
	// CacheSize is needed to calculate maximum message size
	cacheSize int64
}

// SecureListener is the secure communication listener.
type SecureListener struct {
	// TODO: fill here
	ln     net.TCPListener
	config *Config
}

// Client returns a new secure client side connection
// using conn as the underlying transport.
// The config cannot be nil: users must set either ServerName or
// InsecureSkipVerify in the config.
func Client(conn *net.TCPConn, config *Config) *SecureConn {
	c := &SecureConn{
		conn:     conn,
		config:   config,
		isClient: true,
		input:    gob.NewDecoder(io.LimitReader(conn, 65580*config.cacheSize)),
		output:   gob.NewEncoder(io.Writer(conn)),
	}
	c.handshakeFn = c.clientHandshake
	return c
}

// SecureServer returns a new secure server side connection
// using TCPConn as the underlying transport.
func SecureServer(conn *net.TCPConn, config *Config) *SecureConn {
	c := &SecureConn{
		conn:     conn,
		isClient: false,
		input:    gob.NewDecoder(io.LimitReader(conn, 65580*config.cacheSize)),
		output:   gob.NewEncoder(io.Writer(conn)),
	}
	c.handshakeFn = c.serverHandshake
	return c
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

	// Hard code k for proof of work
	k := 12
	return &Config{TrustedIdentitiesPath: trustedIdentitiesPath, HostKey: privateKey, k: k}, nil
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
func Listen(network string, laddr *net.TCPAddr, config *Config) (net.Listener, error) {
	//TODO: Check for prerequisitions
	// Constructs a TCPListener
	ln, err := net.ListenTCP(network, laddr)
	if err != nil {
		return nil, err
	}
	return NewListener(ln, config), nil
}

type timeoutError struct{}
type configError struct{}

func (configError) Error() string { return "no config specified" }

func (timeoutError) Error() string   { return "tls: DialWithDialer timed out" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

// Dial is the function for creating a secure
// communication connection.
func Dial(network, addr string, config *Config) (*SecureConn, error) {
	return DialWithDialer(new(net.Dialer), network, addr, config)
}

// DialWithDialer is the function for creating a secure
// communication connection with the given dialer.
func DialWithDialer(dialer *net.Dialer, network, addr string, config *Config) (*SecureConn, error) {
	return dial(context.Background(), dialer, network, addr, config)
}

func dial(ctx context.Context, netDialer *net.Dialer, network, addr string, config *Config) (*SecureConn, error) {
	// We want the Timeout and Deadline values from dialer to cover the
	// whole process: TCP connection and TLS handshake. This means that we
	// also need to start our own timers now.
	timeout := netDialer.Timeout
	if !netDialer.Deadline.IsZero() {
		deadlineTimeout := time.Until(netDialer.Deadline)
		if timeout == 0 || deadlineTimeout < timeout {
			timeout = deadlineTimeout
		}
	}

	// hsErrCh is non-nil if we might not wait for Handshake to complete.
	var hsErrCh chan error
	if timeout != 0 || ctx.Done() != nil {
		hsErrCh = make(chan error, 2)
	}
	if timeout != 0 {
		timer := time.AfterFunc(timeout, func() {
			hsErrCh <- timeoutError{}
		})
		defer timer.Stop()
	}
	rawConn, err := netDialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	colonPos := strings.LastIndex(addr, ":")
	if colonPos == -1 {
		colonPos = len(addr)
	}
	if config == nil {
		return nil, errors.New("Config is nil")
	}

	//TODO: check if correctly casted
	conn := Client(rawConn.(*net.TCPConn), config)
	if hsErrCh == nil {
		err = conn.Handshake()
	} else {
		go func() {
			hsErrCh <- conn.Handshake()
		}()

		select {
		case <-ctx.Done():
			err = ctx.Err()
		case err = <-hsErrCh:
			if err != nil {
				// If the error was due to the context
				// closing, prefer the context's error, rather
				// than some random network teardown error.
				if e := ctx.Err(); e != nil {
					err = e
				}
			}
		}
	}
	if err != nil {
		rawConn.Close()
		return nil, err
	}

	return conn, nil
}

// Accept waits for and returns the next incoming secure connection.
// The returned connection is of type *SecureConn.
func (l *SecureListener) Accept() (net.Conn, error) {
	c, err := l.ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	return SecureServer(c, l.config), nil
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
	return sc.Read(b)
}

// Write writes data to the connection.
func (sc *SecureConn) Write(b []byte) (int, error) {
	// TODO: fill here
	return sc.Write(b)
}

// Close closes the secure connection properly.
func (sc *SecureConn) Close() error {
	err := sc.Close()
	// TODO: fill here
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

// Clone returns a shallow clone of c. It is safe to clone a Config that is
// being used concurrently by a TLS client or server.
func (c *Config) Clone() *Config {
	return &Config{
		TrustedIdentitiesPath: c.TrustedIdentitiesPath,
		HostKey:               c.HostKey,
	}
}

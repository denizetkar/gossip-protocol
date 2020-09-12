package core

import (
	"gossip/src/crypto/securecomm"
	"net"
	"sync"
)

// Peer is just a placeholder for the TCP\IP address
// of a peer.
type Peer struct {
	Addr string
}

// ValidateAddr tries to validate the address of a peer.
func (peer *Peer) ValidateAddr() error {
	_, err := net.ResolveTCPAddr("tcp", peer.Addr)
	return err
}

// PeerReaderState is a const type for describing the execution state of an
// P2PEndpoint reader goroutine.
type PeerReaderState uint8

const (
	// PeerReaderSTOPPED means the reader goroutine of the connection is stopped.
	PeerReaderSTOPPED PeerReaderState = iota
	// PeerReaderRUNNING means the reader goroutine of the connection is running.
	PeerReaderRUNNING
)

// PeerWriterState is a const type for describing the execution state of an
// P2PEndpoint writer goroutine.
type PeerWriterState uint8

const (
	// PeerWriterSTOPPED means the writer goroutine of the connection is stopped.
	PeerWriterSTOPPED PeerWriterState = iota
	// PeerWriterRUNNING means the writer goroutine of the connection is running.
	PeerWriterRUNNING
)

// PeerState is a struct type that describes the state of a P2P connection.
type PeerState struct {
	readerState PeerReaderState
	writerState PeerWriterState
}

// PeerInfoCentral holds the endpoint variable to access the goroutines
// responsible for communicating with the corresponding peer. There
// is also a usageCounter for counting the number of times this peer
// connection is used by the Gossiper goroutine who will use these
// connections to actually do the gossiping. This struct is meant to be
// used as a value in a map[Peer]*PeerInfoCentral by the Central controller.
// Finally there are state variables for storing the state of a peer.
type PeerInfoCentral struct {
	endpoint     *P2PEndpoint
	usageCounter int
	state        PeerState
	hasCrashed   bool
}

// P2PEndpoint holds a secure connection for communicating with the
// corresponding peer. There are also 2 queues for the communication
// between the Central controller and the endpoint goroutines.
type P2PEndpoint struct {
	// peer is required here because it will later be used by both
	// the writer and the reader goroutines inside every InternalMessage
	// they send to the Central controller.
	peer Peer
	// conn is the secure communication socket.
	conn *securecomm.SecureConn
	// MsgInQueue is the incoming message queue for
	// this P2PEndpoint goroutine.
	MsgInQueue chan InternalMessage
	// MsgOutQueue is the outgoing message queue from
	// this P2PEndpoint goroutine to the Central controller.
	MsgOutQueue chan InternalMessage
	// sigCh is used for signaling the reader goroutine to close gracefully.
	sigCh chan struct{}
	// isOutgoing indicates whether this endpoint was initiated
	// by this node or the remote node. This endpoint is
	// initiated by this node IF AND ONLY IF the remote node
	// is in the 'viewList' OR is in the 'awaitingRemovalViewList'.
	isOutgoing bool
	// A synchronozation variable to execute the Close method only once.
	closeOnce sync.Once
}

// P2PListener is the goroutine that will listen for incoming P2P connection
// requests and it will open a P2PEndpoint for each connection. Then by
// using 'MsgOutQueue', it will inform the Central controller.
type P2PListener struct {
	ln *securecomm.SecureListener
	// MsgOutQueue is the outgoing message queue from
	// this P2PListener goroutine to the Central controller.
	MsgOutQueue chan InternalMessage
	// sigCh is used for signaling the p2p listener goroutine
	// to close gracefully.
	sigCh chan struct{}
}

// NewP2PListener is the constructor function of P2PListener struct.
func NewP2PListener(
	p2pAddr string, outQ chan InternalMessage, config *securecomm.Config,
) (*P2PListener, error) {
	lAddr, err := net.ResolveTCPAddr("tcp", p2pAddr)
	if err != nil {
		return nil, err
	}
	ln, err := securecomm.Listen("tcp", lAddr, config)
	if err != nil {
		return nil, err
	}

	return &P2PListener{
		ln:          ln.(*securecomm.SecureListener),
		MsgOutQueue: outQ,
		sigCh:       make(chan struct{}),
	}, nil
}

func (p2pListener *P2PListener) listenerRoutine() {
	// TODO: fill here

	// TODO: notify the Central controller before closing/returning!
}

// RunListenerGoroutine runs the goroutine that will listen
// for incoming p2p connections and serve them then inform
// the Central controller about it.
func (p2pListener *P2PListener) RunListenerGoroutine() {
	go p2pListener.listenerRoutine()
}

// Close method initiates a graceful closing operation without blocking.
func (p2pListener *P2PListener) Close() error {
	// Closing the 'sigCh' channel signals the listener to close itself.
	close(p2pListener.sigCh)
	p2pListener.ln.Close()
	return nil
}

// NewP2PEndpoint is the constructor function of P2PEndpoint struct.
func NewP2PEndpoint(
	p2pAddr string, config *securecomm.Config, inQ, outQ chan InternalMessage, isOutgoing bool,
) (*P2PEndpoint, error) {
	conn, err := securecomm.DialWithDialer(&net.Dialer{Timeout: connectionTimeout}, "tcp", p2pAddr, config)

	return &P2PEndpoint{
		peer: Peer{Addr: p2pAddr}, conn: conn,
		MsgInQueue: inQ, MsgOutQueue: outQ,
		sigCh: make(chan struct{}), isOutgoing: isOutgoing,
		closeOnce: sync.Once{},
	}, err
}

func (p2pEndpoint *P2PEndpoint) readerRoutine() {
	// TODO: fill here

	// TODO: notify the Central controller before closing/returning!
}

// RunReaderGoroutine runs the goroutine that will read from
// the p2p connection, process the segments and route the
// corresponding InternalMessage to the Central controller.
func (p2pEndpoint *P2PEndpoint) RunReaderGoroutine() {
	go p2pEndpoint.readerRoutine()
}

func (p2pEndpoint *P2PEndpoint) writerRoutine() {
	// TODO: fill here

	// TODO: notify the Central controller before closing/returning!
}

// RunWriterGoroutine runs the goroutine that will receive an
// InternalMessage from the Central controller, create the segment
// and write to the p2p connection in order to send the message.
func (p2pEndpoint *P2PEndpoint) RunWriterGoroutine() {
	go p2pEndpoint.writerRoutine()
}

// Close method initiates a graceful closing operation without blocking.
func (p2pEndpoint *P2PEndpoint) Close() error {
	p2pEndpoint.closeOnce.Do(func() {
		// TODO: send an InternalMessage to the writer for closing it!

		// Closing the 'sigCh' channel signals the reader to close itself.
		close(p2pEndpoint.sigCh)
	})
	return nil
}

// HaveBothStopped return true iff both the reader and
// the writer have a STOPPED state.
func (s *PeerState) HaveBothStopped() bool {
	return (s.readerState == PeerReaderSTOPPED && s.writerState == PeerWriterSTOPPED)
}

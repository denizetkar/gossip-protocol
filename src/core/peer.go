package core

import (
	"encoding/gob"
	"fmt"
	"gossip/src/crypto/securecomm"
	"gossip/src/datastruct/set"
	"io"
	"log"
	"net"
	"sync"
	"time"
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
	sigCh chan interface{}
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
	sigCh chan interface{}
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
		sigCh:       make(chan interface{}),
	}, nil
}

func (p2pListener *P2PListener) listenerRoutine() {
	defer p2pListener.recover()
	for done := false; !done; {
		conn, err := p2pListener.ln.Accept()
		if err != nil {
			switch {
			case <-p2pListener.sigCh:
				done = true
				continue
			default:
				break
			}
		}

		// Send the P2PEndpoint to the controller
		peer := Peer{
			Addr: conn.RemoteAddr().String(),
		}
		endp := &P2PEndpoint{
			isOutgoing:  false,
			conn:        conn.(*securecomm.SecureConn),
			sigCh:       make(chan interface{}),
			MsgInQueue:  make(chan InternalMessage, inQueueSize),
			MsgOutQueue: p2pListener.MsgOutQueue,
			peer:        peer,
			closeOnce:   sync.Once{},
		}
		p2pListener.MsgOutQueue <- InternalMessage{Type: IncomingP2PCreatedMSG, Payload: endp}
	}
	p2pListener.MsgOutQueue <- InternalMessage{Type: P2PListenerClosedMSG, Payload: void{}}
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
		sigCh: make(chan interface{}), isOutgoing: isOutgoing,
		closeOnce: sync.Once{},
	}, err
}

func (p2pEndpoint *P2PEndpoint) readerRoutine() {
	defer p2pEndpoint.recover(true)

	reader := io.Reader(p2pEndpoint.conn)
	gobDecoder := gob.NewDecoder(reader)

	for done := false; !done; {
		p2pEndpoint.conn.SetDeadline(time.Now().Add(closureCheckTimeout))

		var message InternalMessage
		err := gobDecoder.Decode(&message)
		if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
			switch {
			case <-p2pEndpoint.sigCh:
				done = true
			default:
				break
			}
			continue
		} else if err != nil {
			log.Println("P2PEndpoint: Error in readerRoutine():" + err.Error())
			continue
		}
		// Using IncomingP2PMSG message for all messages to be able to use a single handler.
		// The Payload is the whole message, including the right message type.
		switch message.Type {
		case MembershipPushRequestMSG:
			p2pEndpoint.MsgOutQueue <- InternalMessage{Type: IncomingP2PMSG, Payload: InternalMessage{Type: MembershipIncomingPushRequestMSG, Payload: message.Payload}}
		case MembershipPullRequestMSG:
			payload := &MembershipIncomingPullRequestMSGPayload{From: p2pEndpoint.peer}
			p2pEndpoint.MsgOutQueue <- InternalMessage{Type: IncomingP2PMSG, Payload: InternalMessage{Type: MembershipIncomingPullRequestMSG, Payload: payload}}
		case MembershipPullReplyMSG:
			m := message.Payload.(MembershipPullReplyMSGPayload)
			payload := &MembershipIncomingPullReplyMSGPayload{From: p2pEndpoint.peer, ViewList: m.ViewList}
			p2pEndpoint.MsgOutQueue <- InternalMessage{Type: IncomingP2PMSG, Payload: InternalMessage{Type: MembershipIncomingPullReplyMSG, Payload: payload}}
		case GossipPushMSG:
			m := message.Payload.(GossipPushMSGPayload)
			payload := &GossipItemExtended{Item: m.Item, State: m.State, Counter: m.Counter}
			p2pEndpoint.MsgOutQueue <- InternalMessage{Type: IncomingP2PMSG, Payload: InternalMessage{Type: GossipIncomingPushMSG, Payload: payload}}
		case GossipPullRequestMSG:
			payload := &GossipIncomingPullRequestMSGPayload{From: p2pEndpoint.peer}
			p2pEndpoint.MsgOutQueue <- InternalMessage{Type: IncomingP2PMSG, Payload: InternalMessage{Type: GossipIncomingPullRequestMSG, Payload: payload}}
		case GossipPullReplyMSG:
			m := message.Payload.(GossipPullReplyMSGPayload)
			payload := &GossipIncomingPullReplyMSGPayload{From: p2pEndpoint.peer, ItemList: m.ItemList}
			p2pEndpoint.MsgOutQueue <- InternalMessage{Type: IncomingP2PMSG, Payload: InternalMessage{Type: GossipIncomingPullReplyMSG, Payload: payload}}
		default:
			log.Println("P2PEndpoint: Error in readerRoutine(): invalid internal message type used")
			break
		}
		// End loop gracefully
		switch {
		case <-p2pEndpoint.sigCh:
			done = true
		default:
			break
		}
	}
	payload := &P2PEndpointClosedMSGPayload{endp: p2pEndpoint, isReader: true}
	p2pEndpoint.MsgOutQueue <- InternalMessage{Type: P2PEndpointClosedMSG, Payload: payload}
}

// RunReaderGoroutine runs the goroutine that will read from
// the p2p connection, process the segments and route the
// corresponding InternalMessage to the Central controller.
func (p2pEndpoint *P2PEndpoint) RunReaderGoroutine() {
	go p2pEndpoint.readerRoutine()
}

func (p2pEndpoint *P2PEndpoint) writerRoutine() {
	defer p2pEndpoint.recover(false)

	writer := io.Writer(p2pEndpoint.conn)
	gobEncoder := gob.NewEncoder(writer)
	allowedMSGs := set.New().Add(MembershipPushRequestMSG).
		Add(MembershipPullRequestMSG).Add(MembershipPullReplyMSG).
		Add(GossipPushMSG).Add(GossipPullRequestMSG).Add(GossipPullReplyMSG)

	for done := false; !done; {
		select {
		case im := <-p2pEndpoint.MsgInQueue:
			if im.Type == P2PEndpointCloseMSG {
				done = true
				continue
			} else if allowedMSGs.IsMember(im.Type) {
				err := gobEncoder.Encode(im)
				if err != nil {
					log.Println("P2PEndpoint: Error in writerRoutine():" + err.Error())
					continue
				}
			} else if im.Type == P2PEndpointCloseMSG {
				done = true
				continue
			} else {
				log.Println("P2PEndpoint: Error in writerRoutine(): invalid internal message type used")
				continue
			}
		}
	}
	payload := &P2PEndpointClosedMSGPayload{endp: p2pEndpoint, isReader: false}
	p2pEndpoint.MsgOutQueue <- InternalMessage{Type: P2PEndpointClosedMSG, Payload: payload}
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
		// Send an InternalMessage to the writer for closing it!
		p2pEndpoint.MsgInQueue <- InternalMessage{Type: P2PEndpointCloseMSG, Payload: void{}}
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

// recover method tries to catch a panic in listenerRoutine if it exists, then
// informs the Central controller about the crash.
func (p2pListener *P2PListener) recover() {
	var err error
	if r := recover(); r != nil {
		// find out exactly what the error was and set err
		switch x := r.(type) {
		case string:
			err = fmt.Errorf(x)
		case error:
			err = x
		default:
			err = fmt.Errorf("Unknown panic in P2PListener")
		}

		// send P2PListenerCrashedMSG to the Central controller!
		p2pListener.MsgOutQueue <- InternalMessage{Type: P2PListenerCrashedMSG, Payload: err}
	}
}

// recover method tries to catch a panic in an P2PEndpoint if it exists, then
// informs the Central controller about the crash.
func (p2pEndpoint *P2PEndpoint) recover(isReader bool) {
	var err error
	if r := recover(); r != nil {
		// find out exactly what the error was and set err
		switch x := r.(type) {
		case string:
			err = fmt.Errorf(x)
		case error:
			err = x
		default:
			err = fmt.Errorf("Unknown panic in P2PEndpoint")
		}

		// send P2PListenerCrashedMSG to the Central controller!
		payload := &P2PEndpointCrashedMSGPayload{endp: p2pEndpoint, err: err, isReader: isReader}
		p2pEndpoint.MsgOutQueue <- InternalMessage{Type: P2PEndpointCrashedMSG, Payload: payload}
	}
}

package main

import (
	"datastruct/set"
	"net"
	"sync"
)

// APIClient is just a placeholder for the TCP\IP address
// of an API client.
type APIClient struct {
	addr string
}

// APIClientInfoGossiperNotifyDataTypesType is the type of variable
// stored in APIClientInfoGossiper::notifyDataTypes.
type APIClientInfoGossiperNotifyDataTypesType GossipItemDataType

// APIClientInfoGossiper holds the gossip item data types for which the
// API client wants to be notified by the Gossiper controller.
// Therefore, this struct is meant to be used as a value in a
// map[APIClient]*APIClientInfoGossiper by the Gossiper controller.
type APIClientInfoGossiper struct {
	notifyDataTypes set.Set
	// validationMap is a map from message ID's of client notifications to gossip item.
	validationMap   map[uint16]*GossipItem
	nextAvailableID uint16
}

// APIClientReaderState is a const type for describing the execution state of an
// APIEndpoint reader goroutine.
type APIClientReaderState uint8

const (
	// APIClientReaderSTOPPED means the reader goroutine of the connection is stopped.
	APIClientReaderSTOPPED APIClientReaderState = iota
	// APIClientReaderRUNNING means the reader goroutine of the connection is running.
	APIClientReaderRUNNING
)

// APIClientWriterState is a const type for describing the execution state of an
// APIEndpoint writer goroutine.
type APIClientWriterState uint8

const (
	// APIClientWriterSTOPPED means the writer goroutine of the connection is stopped.
	APIClientWriterSTOPPED APIClientWriterState = iota
	// APIClientWriterRUNNING means the writer goroutine of the connection is running.
	APIClientWriterRUNNING
)

// APIClientState is a struct type that describes the state of an APIClient connection.
type APIClientState struct {
	readerState APIClientReaderState
	writerState APIClientWriterState
}

// APIClientInfoCentral holds the endpoint variable to access the goroutines
// responsible for communicating with the corresponding API client and the
// state of the API client. This struct is meant to be used as a value in a
// map[APIClient]*APIClientInfoCentral by the Central controller.
type APIClientInfoCentral struct {
	endpoint   *APIEndpoint
	state      APIClientState
	hasCrashed bool
}

// APIEndpoint holds a secure connection for communicating with the
// corresponding client. There are also 2 queues for the communication
// between the Central controller and the endpoint goroutines.
type APIEndpoint struct {
	apiClient APIClient
	conn      *net.TCPConn
	// MsgInQueue is the incoming message queue for
	// this APIEndpoint goroutine.
	MsgInQueue chan InternalMessage
	// MsgOutQueue is the outgoing message queue from
	// this APIEndpoint goroutine to the Central controller.
	MsgOutQueue chan InternalMessage
	// sigCh is used for signaling the reader goroutine to close gracefully.
	sigCh chan struct{}
	// A synchronozation variable to execute the Close method only once.
	closeOnce sync.Once
}

// APIListener is the goroutine that will listen for incoming API connection
// requests and it will open an APIEndpoint for each connection. Then by
// using 'MsgOutQueue', it will inform the Central controller.
type APIListener struct {
	ln *net.TCPListener
	// MsgOutQueue is the outgoing message queue from
	// this APIListener goroutine to the Central controller.
	MsgOutQueue chan InternalMessage
	// sigCh is used for signaling the api listener goroutine
	// to close gracefully.
	sigCh chan struct{}
}

// NewAPIListener is the constructor function of APIListener struct.
func NewAPIListener(apiAddr string, outQ chan InternalMessage) (*APIListener, error) {
	lAddr, err := net.ResolveTCPAddr("tcp", apiAddr)
	if err != nil {
		return nil, err
	}
	ln, err := net.ListenTCP("tcp", lAddr)
	if err != nil {
		return nil, err
	}

	return &APIListener{
		ln:          ln,
		MsgOutQueue: outQ,
		sigCh:       make(chan struct{}),
	}, nil
}

func (apiListener *APIListener) listenerRoutine() {
	// TODO: fill here

	// TODO: notify the Central controller before closing/returning!
}

// RunListenerGoroutine runs the goroutine that will listen
// for incoming api connections and serve them then inform
// the Central controller about it.
func (apiListener *APIListener) RunListenerGoroutine() {
	go apiListener.listenerRoutine()
}

// Close method initiates a graceful closing operation without blocking.
func (apiListener *APIListener) Close() error {
	// Closing the 'sigCh' channel signals the listener to close itself.
	close(apiListener.sigCh)
	apiListener.ln.Close()
	return nil
}

// NewAPIEndpoint is the constructor function of APIEndpoint struct.
func NewAPIEndpoint(apiAddr string, inQ, outQ chan InternalMessage) (*APIEndpoint, error) {
	conn, err := net.DialTimeout("tcp", apiAddr, connectionTimeout)
	tcpConn := conn.(*net.TCPConn)

	return &APIEndpoint{
		apiClient: APIClient{addr: apiAddr}, conn: tcpConn,
		MsgInQueue: inQ, MsgOutQueue: outQ,
		sigCh: make(chan struct{}), closeOnce: sync.Once{},
	}, err
}

func (apiEndpoint *APIEndpoint) readerRoutine() {
	// TODO: fill here

	// TODO: notify the Central controller before closing/returning!
}

// RunReaderGoroutine runs the goroutine that will read from
// the api connection, process the segments and route the
// corresponding InternalMessage to the Central controller.
func (apiEndpoint *APIEndpoint) RunReaderGoroutine() {
	go apiEndpoint.readerRoutine()
}

func (apiEndpoint *APIEndpoint) writerRoutine() {
	// TODO: fill here

	// TODO: notify the Central controller before closing/returning!
}

// RunWriterGoroutine runs the goroutine that will receive an
// InternalMessage from the Central controller, create the segment
// and write to the api connection in order to send the message.
func (apiEndpoint *APIEndpoint) RunWriterGoroutine() {
	go apiEndpoint.writerRoutine()
}

// Close method initiates a graceful closing operation without blocking.
func (apiEndpoint *APIEndpoint) Close() error {
	apiEndpoint.closeOnce.Do(func() {
		// TODO: send an InternalMessage to the writer for closing it!

		// Closing the 'sigCh' channel signals the reader to close itself.
		close(apiEndpoint.sigCh)
	})
	return nil
}

// HaveBothStopped return true iff both the reader and
// the writer have a STOPPED state.
func (s *APIClientState) HaveBothStopped() bool {
	return (s.readerState == APIClientReaderSTOPPED && s.writerState == APIClientWriterSTOPPED)
}

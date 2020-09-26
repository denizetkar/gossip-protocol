package core

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"gossip/src/datastruct/set"
	"io"
	"log"
	"net"
	"sync"
	"time"
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
	sigCh chan interface{}
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
	sigCh chan interface{}
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
		sigCh:       make(chan interface{}),
	}, nil
}

func (apiListener *APIListener) listenerRoutine() {
	defer apiListener.recover()
	for done := false; !done; {
		conn, err := apiListener.ln.AcceptTCP()
		if err != nil {
			select {
			case <-apiListener.sigCh:
				done = true
				continue
			default:
				log.Println("error in API Listener:", err)
				continue
			}
		}

		// Send the APIEndpoint to the controller
		client := APIClient{
			addr: conn.RemoteAddr().String(),
		}
		endp := &APIEndpoint{
			apiClient:   client,
			conn:        conn,
			MsgInQueue:  make(chan InternalMessage, outQueueSize),
			MsgOutQueue: apiListener.MsgOutQueue,
			sigCh:       make(chan interface{}),
			closeOnce:   sync.Once{},
		}
		log.Println("API Listener -> Central controller, APIEndpointCreatedMSG,", endp)
		apiListener.MsgOutQueue <- InternalMessage{Type: APIEndpointCreatedMSG, Payload: endp}
	}
	log.Println("API Listener -> Central controller, APIListenerClosedMSG")
	apiListener.MsgOutQueue <- InternalMessage{Type: APIListenerClosedMSG, Payload: void{}}
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

func (apiEndpoint *APIEndpoint) readerRoutine() {
	defer apiEndpoint.recover(true)

	reader := io.LimitReader(apiEndpoint.conn, 65535)
	bufioReader := bufio.NewReader(reader)

	binData := make([]byte, 65535)

	for done := false; !done; {
		select {
		case <-apiEndpoint.sigCh:
			done = true
			continue
		default:
			break
		}
		header := APIMessageHeader{}
		apiEndpoint.conn.SetDeadline(time.Now().Add(closureCheckTimeout))

		// Peek size of message
		var size []byte
		size, err := bufioReader.Peek(2)
		if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
			bufioReader.Reset(reader)
			continue
		} else if err != nil {
			panic(fmt.Sprint("Error in readerRoutine():", err))
		}
		sizeVal := binary.BigEndian.Uint16(size[:2])
		if sizeVal < 2 {
			bufioReader.Reset(reader)
			log.Println("Error in readerRoutine(): Received message is not larger than 2 bytes")
			continue
		}

		// Read Message header
		n, err := bufioReader.Read(binData)
		if err != nil {
			log.Println("Error in readerRoutine():", err)
			bufioReader.Reset(reader)
			continue
		}
		if n != int(sizeVal) {
			log.Println("Error in readerRoutine(): Malformed API message")
			bufioReader.Reset(reader)
			continue
		}
		binReader := bytes.NewReader(binData)
		err = binary.Read(binReader, binary.BigEndian, &header.Size)
		if err != nil {
			log.Println("Error in readerRoutine():", err)
			continue
		}

		err = binary.Read(binReader, binary.BigEndian, &header.MessageType)
		if err != nil {
			log.Println("Error in readerRoutine():", err)
			continue
		}

		switch header.MessageType {
		case GossipAnnounce:
			err := apiEndpoint.handleGossipAnnounce(binReader)
			if err != nil {
				log.Println("Error in readerRoutine():", err)
				continue
			}
		case GossipNotify:
			err := apiEndpoint.handleGossipNotify(binReader)
			if err != nil {
				log.Println("Error in readerRoutine():", err)
				continue
			}
		case GossipValidation:
			err := apiEndpoint.handleGossipValidation(binReader)
			if err != nil {
				log.Println("Error in readerRoutine():", err)
				continue
			}
		default:
			log.Println("Error in readerRoutine(): invalid MessageType used")
			break
		}
	}
	payload := APIEndpointClosedMSGPayload{endp: apiEndpoint, isReader: true}
	log.Println("API Endpoint -> Central controller, APIEndpointClosedMSG,", payload)
	apiEndpoint.MsgOutQueue <- InternalMessage{Type: APIEndpointClosedMSG, Payload: payload}
}

func (apiEndpoint *APIEndpoint) handleGossipAnnounce(binReader io.Reader) error {
	gossipItem := &GossipItem{}
	var ttl uint8
	err := binary.Read(binReader, binary.BigEndian, &ttl)
	if err != nil {
		return err
	}
	var reserved uint8
	err = binary.Read(binReader, binary.BigEndian, &reserved)
	if err != nil {
		return err
	}
	err = binary.Read(binReader, binary.BigEndian, &gossipItem.DataType)
	if err != nil {
		return err
	}
	var data []byte
	err = binary.Read(binReader, binary.BigEndian, &data)
	gossipItem.Data = string(data)
	if err != nil {
		return err
	}
	payload := GossipAnnounceMSGPayload{
		Item: gossipItem,
		TTL:  ttl,
	}
	payload2 := InternalMessage{Type: GossipAnnounceMSG, Payload: payload}
	log.Println("API Endpoint -> Central controller, IncomingAPIMSG,", payload2)
	apiEndpoint.MsgOutQueue <- InternalMessage{
		Type:    IncomingAPIMSG,
		Payload: payload2,
	}
	return nil
}
func (apiEndpoint *APIEndpoint) handleGossipNotify(binReader io.Reader) error {
	var reserved uint16
	err := binary.Read(binReader, binary.BigEndian, &reserved)
	if err != nil {
		return err
	}
	var dataType GossipItemDataType
	err = binary.Read(binReader, binary.BigEndian, &dataType)
	if err != nil {
		return err
	}
	payload := GossipNotifyMSGPayload{
		Who:  APIClient{addr: apiEndpoint.conn.RemoteAddr().String()},
		What: dataType,
	}
	payload2 := InternalMessage{Type: GossipNotifyMSG, Payload: payload}
	log.Println("API Endpoint -> Central controller, IncomingAPIMSG,", payload2)
	apiEndpoint.MsgOutQueue <- InternalMessage{
		Type:    IncomingAPIMSG,
		Payload: payload2,
	}
	return nil
}

func (apiEndpoint *APIEndpoint) handleGossipValidation(binReader io.Reader) error {
	var messageID uint16
	err := binary.Read(binReader, binary.BigEndian, &messageID)
	if err != nil {
		return err
	}
	reserved := make([]byte, 2)
	err = binary.Read(binReader, binary.BigEndian, &reserved)
	if err != nil {
		return err
	}
	payload := GossipValidationMSGPayload{
		Who: APIClient{addr: apiEndpoint.conn.RemoteAddr().String()},
		ID:  messageID,
		// Look at last bit and compare to 0
		Valid: 0 != int(reserved[1]&1),
	}
	payload2 := InternalMessage{Type: GossipValidationMSG, Payload: payload}
	log.Println("API Endpoint -> Central controller, IncomingAPIMSG,", payload2)
	apiEndpoint.MsgOutQueue <- InternalMessage{
		Type:    IncomingAPIMSG,
		Payload: payload2,
	}
	return nil
}
func (apiEndpoint *APIEndpoint) handleGossipNotification(_payload AnyMessage) error {
	payload := _payload.(APINotificationMSGPayload)
	// Combine messageID, dataType and data to message
	idByte := make([]byte, 2)
	binary.BigEndian.PutUint16(idByte, payload.ID)

	datatypeByte := make([]byte, 2)
	binary.BigEndian.PutUint16(datatypeByte, uint16(payload.Item.DataType))

	msg := append(idByte, datatypeByte...)
	msg = append(msg, []byte(payload.Item.Data)...)

	var size uint16
	if len([]byte(payload.Item.Data)) <= 65535-8 {
		size = 2 + 2 + uint16(len([]byte(payload.Item.Data))) + 2 + 2
	} else {
		return fmt.Errorf("APIEndpoint: Data field is too large")
	}
	sizeByte := make([]byte, 2)
	binary.BigEndian.PutUint16(sizeByte, uint16(size))

	messageTypeByte := make([]byte, 2)
	binary.BigEndian.PutUint16(messageTypeByte, uint16(GossipNotification))

	msg = append(messageTypeByte, msg...)
	msg = append(sizeByte, msg...)

	// Write message to client
	_, err := apiEndpoint.conn.Write(msg)
	return err
}

// RunReaderGoroutine runs the goroutine that will read from
// the api connection, process the segments and route the
// corresponding InternalMessage to the Central controller.
func (apiEndpoint *APIEndpoint) RunReaderGoroutine() {
	go apiEndpoint.readerRoutine()
}

func (apiEndpoint *APIEndpoint) writerRoutine() {
	defer apiEndpoint.recover(false)
	for done := false; !done; {
		select {
		case im := <-apiEndpoint.MsgInQueue:
			switch im.Type {
			case APIEndpointCloseMSG:
				done = true
				continue
			case APINotificationMSG:
				err := apiEndpoint.handleGossipNotification(im.Payload)
				if err != nil {
					log.Println("Error in writerRoutine():", err)
					continue
				}
			default:
				log.Println("Error in writerRoutine(): invalid internal message type used")
				break
			}
		}
	}
	payload := APIEndpointClosedMSGPayload{endp: apiEndpoint, isReader: false}
	log.Println("API Endpoint -> Central controller, APIEndpointClosedMSG,", payload)
	apiEndpoint.MsgOutQueue <- InternalMessage{Type: APIEndpointClosedMSG, Payload: payload}
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
		// Send an InternalMessage to the writer to close it.
		log.Println("Central controller -> API Endpoint, APIEndpointCloseMSG,", apiEndpoint.apiClient.addr)
		apiEndpoint.MsgInQueue <- InternalMessage{Type: APIEndpointCloseMSG, Payload: void{}}
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

// recover method tries to catch a panic in listenerRoutine if it exists, then
// informs the Central controller about the crash.
func (apiListener *APIListener) recover() {
	var err error
	if r := recover(); r != nil {
		// find out exactly what the error was and set err
		switch x := r.(type) {
		case string:
			err = fmt.Errorf(x)
		case error:
			err = x
		default:
			err = fmt.Errorf("Unknown panic in APIListener")
		}

		// send APIListenerCrashedMSG to the Central controller!
		log.Println("API Listener -> Central controller, APIListenerCrashedMSG,", err)
		apiListener.MsgOutQueue <- InternalMessage{Type: APIListenerCrashedMSG, Payload: err}
	}
}

// recover method tries to catch a panic in an APIEndpoint if it exists, then
// informs the Central controller about the crash.
func (apiEndpoint *APIEndpoint) recover(isReader bool) {
	var err error
	if r := recover(); r != nil {
		// find out exactly what the error was and set err
		switch x := r.(type) {
		case string:
			err = fmt.Errorf(x)
		case error:
			err = x
		default:
			err = fmt.Errorf("Unknown panic in APIEndpoint")
		}

		// send APIListenerCrashedMSG to the Central controller!
		payload := APIEndpointCrashedMSGPayload{endp: apiEndpoint, err: err, isReader: isReader}
		log.Println("API Endpoint -> Central controller, APIEndpointCrashedMSG,", apiEndpoint.apiClient.addr, err, isReader)
		apiEndpoint.MsgOutQueue <- InternalMessage{Type: APIEndpointCrashedMSG, Payload: payload}
	}
}

package main

import (
	"crypto/rsa"
	"crypto/securecomm"
	"crypto/x509"
	"datastruct/indexedmap"
	"datastruct/set"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	mrand "math/rand"
	"net"
	"os"
	"os/signal"
	"strings"
	"time"
)

var centralControllerHandlers map[InternalMessageType]func(*CentralController, AnyMessage) error
var centralControllerStopMessages set.Set

// init is an initialization function for 'main' package, called by Go.
func init() {
	mrand.Seed(time.Now().Unix())
	centralControllerHandlers = map[InternalMessageType]func(*CentralController, AnyMessage) error{}
	// Register all of the event handler methods.
	centralControllerHandlers[PeerAddMSG] = (*CentralController).peerAddHandler
	centralControllerHandlers[PeerRemoveMSG] = (*CentralController).peerRemoveHandler
	centralControllerHandlers[ProbePeerRequestMSG] = (*CentralController).probePeerRequestHandler
	centralControllerHandlers[MembershipCrashedMSG] = (*CentralController).membershipCrashedHandler
	centralControllerHandlers[MembershipClosedMSG] = (*CentralController).membershipClosedHandler
	centralControllerHandlers[RandomPeerListRequestMSG] = (*CentralController).randomPeerListRequestHandler
	centralControllerHandlers[RandomPeerListReleaseMSG] = (*CentralController).randomPeerListReleaseHandler
	centralControllerHandlers[GossiperCrashedMSG] = (*CentralController).gossiperCrashedHandler
	centralControllerHandlers[GossiperClosedMSG] = (*CentralController).gossiperClosedHandler
	centralControllerHandlers[APIListenerCrashedMSG] = (*CentralController).apiEndpointCrashedHandler
	centralControllerHandlers[APIListenerClosedMSG] = (*CentralController).apiEndpointClosedHandler
	centralControllerHandlers[APIEndpointCrashedMSG] = (*CentralController).apiEndpointCrashedHandler
	centralControllerHandlers[APIEndpointClosedMSG] = (*CentralController).apiEndpointClosedHandler
	centralControllerHandlers[P2PListenerCrashedMSG] = (*CentralController).p2pListenerCrashedHandler
	centralControllerHandlers[P2PListenerClosedMSG] = (*CentralController).p2pListenerClosedHandler
	centralControllerHandlers[P2PEndpointCrashedMSG] = (*CentralController).p2pEndpointCrashedHandler
	centralControllerHandlers[P2PEndpointClosedMSG] = (*CentralController).p2pEndpointClosedHandler
	centralControllerHandlers[OutgoingP2PCreatedMSG] = (*CentralController).outgoingP2PCreatedHandler
	centralControllerHandlers[CentralProbePeerReplyMSG] = (*CentralController).centralProbePeerReplyHandler
	centralControllerHandlers[CentralCrashMSG] = (*CentralController).crashHandler
	centralControllerHandlers[CentralCloseMSG] = (*CentralController).closeHandler
	// Create a set of valid event types while the Central controller is stopping.
	centralControllerStopMessages = set.New().Add(PeerRemoveMSG).
		Add(MembershipCrashedMSG).Add(MembershipClosedMSG).
		Add(GossiperCrashedMSG).Add(GossiperClosedMSG).
		Add(APIListenerCrashedMSG).Add(APIListenerClosedMSG).
		Add(APIEndpointCrashedMSG).Add(APIEndpointClosedMSG).
		Add(P2PListenerCrashedMSG).Add(P2PListenerClosedMSG).
		Add(P2PEndpointCrashedMSG).Add(P2PEndpointClosedMSG).
		Add(OutgoingP2PCreatedMSG).Add(CentralCrashMSG)
}

// GetOutboundIP attempts to find the public IP of the
// outgoing TCP or UDP connections.
func GetOutboundIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().String()
	idx := strings.LastIndex(localAddr, ":")
	return localAddr[0:idx], nil
}

// CentralControllerState is a struct type for describing not only the state
// of the Central controller but also for a summary of other running goroutines.
type CentralControllerState struct {
	isStopping      bool
	totalGoroutines int
}

// CentralControllerViewListKeyType is the key type of CentralController::viewList.
type CentralControllerViewListKeyType Peer

// CentralControllerViewListValueType is the value type of CentralController::viewList.
type CentralControllerViewListValueType *PeerInfoCentral

// CentralController contains core logic of the gossip module.
//
// NOTE: An outgoing p2p endpoint can be removed iff:
// (both reader and writer goroutines have stopped) AND
// (
//     (the endpoint was closed because the User requested closing) OR
//     (
//         (usageCounter <= 0) AND
//         (the Membership controller ordered the endpoint to be removed/closed)
//     )
// )
type CentralController struct {
	// p2pConfig contains the trusted identity path and RSA key information.
	p2pConfig *securecomm.Config
	// bootstrapper is the TCP\IP address of the bootstrapping peer.
	bootstrapper string
	// apiAddr is the TCP\IP address to listen for incoming API connections.
	apiAddr string
	// p2pAddr is the TCP\IP address to listen for incoming P2P connections.
	p2pAddr string
	// apiListener is the api listener goroutine
	apiListener *APIListener
	// apiListener is the p2p listener goroutine
	p2pListener *P2PListener
	// viewList is the current map of (Peer, *PeerInfoCentral) pairs for gossiping. It is of size O(n^0.25).
	viewList *indexedmap.IndexedMap
	// awaitingRemovalViewList is the current map of (Peer, *PeerInfoCentral) pairs awaiting deletion.
	// As soon as the 'usageCounter' of a peer in this map reaches 0, they are removed.
	awaitingRemovalViewList map[Peer]*PeerInfoCentral
	// activelyCreatedPeers is a set of Peers which are currently being created and cannot
	// be removed yet with a PeerRemoveMSG. If peer remove command for such a Peer arrives,
	// then the value of that peer is set to 'true', otherwise it is by default 'false'.
	activelyCreatedPeers map[Peer]bool
	// activelyProbedPeers is a set of Peers which are currently being probed and cannot
	// be opened connection with a PeerAddMSG. If peer add command for such a Peer arrives,
	// then the value of that peer is set to 'true', otherwise it is by default 'false'.
	activelyProbedPeers map[Peer]bool
	// incomingViewList is the current map of (Peer, *PeerInfoCentral) pairs where the remote
	// peer is the one who started the communication. Note that NO peer may use their
	// P2P listen address (IP, port) pair for starting a communication with another peer.
	// So, it is safe to start a communication to a peer that is inside 'incomingViewList'.
	incomingViewList    map[Peer]*PeerInfoCentral
	incomingViewListMAX uint16
	// apiClients is a map of currently active API client connections.
	apiClients    map[APIClient]*APIClientInfoCentral
	apiClientsMAX uint16
	// membershipController is the variable holding all the necessary variables
	// to communicate with the Membership controller goroutine.
	membershipController *MembershipController
	// gossiper is the variable holding all the necessary variables
	// to communicate with the Gossiper goroutine.
	gossiper *Gossiper
	// MsgInQueue is the incoming message queue for
	// the Central controller.
	MsgInQueue chan InternalMessage
	// state holds the Central controller state information.
	state CentralControllerState
}

const (
	// maxPeers is the maximum number of peers expected in the P2P network.
	// Since this parameter is critical for the correct operation of the network,
	// it is embedded into the source code instead of the config file. This way,
	// only "power users", who know what they are doing, can modify it!
	maxPeers                = 1e8
	alpha, beta             = 0.45, 0.45
	inQueueSize             = 1024
	outQueueSize            = 64
	membershipRoundDuration = 30 * time.Second
	gossipRoundDuration     = 500 * time.Millisecond
	connectionTimeout       = 2 * time.Second
	closureTimeout          = 6 * time.Second
)

// NewCentralController is a constructor function for the centralController class.
//
// trustedIdentitiesPath parameter is the path to the folder containing the
// empty files whose names are hex encoded 'identity' of the trusted peers.
// This folder HAS TO contain the identity of the 'bootstrapper' !!!
func NewCentralController(
	hostKeyPath, trustedIdentitiesPath, bootstrapper, apiAddr, p2pAddr string, cacheSize uint16, degree, maxTTL uint8,
) (*CentralController, error) {
	// Check the validity of trusted identities path
	s, err := os.Stat(trustedIdentitiesPath)
	if os.IsNotExist(err) {
		return nil, err
	} else if !s.IsDir() {
		return nil, fmt.Errorf("trustedIdentitiesPath is not a directory: %q", trustedIdentitiesPath)
	}
	// Check the validity of host key path (.pem file expected)
	s, err = os.Stat(hostKeyPath)
	if os.IsNotExist(err) {
		return nil, err
	} else if s.IsDir() {
		return nil, fmt.Errorf("hostKeyPath is a directory: %q", hostKeyPath)
	}
	// Check the validity of each TCP\IP address provided
	_, err = net.ResolveTCPAddr("tcp", bootstrapper)
	if err != nil {
		return nil, err
	}
	// Get the outbound ip address for TCP/UDP connections.
	ipAddr, err := GetOutboundIP()
	if err != nil {
		return nil, err
	}
	addr, err := net.ResolveTCPAddr("tcp", apiAddr)
	if err != nil {
		return nil, err
	}
	apiAddr = fmt.Sprintf("%s:%d", ipAddr, addr.Port)
	addr, err = net.ResolveTCPAddr("tcp", p2pAddr)
	if err != nil {
		return nil, err
	}
	p2pAddr = fmt.Sprintf("%s:%d", ipAddr, addr.Port)
	// Check the validity of the integer arguments
	if cacheSize == 0 || degree == 0 || degree > 10 {
		return nil, fmt.Errorf("invalid CentralController arguments, 'cache_size': %d, 'degree': %d", cacheSize, degree)
	}

	if maxTTL == 0 {
		maxTTL = uint8(math.Ceil(math.Log2(maxPeers) / math.Log2(math.Max(2, float64(degree)))))
	}
	viewListCap := uint16(math.Max(1, math.Floor(math.Pow(maxPeers, 0.25))))
	centralController := CentralController{
		bootstrapper:            bootstrapper,
		apiAddr:                 apiAddr,
		p2pAddr:                 p2pAddr,
		viewList:                indexedmap.New(),
		awaitingRemovalViewList: map[Peer]*PeerInfoCentral{},
		activelyProbedPeers:     map[Peer]bool{},
		incomingViewList:        map[Peer]*PeerInfoCentral{},
		incomingViewListMAX:     2 * viewListCap,
		apiClients:              map[APIClient]*APIClientInfoCentral{},
		apiClientsMAX:           cacheSize,
		MsgInQueue:              make(chan InternalMessage, inQueueSize),
	}
	// Read and load the RSA private key.
	priv, err := ioutil.ReadFile(hostKeyPath)
	if err != nil {
		return nil, err
	}
	privPem, _ := pem.Decode(priv)
	if privPem == nil || !strings.Contains(privPem.Type, "PRIVATE KEY") {
		return nil, errors.New("RSA key is not a valid '.pem' type private key")
	}
	privPemBytes := privPem.Bytes
	var parsedKey interface{}
	if parsedKey, err = x509.ParsePKCS1PrivateKey(privPemBytes); err != nil {
		if parsedKey, err = x509.ParsePKCS8PrivateKey(privPemBytes); err != nil {
			return nil, errors.New("Unable to parse RSA private key")
		}
	}
	privateKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("Unable to parse RSA private key")
	}
	centralController.p2pConfig = &securecomm.Config{
		TrustedIdentitiesPath: trustedIdentitiesPath,
		HostKey:               privateKey}

	apiListener, err := NewAPIListener(apiAddr, centralController.MsgInQueue)
	if err != nil {
		return nil, err
	}
	centralController.apiListener = apiListener

	// Create a new p2p listener.
	p2pListener, err := NewP2PListener(p2pAddr, centralController.MsgInQueue, centralController.p2pConfig)
	if err != nil {
		return nil, err
	}
	centralController.p2pListener = p2pListener
	// Create a new Membership controller.
	membershipController, err := NewMembershipController(
		bootstrapper, p2pAddr, alpha, beta, membershipRoundDuration, maxPeers, viewListCap,
		make(chan InternalMessage, outQueueSize), centralController.MsgInQueue,
	)
	if err != nil {
		return nil, err
	}
	centralController.membershipController = membershipController
	// Create a new Gossiper.
	gossiper, err := NewGossiper(
		cacheSize, degree, maxTTL, gossipRoundDuration, maxPeers,
		make(chan InternalMessage, outQueueSize), centralController.MsgInQueue,
	)
	if err != nil {
		return nil, err
	}
	centralController.gossiper = gossiper

	return &centralController, nil
}

// recover method tries to catch a panic in the Run method. If there is a
// panic, it logs and closes all submodules as soon as possible.
func (centralController *CentralController) recover() {
	var err error
	if r := recover(); r != nil {
		// find out exactly what the error was and set err
		switch x := r.(type) {
		case string:
			err = errors.New(x)
		case error:
			err = x
		default:
			err = errors.New("Unknown panic in Central controller")
		}

		// TODO: Log the crash.
		fmt.Println(err)
		os.Exit(1)
	}
}

// peerAddHandler is the method called by the Run method for when
// it receives an internal message of type PeerAddMSG.
func (centralController *CentralController) peerAddHandler(payload AnyMessage) error {
	peer, ok := payload.(Peer)
	if ok {
		// If the peer is already in the view list, then nothing more to do.
		if centralController.viewList.IsMember(peer) {
			return nil
		}
		// If the peer is in the removal view list, then move it back to the view list.
		if info, isMember := centralController.awaitingRemovalViewList[peer]; isMember {
			delete(centralController.awaitingRemovalViewList, peer)
			centralController.viewList.Put(peer, info)
			return nil
		}
		// If the peer is already being created, then
		// signal it to be not removed later.
		if _, isMember := centralController.activelyCreatedPeers[peer]; isMember {
			centralController.activelyCreatedPeers[peer] = false
			return nil
		}
		// If the peer is currently being probed, then signal it to be added later.
		if _, isMember := centralController.activelyProbedPeers[peer]; isMember {
			centralController.activelyProbedPeers[peer] = true
			return nil
		}
		// Register this peer as an actively created peer to not be removed later.
		centralController.activelyCreatedPeers[peer] = false
		// Create an endpoint for the outgoing p2p connection.
		go func(peer Peer) {
			endp, _ := NewP2PEndpoint(
				peer.Addr, centralController.p2pConfig,
				make(chan InternalMessage, outQueueSize),
				centralController.MsgInQueue, true)
			centralController.MsgInQueue <- InternalMessage{Type: OutgoingP2PCreatedMSG, Payload: endp}
		}(peer)
	}

	return nil
}

// peerRemoveHandler is the method called by the Run method for when
// it receives an internal message of type PeerRemoveMSG.
func (centralController *CentralController) peerRemoveHandler(payload AnyMessage) error {
	peer, ok := payload.(Peer)
	if ok {
		// If the peer is in the removal view list, then nothing more to do.
		if _, isMember := centralController.awaitingRemovalViewList[peer]; isMember {
			return nil
		}
		// If the peer is in the view list, then send a command to close.
		if centralController.viewList.IsMember(peer) {
			value := centralController.viewList.GetValue(peer)
			info := value.(*PeerInfoCentral)
			centralController.viewList.Remove(peer)
			centralController.awaitingRemovalViewList[peer] = info
			// If there is no gossip item using this outgoing peer.
			if info.usageCounter <= 0 {
				// If the p2p endpoint has already stopped.
				if info.state.HaveBothStopped() {
					delete(centralController.awaitingRemovalViewList, peer)
				} else {
					info.endpoint.Close()
				}
			}
			return nil
		}
		// If the peer is currently being created, signal it to be removed later.
		if _, isMember := centralController.activelyCreatedPeers[peer]; isMember {
			centralController.activelyCreatedPeers[peer] = true
			return nil
		}
		// If the peer is currently being probed, then signal it to be not added later.
		if _, isMember := centralController.activelyProbedPeers[peer]; isMember {
			centralController.activelyProbedPeers[peer] = false
			return nil
		}
	}

	return nil
}

// probePeerRequestHandler is the method called by the Run method for when
// it receives an internal message of type ProbePeerRequestMSG.
func (centralController *CentralController) probePeerRequestHandler(payload AnyMessage) error {
	peer, ok := payload.(Peer)
	if ok {
		// Check if the peer is already in either the view list or the removal view list
		// or the actively created peer list.
		_, isMember := centralController.awaitingRemovalViewList[peer]
		_, isMember2 := centralController.activelyCreatedPeers[peer]
		if centralController.viewList.IsMember(peer) || isMember || isMember2 {
			centralController.membershipController.MsgInQueue <- InternalMessage{
				Type:    ProbePeerReplyMSG,
				Payload: ProbePeerReplyMSGPayload{Probed: peer, ProbeResult: true},
			}
			return nil
		}
		// Register the peer for probing.
		centralController.activelyProbedPeers[peer] = false
		// Start a goroutine to probe the peer.
		go func(peer Peer) {
			conn, err := net.DialTimeout("tcp", peer.Addr, connectionTimeout)
			probeResult := (err == nil)
			conn.Close()
			centralController.MsgInQueue <- InternalMessage{
				Type:    CentralProbePeerReplyMSG,
				Payload: CentralProbePeerReplyMSGPayload{Probed: peer, ProbeResult: probeResult},
			}
		}(peer)
	}

	return nil
}

// membershipCrashedHandler is the method called by the Run method for when
// it receives an internal message of type MembershipCrashedMSG.
func (centralController *CentralController) membershipCrashedHandler(payload AnyMessage) error {
	err, ok := payload.(error)
	if ok {
		// TODO: Log the crash.
		fmt.Println("Membership controller has crashed.")
		panic(err)
	}

	return nil
}

// membershipClosedHandler is the method called by the Run method for when
// it receives an internal message of type MembershipClosedMSG.
func (centralController *CentralController) membershipClosedHandler(payload AnyMessage) error {
	_, ok := payload.(void)
	if ok {
		centralController.membershipController = nil
		// TODO: Log the graceful closure.
		fmt.Println("Membership controller is closed.")

		centralController.state.totalGoroutines--
		if centralController.state.totalGoroutines <= 0 {
			// Signal for graceful closure.
			return &CloseError{}
		}
	}

	return nil
}

// randomPeerListRequestHandler is the method called by the Run method for when
// it receives an internal message of type RandomPeerListRequestMSG.
func (centralController *CentralController) randomPeerListRequestHandler(payload AnyMessage) error {
	msg, ok := payload.(RandomPeerListRequestMSGPayload)
	if ok {
		// Create a random list of peers as a response.
		var RandomPeers []Peer
		// Pick at random msg.Num of the peer in the view list.
		randomIndexes := mrand.Perm(centralController.viewList.Len())[:msg.Num]
		for _, i := range randomIndexes {
			key, _ := centralController.viewList.KeyAtIndex(i)
			peer := key.(Peer)
			RandomPeers = append(RandomPeers, peer)
			// Increment the usage counter
			value := centralController.viewList.GetValue(key)
			info := value.(*PeerInfoCentral)
			info.usageCounter++
		}
		// Send the random list of peers as a response back to the Gossiper submodule.
		centralController.gossiper.MsgInQueue <- InternalMessage{
			Type:    RandomPeerListReplyMSG,
			Payload: RandomPeerListReplyMSGPayload{Related: msg.Related, RandomPeers: RandomPeers},
		}
	}

	return nil
}

// randomPeerListReleaseHandler is the method called by the Run method for when
// it receives an internal message of type RandomPeerListReleaseMSG.
func (centralController *CentralController) randomPeerListReleaseHandler(payload AnyMessage) error {
	msg, ok := payload.(RandomPeerListReleaseMSGPayload)
	if ok {
		peerList := msg.ReleasedPeers
		if peerList == nil {
			return nil
		}
		// Decrement the usage counters of each peer.
		for _, peer := range peerList {
			// Check if the peer is either in the view list or the removal view list.
			if centralController.viewList.IsMember(peer) {
				value := centralController.viewList.GetValue(peer)
				info := value.(*PeerInfoCentral)
				info.usageCounter--
			} else if info, isMember := centralController.awaitingRemovalViewList[peer]; isMember {
				info.usageCounter--
				// If there is no gossip item using this outgoing peer.
				if info.usageCounter <= 0 {
					// If the p2p endpoint has already stopped.
					if info.state.HaveBothStopped() {
						delete(centralController.awaitingRemovalViewList, peer)
					} else {
						info.endpoint.Close()
					}
				}
			} else {
				// An outgoing p2p endpoint should not have been deleted before
				// (usageCounter <= 0). Unless the User requested a shutdown. But during
				// a shutdown, this event handler cannot be called, so the code must
				// have never reached here!
				// TODO: log this unexpected event.
				fmt.Println("Outgoing P2P endpoint", peer.Addr, "was deleted before (usageCounter <= 0).")
			}
		}
	}

	return nil
}

// gossiperCrashedHandler is the method called by the Run method for when
// it receives an internal message of type GossiperCrashedMSG.
func (centralController *CentralController) gossiperCrashedHandler(payload AnyMessage) error {
	err, ok := payload.(error)
	if ok {
		// TODO: Log the crash.
		fmt.Println("Gossiper has crashed.")
		panic(err)
	}

	return nil
}

// gossiperClosedHandler is the method called by the Run method for when
// it receives an internal message of type GossiperClosedMSG.
func (centralController *CentralController) gossiperClosedHandler(payload AnyMessage) error {
	_, ok := payload.(void)
	if ok {
		centralController.gossiper = nil
		// TODO: Log the graceful closure.
		fmt.Println("Gossiper controller is closed.")

		centralController.state.totalGoroutines--
		if centralController.state.totalGoroutines <= 0 {
			// Signal for graceful closure.
			return &CloseError{}
		}
	}

	return nil
}

// apiListenerCrashedHandler is the method called by the Run method for when
// it receives an internal message of type APIListenerCrashedMSG.
func (centralController *CentralController) apiListenerCrashedHandler(payload AnyMessage) error {
	err, ok := payload.(error)
	if ok {
		// TODO: Log the crash.
		fmt.Println("API listener has crashed.")
		panic(err)
	}

	return nil
}

// apiListenerClosedHandler is the method called by the Run method for when
// it receives an internal message of type APIListenerClosedMSG.
func (centralController *CentralController) apiListenerClosedHandler(payload AnyMessage) error {
	_, ok := payload.(void)
	if ok {
		centralController.apiListener = nil
		// TODO: Log the graceful closure.
		fmt.Println("API listener is closed.")

		centralController.state.totalGoroutines--
		// Check if all submodules (goroutines) are closed.
		if centralController.state.totalGoroutines <= 0 {
			// Signal for graceful closure.
			return &CloseError{}
		}
	}

	return nil
}

// apiEndpointClosed is the method called when an api endpoint is closed.
func (centralController *CentralController) apiEndpointClosed(
	endp *APIEndpoint, isReader bool, err error) error {
	info, isMember := centralController.apiClients[endp.apiClient]
	// Check if the api client exists.
	if !isMember {
		return nil
	}
	// Check if the reader or the writer closed.
	if isReader {
		if info.state.readerState == APIClientReaderSTOPPED {
			return nil
		}
		info.state.readerState = APIClientReaderSTOPPED
	} else {
		if info.state.writerState == APIClientWriterSTOPPED {
			return nil
		}
		info.state.writerState = APIClientWriterSTOPPED
	}
	centralController.state.totalGoroutines--
	// If the endpoint closed with an error, it must have crashed.
	if err != nil {
		info.hasCrashed = true
	}
	// Check if both reader and writer are stopped.
	if info.state.HaveBothStopped() {
		delete(centralController.apiClients, endp.apiClient)
		// Check if the api endpoint is not supposed to be closed.
		if info.hasCrashed {
			// TODO: Log the unexpected closure.
			fmt.Println("API endpoint", endp.apiClient.addr, "has crashed.")
			fmt.Println(err)
		} else {
			// TODO: Log the graceful closure.
			fmt.Println("API endpoint", endp.apiClient.addr, "is closed.")
		}
		// Let the Gossiper know about the removed endpoint.
		centralController.gossiper.MsgInQueue <- InternalMessage{
			Type: GossipUnnofityMSG, Payload: endp.apiClient}
		// Check if all submodules (goroutines) are closed.
		if centralController.state.totalGoroutines <= 0 {
			// Signal for graceful closure.
			return &CloseError{}
		}
	} else if info.hasCrashed {
		// Either reader or writer is still running. Stop it!
		info.endpoint.Close()
	}

	return nil
}

// apiEndpointCrashedHandler is the method called by the Run method for when
// it receives an internal message of type APIEndpointCrashedMSG.
func (centralController *CentralController) apiEndpointCrashedHandler(payload AnyMessage) error {
	msg, ok := payload.(APIEndpointCrashedMSGPayload)
	if ok {
		return centralController.apiEndpointClosed(msg.endp, msg.isReader, msg.err)
	}

	return nil
}

// apiEndpointClosedHandler is the method called by the Run method for when
// it receives an internal message of type APIEndpointClosedMSG.
func (centralController *CentralController) apiEndpointClosedHandler(payload AnyMessage) error {
	msg, ok := payload.(APIEndpointClosedMSGPayload)
	if ok {
		return centralController.apiEndpointClosed(msg.endp, msg.isReader, nil)
	}

	return nil
}

// p2pListenerCrashedHandler is the method called by the Run method for when
// it receives an internal message of type P2PListenerCrashedMSG.
func (centralController *CentralController) p2pListenerCrashedHandler(payload AnyMessage) error {
	err, ok := payload.(error)
	if ok {
		// TODO: Log the crash.
		fmt.Println("P2P listener has crashed.")
		panic(err)
	}

	return nil
}

// p2pListenerClosedHandler is the method called by the Run method for when
// it receives an internal message of type P2PListenerClosedMSG.
func (centralController *CentralController) p2pListenerClosedHandler(payload AnyMessage) error {
	_, ok := payload.(void)
	if ok {
		centralController.p2pListener = nil
		// TODO: Log the graceful closure.
		fmt.Println("P2P listener is closed.")

		centralController.state.totalGoroutines--
		// Check if all submodules (goroutines) are closed.
		if centralController.state.totalGoroutines <= 0 {
			// Signal for graceful closure.
			return &CloseError{}
		}
	}

	return nil
}

// outgoingPeerCompletelyClosed is the method called when both the reader and
// the writer goroutines of an outgoing p2p endpoint are closed.
func (centralController *CentralController) outgoingPeerCompletelyClosed(
	info *PeerInfoCentral, isInRemovalList bool, err error) error {
	peer := info.endpoint.peer
	// Check if the p2p endpoint is not supposed to be closed.
	if info.hasCrashed {
		// TODO: Log the unexpected closure.
		fmt.Println("Outgoing P2P endpoint", peer.Addr, "has crashed.")
		fmt.Println(err)
		if centralController.state.isStopping {
			// If it was the User that ordered the closure, remove as soon as
			// both the reader and the writer goroutines are stopped.
			centralController.viewList.Remove(peer)
			delete(centralController.awaitingRemovalViewList, peer)
		} else if !isInRemovalList {
			// If this p2p endpoint was not removed by the Membership controller, then
			// let the Membership controller know about the abruptly removed endpoint.
			centralController.membershipController.MsgInQueue <- InternalMessage{
				Type: PeerDisconnectedMSG, Payload: peer}
		} else if info.usageCounter <= 0 {
			// If this p2p endpoint was removed by the orders of the Membership
			// controller, then check if the usage counter reached 0. If so, then
			// delete it.
			delete(centralController.awaitingRemovalViewList, peer)
		}
	} else {
		// TODO: Log the graceful closure.
		fmt.Println("Outgoing P2P endpoint", peer.Addr, "is closed.")
		if centralController.state.isStopping {
			// If it was the User that ordered the closure, remove as soon as
			// both the reader and the writer goroutines are stopped.
			centralController.viewList.Remove(peer)
			delete(centralController.awaitingRemovalViewList, peer)
		} else if !isInRemovalList {
			// If this p2p endpoint was not removed by the Membership controller, then
			// it must not have gracefully closed!
			// TODO: log this unexpected event.
			fmt.Println("Outgoing P2P endpoint", peer.Addr, "is closed without "+
				"the explicit request of neither the User nor the Membership controller!")
		} else if info.usageCounter <= 0 {
			// If this p2p endpoint was removed by the orders of the Membership
			// controller, then check if the usage counter reached 0. If so, then
			// delete it.
			delete(centralController.awaitingRemovalViewList, peer)
		}
	}
	// Check if all submodules (goroutines) are closed.
	if centralController.state.totalGoroutines <= 0 {
		// Signal for graceful closure.
		return &CloseError{}
	}

	return nil
}

// outgoingPeerClosed is the method called when either the reader or the writer
// goroutine of an outgoing p2p endpoint is closed.
func (centralController *CentralController) outgoingPeerClosed(
	endp *P2PEndpoint, isReader bool, err error) error {
	var info *PeerInfoCentral
	// Check if the peer exists in either the view list or the removal view list.
	_info, isInRemovalList := centralController.awaitingRemovalViewList[endp.peer]
	if centralController.viewList.IsMember(endp.peer) {
		value := centralController.viewList.GetValue(endp.peer)
		info = value.(*PeerInfoCentral)
	} else if isInRemovalList {
		info = _info
	} else {
		return nil
	}
	// Check if the reader or the writer closed.
	if isReader {
		if info.state.readerState == PeerReaderSTOPPED {
			return nil
		}
		info.state.readerState = PeerReaderSTOPPED
	} else {
		if info.state.writerState == PeerWriterSTOPPED {
			return nil
		}
		info.state.writerState = PeerWriterSTOPPED
	}
	centralController.state.totalGoroutines--
	// If the endpoint closed with an error, it must have crashed.
	if err != nil {
		info.hasCrashed = true
	}
	// Check if both reader and writer are stopped.
	if info.state.HaveBothStopped() {
		return centralController.outgoingPeerCompletelyClosed(info, isInRemovalList, err)
	} else if info.hasCrashed {
		// Either reader or writer is still running. Stop it!
		info.endpoint.Close()
	}

	return nil
}

// incomingPeerClosed is the method called when an incoming p2p endpoint is closed.
func (centralController *CentralController) incomingPeerClosed(
	endp *P2PEndpoint, isReader bool, err error) error {
	info, isMember := centralController.incomingViewList[endp.peer]
	// Check if the peer exists in the incoming view list.
	if !isMember {
		return nil
	}
	// Check if the reader or the writer closed.
	if isReader {
		if info.state.readerState == PeerReaderSTOPPED {
			return nil
		}
		info.state.readerState = PeerReaderSTOPPED
	} else {
		if info.state.writerState == PeerWriterSTOPPED {
			return nil
		}
		info.state.writerState = PeerWriterSTOPPED
	}
	centralController.state.totalGoroutines--
	// If the endpoint closed with an error, it must have crashed.
	if err != nil {
		info.hasCrashed = true
	}
	// Check if both reader and writer are stopped.
	if info.state.HaveBothStopped() {
		delete(centralController.incomingViewList, endp.peer)
		// Check if the p2p endpoint is not supposed to be closed.
		if info.hasCrashed {
			// TODO: Log the unexpected closure.
			fmt.Println("Incoming P2P endpoint", endp.peer.Addr, "has crashed.")
			fmt.Println(err)
		} else {
			// TODO: Log the graceful closure.
			fmt.Println("Incoming P2P endpoint", endp.peer.Addr, "is closed.")
		}
		// Check if all submodules (goroutines) are closed.
		if centralController.state.totalGoroutines <= 0 {
			// Signal for graceful closure.
			return &CloseError{}
		}
	} else if info.hasCrashed {
		// Either reader or writer is still running. Stop it!
		info.endpoint.Close()
	}

	return nil
}

// p2pEndpointCrashedHandler is the method called by the Run method for when
// it receives an internal message of type P2PEndpointCrashedMSG.
func (centralController *CentralController) p2pEndpointCrashedHandler(payload AnyMessage) error {
	msg, ok := payload.(P2PEndpointCrashedMSGPayload)
	if ok {
		if msg.endp.isOutgoing {
			return centralController.outgoingPeerClosed(msg.endp, msg.isReader, msg.err)
		}
		return centralController.incomingPeerClosed(msg.endp, msg.isReader, msg.err)
	}

	return nil
}

// p2pEndpointClosedHandler is the method called by the Run method for when
// it receives an internal message of type P2PEndpointClosedMSG.
func (centralController *CentralController) p2pEndpointClosedHandler(payload AnyMessage) error {
	msg, ok := payload.(P2PEndpointClosedMSGPayload)
	if ok {
		if msg.endp.isOutgoing {
			return centralController.outgoingPeerClosed(msg.endp, msg.isReader, nil)
		}
		return centralController.incomingPeerClosed(msg.endp, msg.isReader, nil)
	}

	return nil
}

// outgoingP2PCreatedHandler is the method called by the Run method for when
// it receives an internal message of type OutgoingP2PCreatedMSG.
func (centralController *CentralController) outgoingP2PCreatedHandler(payload AnyMessage) error {
	endp, ok := payload.(*P2PEndpoint)
	if ok {
		// Check if the peer of this endpoint was registered.
		isToBeRemoved, isMember := centralController.activelyCreatedPeers[endp.peer]
		if !isMember {
			// TODO: log this unexpected event.
			fmt.Println("Outgoing P2P endpoint", endp.peer.Addr, "was created without registration!")
			return nil
		}
		delete(centralController.activelyCreatedPeers, endp.peer)
		// Start running the reader and writer goroutines.
		endp.RunReaderGoroutine()
		endp.RunWriterGoroutine()
		// Account for the reader and writer goroutines.
		centralController.state.totalGoroutines += 2
		// Add the peer into the view list.
		centralController.viewList.Put(endp.peer,
			&PeerInfoCentral{
				endpoint: endp, usageCounter: 0,
				state: PeerState{
					readerState: PeerReaderRUNNING,
					writerState: PeerWriterRUNNING},
				hasCrashed: false,
			})
		// If this peer was attempted to be removed before creation
		// was done, then let it be removed.
		if isToBeRemoved {
			centralController.MsgInQueue <- InternalMessage{Type: PeerRemoveMSG, Payload: endp.peer}
		}
	}

	return nil
}

// centralProbePeerReplyHandler is the method called by the Run method for when
// it receives an internal message of type CentralProbePeerReplyMSG.
func (centralController *CentralController) centralProbePeerReplyHandler(payload AnyMessage) error {
	msg, ok := payload.(CentralProbePeerReplyMSGPayload)
	if ok {
		// Check if for some unexpected reason the probed peer is not registered.
		addPeer, isMember := centralController.activelyProbedPeers[msg.Probed]
		if !isMember {
			// TODO: log this unexpected event.
			fmt.Println("Peer", msg.Probed.Addr, "was probed without registration!")
			return nil
		}
		delete(centralController.activelyProbedPeers, msg.Probed)
		// Send the probe results back to the Membership controller.
		centralController.membershipController.MsgInQueue <- InternalMessage{
			Type:    ProbePeerReplyMSG,
			Payload: ProbePeerReplyMSGPayload{Probed: msg.Probed, ProbeResult: msg.ProbeResult},
		}
		// If this peer was attempted to be added before probing
		// was done, then let it be added.
		if addPeer {
			centralController.MsgInQueue <- InternalMessage{Type: PeerAddMSG, Payload: msg.Probed}
		}
	}

	return nil
}

// crashHandler is the method called by the Run method for when
// it receives an internal message of type CentralCrashMSG.
func (centralController *CentralController) crashHandler(payload AnyMessage) error {
	err, ok := payload.(error)
	if ok {
		panic(err)
	}

	return nil
}

// closeHandler is the method called by the Run method for when
// it receives an internal message of type CentralCloseMSG.
func (centralController *CentralController) closeHandler(payload AnyMessage) error {
	_, ok := payload.(void)
	if ok {
		// TODO: Log the graceful closure.
		fmt.Println("Central controller is closing.")
		// Before closing the Central controller, make sure to have already
		// closed all other submodules (goroutines)!
		centralController.apiListener.Close()
		centralController.p2pListener.Close()
		centralController.membershipController.MsgInQueue <- InternalMessage{Type: MembershipCloseMSG, Payload: void{}}
		centralController.gossiper.MsgInQueue <- InternalMessage{Type: GossiperCloseMSG, Payload: void{}}
		for _, valueAndIndex := range centralController.viewList.Iterate() {
			info := valueAndIndex.Value.(*PeerInfoCentral)
			info.endpoint.Close()
		}
		for _, info := range centralController.awaitingRemovalViewList {
			info.endpoint.Close()
		}
		// Signal all actively created outgoing p2p endpoints to be removed later.
		for peer := range centralController.activelyCreatedPeers {
			centralController.activelyCreatedPeers[peer] = true
		}
		// Signal all actively probed peers to be not added later.
		for peer := range centralController.activelyProbedPeers {
			centralController.activelyProbedPeers[peer] = false
		}
		for _, info := range centralController.apiClients {
			info.endpoint.Close()
		}

		centralController.state.isStopping = true
		go func() {
			time.Sleep(closureTimeout)
			centralController.MsgInQueue <- InternalMessage{
				Type: CentralCrashMSG, Payload: errors.New("graceful closure timed out")}
		}()
	}

	return nil
}

// Run is the core logic of this Gossip module.
func (centralController *CentralController) Run() {
	defer centralController.recover()
	// Register for the signals generated by the OS (especially for
	// the purpose of catching shutdown request from the User).
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	go func() {
		<-sigs
		centralController.MsgInQueue <- InternalMessage{Type: CentralCloseMSG, Payload: void{}}
	}()

	// TODO: Run the Membership controller and Gossiper.
	//centralController.membershipController.RunControllerGoroutine()
	//centralController.gossiper.RunControllerGoroutine()

	// TODO: Run the API and P2P listeners.
	//centralController.p2pListener.RunListenerGoroutine()
	//centralController.apiListener.RunListenerGoroutine()

	// Account for the 2 listener submodules and 2 controller submodules.
	centralController.state.totalGoroutines += 4

	for done := false; !done; {
		// Check for any incoming event.
		select {
		case im := <-centralController.MsgInQueue:
			if centralController.state.isStopping && !centralControllerStopMessages.IsMember(im.Type) {
				break
			}
			handler := centralControllerHandlers[im.Type]
			err := handler(centralController, im.Payload)
			if err != nil {
				switch err.(type) {
				case *CloseError:
					done = true
				default:
					break
				}
			}
		}
	}
}

func (centralController *CentralController) String() string {
	reprFormat := "*CentralController{\n" +
		"\tp2pConfig: %v,\n" +
		"\tbootstrapper: %q,\n" +
		"\tapiAddr: %q,\n" +
		"\tp2pAddr: %q,\n" +
		"\tapiListener: %v,\n" +
		"\tp2pListener: %v,\n" +
		"\tviewList: %v,\n" +
		"\tawaitingRemovalViewList: %s,\n" +
		"\tactivelyCreatedPeers: %s,\n" +
		"\tactivelyProbedPeers: %s,\n" +
		"\tincomingViewList: %s,\n" +
		"\tincomingViewListMAX: %d,\n" +
		"\tapiClients: %s,\n" +
		"\tapiClientsMAX: %d,\n" +
		"\tmembershipController: %s,\n" +
		"\tgossiper: %s,\n" +
		"\tstate: %v,\n" +
		"}"
	return fmt.Sprintf(reprFormat,
		centralController.p2pConfig,
		centralController.bootstrapper,
		centralController.apiAddr,
		centralController.p2pAddr,
		centralController.apiListener,
		centralController.p2pListener,
		centralController.viewList,
		centralController.awaitingRemovalViewList,
		centralController.activelyCreatedPeers,
		centralController.activelyProbedPeers,
		centralController.incomingViewList,
		centralController.incomingViewListMAX,
		centralController.apiClients,
		centralController.apiClientsMAX,
		centralController.membershipController,
		centralController.gossiper,
		centralController.state,
	)
}

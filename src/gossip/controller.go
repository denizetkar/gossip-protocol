package main

import (
	"crypto/securecomm"
	"datastruct/indexedmap"
	"fmt"
	"math"
	mrand "math/rand"
	"net"
	"os"
	"strings"
	"time"
)

// init is an initialization function for 'main' package, called by Go.
func init() {
	mrand.Seed(time.Now().Unix())
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
	isStopping                    bool
	isGossiperRunning             bool
	isMembershipControllerRunning bool
	isAPIListenerRunning          bool
	isP2PListenerRunning          bool
	totalGoroutines               uint32
}

// CentralControllerViewListKeyType is the key type of CentralController::viewList.
type CentralControllerViewListKeyType Peer

// CentralControllerViewListValueType is the value type of CentralController::viewList.
type CentralControllerViewListValueType *PeerInfoCentral

// CentralController contains core logic of the gossip module.
type CentralController struct {
	// trustedIdentitiesPath is the path to the folder containing the
	// empty files whose names are hex encoded 'identity' of the trusted peers.
	// This folder HAS TO contain the identity of the 'bootstrapper' !!!
	trustedIdentitiesPath string
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
	// viewList is the current map of (Peer, PeerInfoCentral) pairs for gossiping. It is of size O(n^0.25).
	viewList    *indexedmap.IndexedMap
	viewListCap uint16
	// awaitingRemovalViewList is the current map of (Peer, PeerInfoCentral) pairs awaiting deletion.
	// As soon as the 'usageCounter' of a peer in this map reaches 0, they are removed.
	awaitingRemovalViewList map[Peer]*PeerInfoCentral
	// incomingViewList is the current map of (Peer, PeerInfoCentral) pairs where the remote
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

// NewCentralController is a constructor function for the centralController class.
func NewCentralController(
	trustedIdentitiesPath, bootstrapper, apiAddr, p2pAddr string, cacheSize uint16, degree, maxTTL uint8,
) (*CentralController, error) {
	// Check the validity of trusted identities path
	s, err := os.Stat(trustedIdentitiesPath)
	if os.IsNotExist(err) {
		return nil, err
	} else if !s.IsDir() {
		return nil, fmt.Errorf("trustedIdentitiesPath is not a directory: %q", trustedIdentitiesPath)
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

	// maxPeers is the maximum number of peers expected in the P2P network.
	// Since this parameter is critical for the correct operation of the network,
	// it is embedded into the source code instead of the config file. This way,
	// only "power users", who know what they are doing, can modify it!
	maxPeers := 1e8
	alpha, beta := 0.45, 0.45
	inQueueSize := 1024
	outQueueSize := 64
	membershipRoundDuration := 30 * time.Second
	gossipRoundDuration := 500 * time.Millisecond

	if maxTTL == 0 {
		maxTTL = uint8(math.Ceil(math.Log2(maxPeers) / math.Log2(math.Max(2, float64(degree)))))
	}
	viewListCap := uint16(math.Max(1, math.Floor(math.Pow(maxPeers, 0.25))))
	centralController := CentralController{
		trustedIdentitiesPath:   trustedIdentitiesPath,
		bootstrapper:            bootstrapper,
		apiAddr:                 apiAddr,
		p2pAddr:                 p2pAddr,
		viewList:                indexedmap.New(),
		viewListCap:             viewListCap,
		awaitingRemovalViewList: map[Peer]*PeerInfoCentral{},
		incomingViewList:        map[Peer]*PeerInfoCentral{},
		incomingViewListMAX:     2 * viewListCap,
		apiClients:              map[APIClient]*APIClientInfoCentral{},
		apiClientsMAX:           cacheSize,
		MsgInQueue:              make(chan InternalMessage, inQueueSize),
	}

	apiListener, err := NewAPIListener(apiAddr, centralController.MsgInQueue)
	if err != nil {
		return nil, err
	}
	centralController.apiListener = apiListener

	// TODO: completely describe the secure communication configs below!
	p2pConfig := securecomm.Config{}
	p2pListener, err := NewP2PListener(p2pAddr, centralController.MsgInQueue, &p2pConfig)
	if err != nil {
		return nil, err
	}
	centralController.p2pListener = p2pListener

	membershipController, err := NewMembershipController(
		bootstrapper, p2pAddr, alpha, beta, membershipRoundDuration, maxPeers, centralController.viewListCap,
		make(chan InternalMessage, outQueueSize), centralController.MsgInQueue,
	)
	if err != nil {
		return nil, err
	}
	centralController.membershipController = membershipController

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

// Run is the core logic of this Gossip module.
func (centralController *CentralController) Run() {
	// TODO: change the code below to the real logic !!!

	// Before closing the Central controller, make sure to have already
	// closed all other submodules (goroutines)! Use CentralControllerState
	// for the purposes of tracking which submodule is up or down.
	centralController.apiListener.Close()
	centralController.p2pListener.Close()
}

func (centralController *CentralController) String() string {
	reprFormat := "*CentralController{\n" +
		"\ttrustedIdentitiesPath: %q,\n" +
		"\tbootstrapper: %q,\n" +
		"\tapiAddr: %q,\n" +
		"\tp2pAddr: %q,\n" +
		"\tapiListener: %v,\n" +
		"\tp2pListener: %v,\n" +
		"\tviewList: %v,\n" +
		"\tviewListCap: %d,\n" +
		"\tawaitingRemovalViewList: %s,\n" +
		"\tincomingViewList: %s,\n" +
		"\tincomingViewListMAX: %d,\n" +
		"\tapiClients: %s,\n" +
		"\tapiClientsMAX: %d,\n" +
		"\tmembershipController: %s,\n" +
		"\tgossiper: %s,\n" +
		"\tstate: %v,\n" +
		"}"
	return fmt.Sprintf(reprFormat,
		centralController.trustedIdentitiesPath,
		centralController.bootstrapper,
		centralController.apiAddr,
		centralController.p2pAddr,
		centralController.apiListener,
		centralController.p2pListener,
		centralController.viewList,
		centralController.viewListCap,
		centralController.awaitingRemovalViewList,
		centralController.incomingViewList,
		centralController.incomingViewListMAX,
		centralController.apiClients,
		centralController.apiClientsMAX,
		centralController.membershipController,
		centralController.gossiper,
		centralController.state,
	)
}

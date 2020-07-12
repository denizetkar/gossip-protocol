package main

import (
	"datastruct/set"
	"errors"
	"fmt"
	"math"
	"time"
)

// MedianCounterConfig holds the configuration for the maximum counter
// value of a gossip item to be considered in state B and likewise in
// state C. The paper authors suggests O(loglog(n)) for both.
type MedianCounterConfig struct {
	bMax uint8
	cMax uint8
}

// GossiperPullPeersType is the type of variable stored in
// Gossiper::pullPeers.
type GossiperPullPeersType Peer

// Gossiper is the struct for the goroutine that will be exclusively responsible for handling all p2p Gossip
// calls. It will both share items by Gossiping itself and it will also respond to the GossipPullRequest calls.
type Gossiper struct {
	// cacheSize is the maximum number of Gossip items to gossip at any time.
	cacheSize uint16
	// degree is the number of peers to gossip with per round.
	degree uint8
	// maxTTL is the maximum number of hops to propagate any Gossip item.
	maxTTL uint8
	// roundPeriod is the time duration between each membership round.
	roundPeriod time.Duration
	// mcConfig is the configuration for the "median-counter algorithm".
	mcConfig MedianCounterConfig
	// gossipList is going to contain all hot topics to propagate. Hence it is of size 'cache_size'.
	gossipList map[GossipItem]*GossipItemInfoGossiper
	// oldGossipList is going to contain all outdated gossips. If an incoming gossip is
	// in this map then it will be ignored and not propagated any further.
	oldGossipList map[GossipItem]*GossipItemInfoGossiper
	// apiClientsToNotify is a map of API clients to be notified upon receiving
	// a gossip item with a 'data type' that interests the client.
	apiClientsToNotify map[APIClient]*APIClientInfoGossiper
	// validationMap is a map from message ID's of client notifications to gossip item.
	validationMap   map[uint16]GossipItem
	nextAvailableID uint16
	// pullPeers are peers to whom the Gossiper sent a gossip pull request and
	// is waiting for a pull reply. Any gossip pull reply from a peer outside of
	// this set will be ignored!
	pullPeers set.Set
	// MsgInQueue is the incoming message queue for
	// the Gossiper goroutine.
	MsgInQueue chan InternalMessage
	// MsgOutQueue is the outgoing message queue from
	// the Gossiper goroutine to the Central controller.
	MsgOutQueue chan InternalMessage
}

// NewGossiper is the constructor function for the Gossiper struct.
func NewGossiper(cacheSize uint16, degree, maxTTL uint8, roundPeriod time.Duration, maxPeers float64,
	inQ, outQ chan InternalMessage,
) (*Gossiper, error) {
	loglogN := uint8(math.Ceil(math.Log2(math.Log2(maxPeers))))
	return &Gossiper{
		cacheSize:          cacheSize,
		degree:             degree,
		maxTTL:             maxTTL,
		roundPeriod:        roundPeriod,
		mcConfig:           MedianCounterConfig{bMax: loglogN, cMax: loglogN},
		gossipList:         map[GossipItem]*GossipItemInfoGossiper{},
		oldGossipList:      map[GossipItem]*GossipItemInfoGossiper{},
		apiClientsToNotify: map[APIClient]*APIClientInfoGossiper{},
		validationMap:      map[uint16]GossipItem{},
		nextAvailableID:    0,
		pullPeers:          set.New(),
		MsgInQueue:         inQ,
		MsgOutQueue:        outQ,
	}, nil
}

// recover method tries to catch a panic in controllerRoutine if it exists, then
// informs the Central controller about the crash. If there is no panic, then
// informs the Central controller about the graceful closure.
func (gossiper *Gossiper) recover() {
	var err error
	if r := recover(); r != nil {
		// find out exactly what the error was and set err
		switch x := r.(type) {
		case string:
			err = errors.New(x)
		case error:
			err = x
		default:
			err = errors.New("Unknown panic in Gossiper")
		}

		// send GossiperCrashedMSG to the Central controller!
		gossiper.MsgOutQueue <- InternalMessage{Type: GossiperCrashedMSG, Payload: err}
	} else {
		// send GossiperClosedMSG to the Central controller!
		gossiper.MsgOutQueue <- InternalMessage{Type: GossiperClosedMSG, Payload: void{}}
	}
}

func (gossiper *Gossiper) controllerRoutine() {
	defer gossiper.recover()
	// TODO: fill here
}

// RunControllerGoroutine runs the Gossiper goroutine.
func (gossiper *Gossiper) RunControllerGoroutine() {
	go gossiper.controllerRoutine()
}

func (gossiper *Gossiper) String() string {
	reprFormat := "*Gossiper{\n" +
		"\tcacheSize: %d,\n" +
		"\tdegree: %d,\n" +
		"\tmaxTTL: %d,\n" +
		"\troundPeriod: %s,\n" +
		"\tgossipList: %s,\n" +
		"\toldGossipList: %s,\n" +
		"\tapiClientsToNotify: %s,\n" +
		"\tvalidationMap: %s,\n" +
		"\tnextAvailableID: %d,\n" +
		"\tpullPeers: %s,\n" +
		"}"
	return fmt.Sprintf(reprFormat,
		gossiper.cacheSize,
		gossiper.degree,
		gossiper.maxTTL,
		gossiper.roundPeriod,
		gossiper.gossipList,
		gossiper.oldGossipList,
		gossiper.apiClientsToNotify,
		gossiper.validationMap,
		gossiper.nextAvailableID,
		gossiper.pullPeers,
	)
}

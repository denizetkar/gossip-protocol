package main

import "fmt"

// Gossiper is the struct for the goroutine that will be exclusively responsible for handling all p2p Gossip
// calls. It will both share items by Gossiping itself and it will also respond to the GossipPullRequest calls.
type Gossiper struct {
	// cacheSize is the maximum number of Gossip items to gossip at any time.
	cacheSize uint16
	// degree is the number of peers to gossip with per round.
	degree uint8
	// maxTTL is the maximum number of hops to propagate any Gossip item.
	maxTTL uint8
	// gossipList is going to contain all hot topic to propagate. Hence it is of size 'cache_size'.
	gossipList map[RawGossipItem]*GossipItemInfoGossiper
	// apiClientsToNotify is a map of API clients to be notified upon receiving
	// a gossip item with a 'data type' that interests the client.
	apiClientsToNotify map[APIClient]*APIClientInfoGossiper
	// MsgInQueue is the incoming message queue for
	// the Gossiper goroutine.
	MsgInQueue chan InternalMessage
	// MsgOutQueue is the outgoing message queue from
	// the Gossiper goroutine to the Central controller.
	MsgOutQueue chan InternalMessage
}

// NewGossiper is the constructor function for the Gossiper struct.
func NewGossiper(cacheSize uint16, degree, maxTTL uint8, inQ, outQ chan InternalMessage) (*Gossiper, error) {
	return &Gossiper{
		cacheSize:          cacheSize,
		degree:             degree,
		maxTTL:             maxTTL,
		gossipList:         map[RawGossipItem]*GossipItemInfoGossiper{},
		apiClientsToNotify: map[APIClient]*APIClientInfoGossiper{},
		MsgInQueue:         inQ,
		MsgOutQueue:        outQ,
	}, nil
}

func (gossiper *Gossiper) gossiperRoutine() {
	// TODO: fill here
}

// RunGossiperGoroutine runs the Gossiper goroutine.
func (gossiper *Gossiper) RunGossiperGoroutine() {
	go gossiper.gossiperRoutine()
}

func (gossiper *Gossiper) String() string {
	reprFormat := "*Gossiper{\n" +
		"\tcacheSize: %d,\n" +
		"\tdegree: %d,\n" +
		"\tmaxTTL: %d,\n" +
		"\tgossipList: %s,\n" +
		"\tapiClientsToNotify: %s,\n" +
		"}"
	return fmt.Sprintf(reprFormat,
		gossiper.cacheSize,
		gossiper.degree,
		gossiper.maxTTL,
		gossiper.gossipList,
		gossiper.apiClientsToNotify,
	)
}

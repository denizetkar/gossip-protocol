package main

// GossipItemDataType is the 16-bit unsigned integer
// that specifies the 'data type' of the gossip item as
// described in the specifications.pdf .
type GossipItemDataType uint16

// GossipItem holds the Gossip item coming from
// a "GOSSIP ANNOUCE" api call.
type GossipItem struct {
	DataType GossipItemDataType
	// Data has to be of type 'string' instead of '[]byte'
	// so that GossipItem struct is hashable for use in maps.
	Data string
}

// MedianCounterState is the type for states A, B, C and D as
// described by the "median-counter algorithm".
type MedianCounterState uint8

// A gossip item with state A cannot exist!
const (
	// MedianCounterStateB is the state B.
	MedianCounterStateB MedianCounterState = iota
	// MedianCounterStateC is the state C.
	MedianCounterStateC
	// MedianCounterStateD is the state D.
	MedianCounterStateD
)

// GossipItemState is the struct for holding the counter,
// the threshold for state B log(log(n)), the threshold for
// state C log(log(n)) and the maximum allowed time to live
// as described by the "median-counter algorithm":
// https://zoo.cs.yale.edu/classes/cs426/2012/bib/karp00randomized.pdf
type GossipItemState struct {
	state      MedianCounterState
	counter    uint8
	ttl        uint8
	medianRule int8
}

// GossipItemInfoGossiper contains the current state of the corresponding
// GossipItem and the list of peers to gossip this item. The
// 'peerList' is going to be a random subset of the current view list.
// This struct is meant to be used as a value in a
// map[GossipItem]*GossipItemInfoGossiper by the Gossiper.
type GossipItemInfoGossiper struct {
	state    GossipItemState
	peerList []Peer
}

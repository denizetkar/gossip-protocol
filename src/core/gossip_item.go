package core

// GossipItemDataType is the 16-bit unsigned integer
// that specifies the 'data type' of the gossip item as
// described in the specifications.pdf .
type GossipItemDataType uint16

const (
	// GossipAnnounce is the enumeration of 'GOSSIP ANNOUNCE' api message
	GossipAnnounce GossipItemDataType = iota + 500
	// GossipNotify is the enumeration of 'GOSSIP NOTIFY' api message
	GossipNotify
	// GossipNotification is the enumeration of 'GOSSIP NOTIFICATION' api message
	GossipNotification
	// GossipValidation is the enumeration of 'GOSSIP VALIDATION' api message
	GossipValidation
)

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
	medianRule int
}

// GossipItemInfoGossiper contains the current state of the corresponding
// GossipItem and the list of peers to gossip this item. The
// 'peerList' is going to be a random subset of the current view list.
// This struct is meant to be used as a value in a
// map[GossipItem]*GossipItemInfoGossiper by the Gossiper.
type GossipItemInfoGossiper struct {
	s        GossipItemState
	peerList []Peer
}

// Cmp compares 2 GossipItemState's by essentially checking if the 'state'
// of 'ls' has higher order than 'rs', if it is then returns 1. If it has,
// a lower order then returns -1. Otherwise, compares the 'counter' of both
// and similarly, if 'ls' has a higher counter then returns 1. If it is
// smaller then returns -1. Otherwise, returns 0.
//
// Note that MedianCounterState's are defined in increasing order.
func (ls *GossipItemState) Cmp(rs *GossipItemState) int {
	lsVal := (uint16(ls.state) << 8) | uint16(ls.counter)
	rsVal := (uint16(rs.state) << 8) | uint16(rs.counter)
	if lsVal > rsVal {
		return 1
	} else if lsVal < rsVal {
		return -1
	}
	return 0
}

// UpdateItemInfo is the method for updating state of a gossip item
// with the state of an incoming gossip item, as described in the
// "median-counter algorithm".
func (info *GossipItemInfoGossiper) UpdateItemInfo(newInfo *GossipItemInfoGossiper) {
	switch info.s.state {
	case MedianCounterStateB:
		switch newInfo.s.state {
		case MedianCounterStateB:
			if newInfo.s.counter >= info.s.counter {
				info.s.medianRule++
			} else {
				info.s.medianRule--
			}
		case MedianCounterStateC:
			info.s.state = MedianCounterStateC
			info.s.counter = 0
			info.s.medianRule = 0
		}
	case MedianCounterStateC:
		// We don't need to update in this case.
	}
}

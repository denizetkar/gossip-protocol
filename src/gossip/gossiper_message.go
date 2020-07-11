package main

// RandomPeerListRequestMSGPayload is the payload type of an InternalMessage
// with type RandomPeerListRequestMSG.
type RandomPeerListRequestMSGPayload struct {
	// Related is the gossip item for which the random peer list is requested.
	// If it is nil, then the peer list is requested for pull.
	Related *GossipItem
	// Num is the number of random peers requested. The actual number
	// of peers returned might be less.
	Num int
}

// RandomPeerListReplyMSGPayload is the payload type of an InternalMessage
// with type RandomPeerListReplyMSG.
type RandomPeerListReplyMSGPayload struct {
	// Related is the gossip item for which the random peer list is requested.
	// If it is nil, then the peer list was requested for pull.
	Related *GossipItem
	// RandomPeers is the list of random peers to gossip with.
	RandomPeers []Peer
}

// RandomPeerListReleaseMSGPayload is the payload type of an InternalMessage
// with type RandomPeerListReleaseMSG.
type RandomPeerListReleaseMSGPayload struct {
	// ReleasedPeers are the peers that are no longer being used
	// for the task that there were requested for.
	ReleasedPeers []Peer
}

// GossipAnnounceMSGPayload is the payload type of an InternalMessage
// with type GossipAnnounceMSG.
type GossipAnnounceMSGPayload struct {
	// Item is the gossip item to be announced/gossiped.
	Item *GossipItem
	// TTL is the requested time to live. If 0, then the default maxTTL will be used.
	TTL uint8
}

// GossipNotifyMSGPayload is the payload type of an InternalMessage
// with type GossipNotifyMSG.
type GossipNotifyMSGPayload struct {
	// Who is the api client to be notified for the message type specified.
	Who APIClient
	// What is the data type for which to notify the client.
	What GossipItemDataType
}

// GossipUnnofityMSGPayload is the payload type of an InternalMessage
// with type GossipUnnofityMSG.
type GossipUnnofityMSGPayload APIClient

// GossipPushMSGPayload is the payload type of an InternalMessage
// with type GossipPushMSG.
type GossipPushMSGPayload struct {
	// Item is the gossip item to be pushed.
	Item    *GossipItem
	State   MedianCounterState
	Counter uint8
	// To is the peer to push the gossip item.
	To Peer
}

// GossipItemExtended is a struct for holding everything necessary
// to exchange a gossip item on the network either for push or for
// pulls.
type GossipItemExtended struct {
	// Item is the gossip item that was pushed.
	Item    *GossipItem
	State   MedianCounterState
	Counter uint8
}

// GossipIncomingPushMSGPayload is the payload type of an InternalMessage
// with type GossipIncomingPushMSG.
type GossipIncomingPushMSGPayload GossipItemExtended

// GossipPullRequestMSGPayload is the payload type of an InternalMessage
// with type GossipPullRequestMSG.
type GossipPullRequestMSGPayload Peer

// GossipIncomingPullRequestMSGPayload is the payload type of an InternalMessage
// with type GossipIncomingPullRequestMSG.
type GossipIncomingPullRequestMSGPayload struct {
	// From is the remote peer who sent the pull request.
	From Peer
}

// GossipPullReplyMSGPayload is the payload type of an InternalMessage
// with type GossipPullReplyMSG.
type GossipPullReplyMSGPayload struct {
	// To is the remote peer who sent the pull request.
	To Peer
	// ItemList is the list of gossip items for the pull reply.
	ItemList []*GossipItemExtended
}

// GossipIncomingPullReplyMSGPayload is the payload type of an InternalMessage
// with type GossipIncomingPullReplyMSG.
type GossipIncomingPullReplyMSGPayload struct {
	// From is the remote peer who replied to the pull request.
	From Peer
	// ItemList is the list of gossip items for the pull reply.
	ItemList []*GossipItemExtended
}

// GossiperCrashedMSGPayload is the payload type of an InternalMessage
// with type GossiperCrashedMSG.
type GossiperCrashedMSGPayload error

// GossiperCloseMSGPayload is the payload type of an InternalMessage
// with type GossiperCloseMSG.
type GossiperCloseMSGPayload void

// GossiperClosedMSGPayload is the payload type of an InternalMessage
// with type GossiperClosedMSG.
type GossiperClosedMSGPayload void

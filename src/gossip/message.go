package main

// InternalMessageType is a 16-bit unsigned integer specifying type
// of an internal message.
type InternalMessageType uint16

const (
	// PeerAddMSG is a command from the Membership controller to
	// the Central controller for adding a peer and its connection.
	PeerAddMSG InternalMessageType = iota + 1000
	// PeerRemoveMSG is a command from the Membership controller to
	// the Central controller for removing a peer and its connection.
	PeerRemoveMSG
	// PeerDisconnectedMSG is a notification from the Central controller to
	// the Membership controller that the peer is abruptly disconnected so that
	// it can also remove it from its viewList.
	PeerDisconnectedMSG
	// ProbePeerRequestMSG is a command from the Membership controller to
	// the Central controller for probing a peer.
	ProbePeerRequestMSG
	// ProbePeerReplyMSG is a reply from the Central controller to
	// the Membership controller for probing a peer.
	ProbePeerReplyMSG
	// MembershipPushRequestMSG is a command from the Membership controller to
	// the Central controller for sending a push request to a peer.
	MembershipPushRequestMSG
	// MembershipIncomingPushRequestMSG is a notification from the Central controller to
	// the Membership controller for an incoming MembershipPushRequestMSGPayload from the peer specified.
	MembershipIncomingPushRequestMSG
	// MembershipPullRequestMSG is a command from the Membership controller to
	// the Central controller for sending a pull request to a peer.
	MembershipPullRequestMSG
	// MembershipIncomingPullRequestMSG is a notification from the Central controller to
	// the Membership controller for an incoming MembershipPullRequest from the peer specified.
	MembershipIncomingPullRequestMSG
	// MembershipPullReplyMSG is a reply from the Membership controller to
	// the Central controller for the incoming MembershipPullRequest from the peer specified.
	MembershipPullReplyMSG
	// MembershipIncomingPullReplyMSG is a notification from the Central controller to
	// the Membership controller after receiving MembershipPullReplyMSGPayload from the peer.
	MembershipIncomingPullReplyMSG
	// MembershipCrashedMSG is a notification from the Membership controller to
	// the Central controller that it crashed.
	MembershipCrashedMSG
	// MembershipCloseMSG is a command from the Central controller to the
	// Membership controller to close gracefully as soon as possible.
	MembershipCloseMSG
	// MembershipClosedMSG is a notification from the Membership controller to the
	// Central controller for closing gracefully as requested.
	MembershipClosedMSG

	// RandomPeerListRequestMSG is a command from the Gossiper to
	// the Central controller for requesting a random subset of peers.
	RandomPeerListRequestMSG = iota + 2000 - (MembershipIncomingPullReplyMSG - 999)
	// RandomPeerListReplyMSG is a reply from the Central controller to
	// the Gossiper for requesting a random subset of peers. Usage
	// counters of the corresponding peers are incremented.
	RandomPeerListReplyMSG
	// RandomPeerListReleaseMSG is a notification from the Gossiper to
	// the Central controller for decreasing the usage counters of the
	// corresponding peers.
	RandomPeerListReleaseMSG
	// GossipAnnounceMSG is a command from the Central controller to
	// the Gossiper to spread the information provided in the payload.
	GossipAnnounceMSG
	// GossipNotifyMSG is a command from the Central controller to
	// the Gossiper to notify the corresponding API client upon
	// receiving a gossip of given type.
	GossipNotifyMSG
	// GossipUnnofityMSG is a command from the Central controller to
	// the Gossiper to entirely remove the given API client.
	GossipUnnofityMSG
	// GossipPushMSG is a command from the Gossiper to the Central
	// controller to send the GossipPushMSGPayload to the peer specified.
	GossipPushMSG
	// GossipIncomingPushMSG is a notification from the Central controller
	// to the Gossiper for the arrival of a GossipPushMSGPayload.
	GossipIncomingPushMSG
	// GossipPullRequestMSG is a command from the Gossiper to the Central
	// controller to send the GossipPullRequest to the peer specified.
	GossipPullRequestMSG
	// GossipIncomingPullRequestMSG is a notification from the Central controller
	// to the Gossiper for an incoming GossipPullRequest from the peer specified.
	GossipIncomingPullRequestMSG
	// GossipPullReplyMSG is a reply from the Gossiper to the Central
	// controller for the incoming GossipPullRequest from the peer specified.
	GossipPullReplyMSG
	// GossipIncomingPullReplyMSG is a notification from the Central controller to the
	// Gossiper after receiving GossipPullReply from the peer.
	GossipIncomingPullReplyMSG
	// GossiperCrashedMSG is a notification from the Gossiper to
	// the Central controller that it crashed.
	GossiperCrashedMSG
	// GossiperCloseMSG is a command from the Central controller to the
	// Gossiper to close gracefully as soon as possible.
	GossiperCloseMSG
	// GossiperClosedMSG is a notification from the Gossiper to the
	// Central controller for closing gracefully as requested.
	GossiperClosedMSG

	// TODO: Add message types in intervals of 1000 for other submodules such as:
	//       APIListener, P2PListener, APIEndpoint, P2PEndpoint.
)

// AnyMessage is the type of any internal message between goroutines
// of the Gossip module. The contents of these messages MUST NOT be
// shared between the communicating goroutines. So, make an explicit
// copy of the message content if necessary!
type AnyMessage interface{}

// void is the payload type for all internal message types which
// do not need any payload.
type void struct{}

// InternalMessage is a generic message struct for sending
// any information between the goroutines of the Gossip module.
type InternalMessage struct {
	Type    InternalMessageType
	Payload AnyMessage
}

// TODO: Create payload types for all internal message types as necessary.

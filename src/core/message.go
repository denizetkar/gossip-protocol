package core

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
)

const (
	// RandomPeerListRequestMSG is a command from the Gossiper to
	// the Central controller for requesting a random subset of peers.
	RandomPeerListRequestMSG InternalMessageType = iota + 2000
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
	// the Gossiper to register the corresponding API client for notifications
	// about the given gossip data types.
	GossipNotifyMSG
	// GossipUnnofityMSG is a command from the Central controller to
	// the Gossiper to entirely remove the given API client.
	GossipUnnofityMSG
	// GossipNotificationMSG is a command from the Gossiper to the
	// Central controller to notify the corresponding API client upon
	// receiving a gossip of given data type.
	GossipNotificationMSG
	// GossipValidationMSG is a notification from the Central controller to
	// the Gossiper to for an incoming GOSSIP VALIDATION api call.
	GossipValidationMSG
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
)

const (
	// OutgoingP2PCreatedMSG is a reply from a goroutine that created
	// an outgoing p2p endpoint to the Central controller.
	OutgoingP2PCreatedMSG InternalMessageType = iota + 3000
	// CentralProbePeerReplyMSG is a reply from a peer prober to
	// the Central controller for probing a peer.
	CentralProbePeerReplyMSG
	// CentralCrashMSG is a command from a closing timer to the
	// Central controller to panic and crash.
	CentralCrashMSG
	// CentralCloseMSG is a command from the User to the Central controller to close.
	CentralCloseMSG
)

const (
	// APIListenerCrashedMSG is a notification from the api listener to
	// the Central controller that it crashed.
	APIListenerCrashedMSG InternalMessageType = iota + 4000
	// APIListenerClosedMSG is a notification from the api listener to
	// the Central controller for closing gracefully as requested.
	APIListenerClosedMSG
	// APIEndpointCreatedMSG is a notification from the api listener to the
	// Central controller that it created an api endpoint.
	APIEndpointCreatedMSG
)

const (
	// APIEndpointCrashedMSG is a notification from the api endpoint to
	// the Central controller that it crashed.
	APIEndpointCrashedMSG InternalMessageType = iota + 5000
	// APIEndpointClosedMSG is a notification from the api endpoint to
	// the Central controller for closing gracefully as requested.
	APIEndpointClosedMSG
)

const (
	// P2PListenerCrashedMSG is a notification from the p2p listener to
	// the Central controller that it crashed.
	P2PListenerCrashedMSG InternalMessageType = iota + 6000
	// P2PListenerClosedMSG is a notification from the p2p listener to
	// the Central controller for closing gracefully as requested.
	P2PListenerClosedMSG
	// IncomingP2PCreatedMSG is a notification from the p2p listener to the
	// Central controller that it created an incoming p2p endpoint.
	IncomingP2PCreatedMSG
)

const (
	// P2PEndpointCrashedMSG is a notification from the p2p endpoint to
	// the Central controller that it crashed.
	P2PEndpointCrashedMSG InternalMessageType = iota + 7000
	// P2PEndpointClosedMSG is a notification from the p2p endpoint to
	// the Central controller for closing gracefully as requested.
	P2PEndpointClosedMSG
)

// TODO: Add more message types for the following submodules:
//       APIListener, P2PListener, APIEndpoint, P2PEndpoint.

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

// CloseError is a special type of error used by a message handler function,
// in order to signal to a goroutine that it should close gracefully.
type CloseError struct {
	s string
}

func (err *CloseError) Error() string {
	return err.s
}

package core

// APIMessageType is the 16-bit unsigned integer that
// specifies the 'message type' of an api message as
// described in the specifications.pdf .
type APIMessageType uint16

const (
	// GossipAnnounce is the enumeration of 'GOSSIP ANNOUNCE' api message
	GossipAnnounce APIMessageType = iota + 500
	// GossipNotify is the enumeration of 'GOSSIP NOTIFY' api message
	GossipNotify
	// GossipNotification is the enumeration of 'GOSSIP NOTIFICATION' api message
	GossipNotification
	// GossipValidation is the enumeration of 'GOSSIP VALIDATION' api message
	GossipValidation
)

// APIListenerCrashedMSGPayload is the payload type of an InternalMessage
// with type APIListenerCrashedMSG.
type APIListenerCrashedMSGPayload error

// APIListenerClosedMSGPayload is the payload type of an InternalMessage
// with type APIListenerClosedMSG.
type APIListenerClosedMSGPayload void

// APIEndpointCreatedMSGPayload is the payload type of an InternalMessage
// with type APIEndpointCreatedMSG.
type APIEndpointCreatedMSGPayload *APIEndpoint

// APIEndpointCrashedMSGPayload is the payload type of an InternalMessage
// with type APIEndpointCrashedMSG.
type APIEndpointCrashedMSGPayload struct {
	endp     *APIEndpoint
	isReader bool
	err      error
}

// APIEndpointClosedMSGPayload is the payload type of an InternalMessage
// with type APIEndpointClosedMSG.
type APIEndpointClosedMSGPayload struct {
	endp     *APIEndpoint
	isReader bool
}

// APIEndpointCloseMSGPayload is the payload type of an InternalMessage
// with type APIEndpointClosedMSG.
type APIEndpointCloseMSGPayload void

// APIMessageHeader is the header from the message that is received from an API client
type APIMessageHeader struct {
	Size        uint16
	MessageType APIMessageType
}

// APIAnnounceMSGPayload is the payload type of an InternalMessage
// with type APIAnnounceMSG.
type APIAnnounceMSGPayload GossipAnnounceMSGPayload

// APINotifyMSGPayload is the payload type of an InternalMessage
// with type APINotifyMSG.
type APINotifyMSGPayload GossipNotifyMSGPayload

// APINotificationMSGPayload is the payload type of an InternalMessage
// with type APINotificationMSG.
type APINotificationMSGPayload GossipNotificationMSGPayload

// APIValidationMSGPayload is the payload type of an InternalMessage
// with type APIValidationMSG.
type APIValidationMSGPayload GossipValidationMSGPayload

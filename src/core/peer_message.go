package core

// P2PListenerCrashedMSGPayload is the payload type of an InternalMessage
// with type P2PListenerCrashedMSG.
type P2PListenerCrashedMSGPayload error

// P2PListenerClosedMSGPayload is the payload type of an InternalMessage
// with type P2PListenerClosedMSG.
type P2PListenerClosedMSGPayload void

// P2PEndpointCrashedMSGPayload is the payload type of an InternalMessage
// with type P2PEndpointCrashedMSG.
type P2PEndpointCrashedMSGPayload struct {
	endp     *P2PEndpoint
	isReader bool
	err      error
}

// P2PEndpointClosedMSGPayload is the payload type of an InternalMessage
// with type P2PEndpointClosedMSG.
type P2PEndpointClosedMSGPayload struct {
	endp     *P2PEndpoint
	isReader bool
}

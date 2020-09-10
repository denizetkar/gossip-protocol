package core

// APIListenerCrashedMSGPayload is the payload type of an InternalMessage
// with type APIListenerCrashedMSG.
type APIListenerCrashedMSGPayload error

// APIListenerClosedMSGPayload is the payload type of an InternalMessage
// with type APIListenerClosedMSG.
type APIListenerClosedMSGPayload void

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

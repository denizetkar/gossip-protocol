package main

// OutgoingP2PCreatedMSGPayload is the payload type of an InternalMessage
// with type OutgoingP2PCreatedMSG.
type OutgoingP2PCreatedMSGPayload *P2PEndpoint

// CentralProbePeerReplyMSGPayload is the payload type of an InternalMessage
// with type CentralProbePeerReplyMSG.
type CentralProbePeerReplyMSGPayload struct {
	Probed      Peer
	ProbeResult bool
}

// CentralCrashMSGPayload is the payload type of an InternalMessage
// with type CentralCrashMSG.
type CentralCrashMSGPayload error

// CentralCloseMSGPayload is the payload type of an InternalMessage
// with type CentralCloseMSG.
type CentralCloseMSGPayload void

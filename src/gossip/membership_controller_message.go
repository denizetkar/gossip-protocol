package main

import (
	"fmt"
	"math/big"
	mrand "math/rand"
	"time"

	"golang.org/x/crypto/scrypt"
)

// PeerAddMSGPayload is the payload type of an InternalMessage with
// type PeerAddMSG.
type PeerAddMSGPayload Peer

// PeerRemoveMSGPayload is the payload type of an InternalMessage with
// type PeerRemoveMSG.
type PeerRemoveMSGPayload Peer

// PeerDisconnectedMSGPayload is the payload type of an InternalMessage
// with type PeerDisconnectedMSG.
type PeerDisconnectedMSGPayload Peer

// ProbePeerRequestMSGPayload is the payload type of an InternalMessage
// with type ProbePeerRequestMSG.
type ProbePeerRequestMSGPayload Peer

// ProbePeerReplyMSGPayload is the payload type of an InternalMessage
// with type ProbePeerReplyMSG.
type ProbePeerReplyMSGPayload struct {
	Probed      Peer
	ProbeResult bool
}

// MembershipPushRequestMSGPayload is the payload type of an InternalMessage
// with type MembershipPushRequestMSG.
type MembershipPushRequestMSGPayload struct {
	// From is the requesting peer (with p2p listen address).
	From Peer
	// To is the requested peer (with p2p listen address).
	To Peer
	// When is the time of creation for this pull request (UTC).
	When time.Time
	// Nonce is the number used for the Proof of Work.
	Nonce uint64
}

// MembershipIncomingPushRequestMSGPayload is the payload type of an InternalMessage
// with type MembershipIncomingPushRequestMSG.
type MembershipIncomingPushRequestMSGPayload MembershipPushRequestMSGPayload

// MembershipPullRequestMSGPayload is the payload type of an InternalMessage
// with type MembershipPullRequestMSG.
type MembershipPullRequestMSGPayload Peer

// MembershipIncomingPullRequestMSGPayload is the payload type of an InternalMessage
// with type MembershipIncomingPullRequestMSG.
type MembershipIncomingPullRequestMSGPayload struct {
	// From is the remote peer who sent the pull request.
	From Peer
}

// MembershipPullReplyMSGPayload is the payload type of an InternalMessage
// with type MembershipPullReplyMSG.
type MembershipPullReplyMSGPayload struct {
	// To is the remote peer who sent the pull request.
	To Peer
	// ViewList is the list of peers in the viewList of the replying peer.
	ViewList []Peer
}

// MembershipIncomingPullReplyMSGPayload is the payload type of an InternalMessage
// with type MembershipIncomingPullReplyMSG.
type MembershipIncomingPullReplyMSGPayload struct {
	// From is the remote peer who replied to the pull request.
	From Peer
	// ViewList is the list of peers in the viewList of the replying peer.
	ViewList []Peer
}

// MembershipCrashedMSGPayload is the payload type of an InternalMessage
// with type MembershipCrashedMSG.
type MembershipCrashedMSGPayload error

// MembershipCloseMSGPayload is the payload type of an InternalMessage
// with type MembershipCloseMSG.
type MembershipCloseMSGPayload void

// MembershipClosedMSGPayload is the payload type of an InternalMessage
// with type MembershipClosedMSG.
type MembershipClosedMSGPayload void

// NewMembershipPushRequestMSGPayload is the constructor function for struct type MembershipPushRequestMSGPayload.
func NewMembershipPushRequestMSGPayload(
	from, to Peer, hardness, repetition uint64,
) (*MembershipPushRequestMSGPayload, error) {
	// k = (2^256 - 1)/repetition
	k := new(big.Int).Exp(new(big.Int).SetInt64(2), new(big.Int).SetInt64(256), nil)
	k = new(big.Int).Sub(k, new(big.Int).SetInt64(1))
	k = k.Div(k, new(big.Int).SetUint64(repetition))

	pr := MembershipPushRequestMSGPayload{From: from, To: to, When: time.Now().UTC(), Nonce: mrand.Uint64()}
	for i := uint64(0); i < 2*repetition; i++ {
		pass := []byte(fmt.Sprintf("%v", pr))
		hash, err := scrypt.Key(pass, nil, 1<<hardness, 8, 1, 32)
		if err != nil {
			return nil, err
		}
		hashVal := new(big.Int).SetBytes(hash)
		if hashVal.Cmp(k) <= 0 {
			return &pr, nil
		}
		pr.Nonce++
	}
	return nil, fmt.Errorf("failed to create a valid membership pull request in time")
}

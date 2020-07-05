package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/cipher/ecb"
	"crypto/rand"
	"datastruct/set"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
)

// MinWiseIndependentPermutation is implementation of a min-wise
// independent permutation function.
type MinWiseIndependentPermutation struct {
	enc cipher.BlockMode
}

// NewMinWiseIndependentPermutation is the constructor function for struct
// type MinWiseIndependentPermutation.
func NewMinWiseIndependentPermutation() (*MinWiseIndependentPermutation, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	enc := ecb.NewECBEncrypter(block)

	return &MinWiseIndependentPermutation{enc: enc}, nil
}

// Permute method performs the actual min-wise independent permutation.
func (p *MinWiseIndependentPermutation) Permute(identity []byte) (pVal []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			// find out exactly what the error was and set err
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("Unknown panic in MinWiseIndependentPermutation::Permute")
			}
		}
	}()
	if len(identity) != 32 {
		err = fmt.Errorf("len(identity) is not 32, it is (%d)", len(identity))
		return
	}

	pVal = make([]byte, 32)
	p.enc.CryptBlocks(pVal, identity)
	return
}

// PeerSampler is a peer sampler as described in BRAHMS paper.
type PeerSampler struct {
	peer     *Peer
	peerPVal []byte
	permuter *MinWiseIndependentPermutation
}

// NewPeerSampler is the constructor function for struct type PeerSampler.
func NewPeerSampler() (*PeerSampler, error) {
	permuter, err := NewMinWiseIndependentPermutation()
	if err != nil {
		return nil, err
	}
	return &PeerSampler{peerPVal: make([]byte, 32), permuter: permuter}, nil
}

// Next is the method for introducing peer sampler with new peers.
func (peerSampler *PeerSampler) Next(peer Peer, identity []byte) error {
	// calculate the permuted value of new peer's identity
	newPeerPVal, err := peerSampler.permuter.Permute(identity)
	if err != nil {
		return err
	}
	// if we don't already have a peer, accept the new peer
	if peerSampler.peer == nil {
		peerSampler.peer = &peer
		peerSampler.peerPVal = newPeerPVal
		return nil
	}

	// if permuted identity of new peer is smaller than the existing peer, accept the new peer
	if new(big.Int).SetBytes(newPeerPVal).Cmp(new(big.Int).SetBytes(peerSampler.peerPVal)) < 0 {
		peerSampler.peer = &peer
		peerSampler.peerPVal = newPeerPVal
	}

	return nil
}

// Sample is the method for sampling the peer from the peer sampler.
func (peerSampler *PeerSampler) Sample() Peer {
	if peerSampler.peer == nil {
		return Peer{}
	}

	return *peerSampler.peer
}

// MembershipControllerViewListType is the type of variable
// stored in MembershipController::viewList.
type MembershipControllerViewListType Peer

// MembershipController is going to run async to maintain membership lists.
type MembershipController struct {
	bootstrapper string
	p2pAddr      string
	// configuration parameters
	alphaSize, betaSize, gammaSize uint16
	// viewList is the current set of peers for gossiping. It is of size O(n^0.25).
	viewList    *set.Set
	viewListCap uint16
	// sampleList is the current list of randomly sampled peers. It is of size O(n^0.5).
	sampleList []*PeerSampler
	// MsgInQueue is the incoming message queue for
	// the Membership controller goroutine.
	MsgInQueue chan InternalMessage
	// MsgOutQueue is the outgoing message queue from
	// the Membership controller goroutine to the Central controller.
	MsgOutQueue chan InternalMessage
}

// NewMembershipController is a constructor for the MembershipController class.
func NewMembershipController(
	bootstrapper, p2pAddr string, alpha, beta float64, maxPeers float64, viewListCap uint16,
	inQ, outQ chan InternalMessage,
) (*MembershipController, error) {
	membershipController := MembershipController{
		bootstrapper: bootstrapper,
		p2pAddr:      p2pAddr,
		viewList:     set.New(),
		viewListCap:  viewListCap,
		MsgInQueue:   inQ,
		MsgOutQueue:  outQ,
	}

	// Check the validity of alpha and beta parameters
	if alpha <= 0 || beta <= 0 || alpha+beta >= 1 {
		return nil, fmt.Errorf("alpha, beta and gamma parameters are invalid: %f, %f, %f", alpha, beta, 1-(alpha+beta))
	}

	membershipController.alphaSize = uint16(math.Floor(alpha * float64(viewListCap)))
	membershipController.betaSize = uint16(math.Floor(beta * float64(viewListCap)))
	membershipController.gammaSize = viewListCap - membershipController.alphaSize - membershipController.betaSize

	sampleListSize := uint32(maxPeers / float64(viewListCap*viewListCap))
	membershipController.sampleList = make([]*PeerSampler, sampleListSize)

	return &membershipController, nil
}

func (membershipController *MembershipController) controllerRoutine() {
	// TODO: fill here
}

// RunControllerGoroutine runs the membership management&control goroutine.
func (membershipController *MembershipController) RunControllerGoroutine() {
	go membershipController.controllerRoutine()
}

func (membershipController *MembershipController) String() string {
	reprFormat := "*MembershipController{\n" +
		"\tbootstrapper: %q,\n" +
		"\tp2pAddr: %q,\n" +
		"\talphaSize: %d,\n" +
		"\tbetaSize: %d,\n" +
		"\tgammaSize: %d,\n" +
		"\tviewList: %v,\n" +
		"\tviewListCap: %d,\n" +
		"\tlen(sampleList): %d,\n" +
		"}"
	return fmt.Sprintf(reprFormat,
		membershipController.bootstrapper,
		membershipController.p2pAddr,
		membershipController.alphaSize,
		membershipController.betaSize,
		membershipController.gammaSize,
		membershipController.viewList,
		membershipController.viewListCap,
		len(membershipController.sampleList),
	)
}

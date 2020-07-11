package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/cipher/ecb"
	"crypto/rand"
	"crypto/sha256"
	"datastruct/indexedmap"
	"datastruct/indexedset"
	"datastruct/set"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"time"
)

// MinWiseIndependentPermutation is implementation of a min-wise
// independent permutation function.
type MinWiseIndependentPermutation struct {
	enc cipher.BlockMode
}

// PeerSampler is a peer sampler as described in the BRAHMS paper:
// https://www.researchgate.net/publication/221343826_Brahms_Byzantine_Resilient_Random_Membership_Sampling
type PeerSampler struct {
	peer     *Peer
	peerPVal *big.Int
	permuter *MinWiseIndependentPermutation
}

// MembershipPoWConfig holds the limited push request Proof of Work configurations.
type MembershipPoWConfig struct {
	// hardness determines how long each scrypt hashing of MembershipPushRequestMSGPayload takes.
	hardness uint64
	// repetition determines how many scrypt hashing is performed to create a valid
	// MembershipPushRequestMSGPayload on average.
	repetition uint64
	// validityDuration is the amount of time a limited push request is valid
	// after its creation time.
	validityDuration time.Duration
}

// MembershipControllerViewListType is the type of variable
// stored in MembershipController::viewList.
type MembershipControllerViewListType Peer

// MembershipControllerPushRequestsType is the type of variable
// stored in MembershipController::pushRequests.
type MembershipControllerPushRequestsType Peer

// MembershipControllerPullRepliesType is the type of variable
// stored in MembershipController::pullReplies.
type MembershipControllerPullRepliesType Peer

// MembershipControllerSampleListKeyType is the key type of
// MembershipController::sampleList.
type MembershipControllerSampleListKeyType Peer

// MembershipControllerSampleListValueType is the value type of
// MembershipController::sampleList.
type MembershipControllerSampleListValueType set.Set

// MembershipControllerSampleListValueTypeType is the type of variable
// stored in the value type of MembershipController::sampleList.
type MembershipControllerSampleListValueTypeType *PeerSampler

// MembershipControllerPullPeersType is the type of variable
// stored in MembershipController::pullPeers.
type MembershipControllerPullPeersType Peer

// MembershipController is going to run async to maintain membership lists.
type MembershipController struct {
	bootstrapper string
	p2pAddr      string
	// configuration parameters
	alphaSize, betaSize, gammaSize uint16
	// pushProbability is the probability of making a push request.
	pushProbability float64
	// roundPeriod is the time duration between each membership round.
	roundPeriod time.Duration
	// powConfig is the PoW config for push requests (both incoming and outgoing).
	powConfig MembershipPoWConfig
	// viewList is the current set of peers for gossiping. It is of size O(n^0.25).
	viewList *indexedset.IndexedSet
	// viewListCap is the total capacity of viewList for Peer's.
	viewListCap uint16
	// sampleList is the current map of randomly sampled peers. It is of size O(n^0.5).
	// The map is of the form map[Peer]set[*PeerSampler] where the 'peer' inside each
	// PeerSampler must be the same as the key Peer. As each PeerSampler sample new
	// peers, their corresponding key peer must also change to the new one.
	sampleList *indexedmap.IndexedMap
	// sampleListRemainingCap is the remaining capacity of sampleList for new PeerSampler's.
	sampleListRemainingCap uint32
	// pushRequests is a set of peers who sent us a valid push request
	// since the end of previous membership round.
	pushRequests set.Set
	// pullReplies is a set of peers that were sent to us as a pull reply.
	pullReplies set.Set
	// pullPeers are peers to whom the Membership controller sent a membership
	// pull request and is waiting for a pull reply. Any membership pull reply
	// from a peer outside of this set will be ignored!
	pullPeers set.Set
	// MsgInQueue is the incoming message queue for
	// the Membership controller goroutine.
	MsgInQueue chan InternalMessage
	// MsgOutQueue is the outgoing message queue from
	// the Membership controller goroutine to the Central controller.
	MsgOutQueue chan InternalMessage
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
func (p *MinWiseIndependentPermutation) Permute(peer *Peer) (pVal []byte, err error) {
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
	// identity is not the same as the one described in the specifications.pdf .
	identity := sha256.Sum256([]byte(peer.Addr))

	pVal = make([]byte, 32)
	p.enc.CryptBlocks(pVal, identity[:])
	return
}

// NewPeerSampler is the constructor function for struct type PeerSampler.
func NewPeerSampler() (*PeerSampler, error) {
	permuter, err := NewMinWiseIndependentPermutation()
	if err != nil {
		return nil, err
	}
	return &PeerSampler{permuter: permuter}, nil
}

// Next is the method for introducing peer sampler with new peers.
func (peerSampler *PeerSampler) Next(peer *Peer) error {
	// calculate the permuted value of new peer's identity
	newPeerPValBytes, err := peerSampler.permuter.Permute(peer)
	if err != nil {
		return err
	}
	newPeerPVal := new(big.Int).SetBytes(newPeerPValBytes)

	// if we don't already have a peer, accept the new peer
	if peerSampler.peer == nil {
		peerSampler.peer = peer
		peerSampler.peerPVal = newPeerPVal
		return nil
	}

	// if permuted identity of new peer is smaller than the existing peer, accept the new peer
	if newPeerPVal.Cmp(peerSampler.peerPVal) < 0 {
		peerSampler.peer = peer
		peerSampler.peerPVal = newPeerPVal
		return nil
	}

	return fmt.Errorf("new peer does not have a smaller hash value")
}

// Sample is the method for sampling the peer from the peer sampler.
func (peerSampler *PeerSampler) Sample() Peer {
	if peerSampler.peer == nil {
		return Peer{}
	}

	return *peerSampler.peer
}

// NewMembershipController is a constructor for the MembershipController class.
func NewMembershipController(
	bootstrapper, p2pAddr string, alpha, beta float64, roundDuration time.Duration, maxPeers float64, viewListCap uint16,
	inQ, outQ chan InternalMessage,
) (*MembershipController, error) {
	// Since the following parameters are critical for the correct operation of the
	// network, they are embedded into the source code instead of the config file.
	// This way, only "power users", who know what they are doing, can modify it!
	powHardness := uint64(4)
	powRepetition := uint64(512)
	powValidityDuration := roundDuration

	membershipController := MembershipController{
		bootstrapper:    bootstrapper,
		p2pAddr:         p2pAddr,
		pushProbability: 0.0,
		roundPeriod:     roundDuration,
		powConfig: MembershipPoWConfig{
			hardness:         powHardness,
			repetition:       powRepetition,
			validityDuration: powValidityDuration,
		},
		viewList:               indexedset.New(),
		viewListCap:            viewListCap,
		sampleList:             indexedmap.New(),
		sampleListRemainingCap: uint32(maxPeers / float64(viewListCap*viewListCap)),
		pushRequests:           set.New(),
		pullReplies:            set.New(),
		pullPeers:              set.New(),
		MsgInQueue:             inQ,
		MsgOutQueue:            outQ,
	}

	// Check the validity of alpha and beta parameters
	if alpha <= 0 || beta <= 0 || alpha+beta >= 1 {
		return nil, fmt.Errorf("alpha, beta and gamma parameters are invalid: %f, %f, %f", alpha, beta, 1-(alpha+beta))
	}

	membershipController.alphaSize = uint16(math.Floor(alpha * float64(viewListCap)))
	membershipController.betaSize = uint16(math.Floor(beta * float64(viewListCap)))
	membershipController.gammaSize = viewListCap - membershipController.alphaSize - membershipController.betaSize

	return &membershipController, nil
}

// recover method tries to catch a panic in controllerRoutine if it exists, then
// informs the Central controller about the crash.
func (membershipController *MembershipController) recover() {
	var err error
	if r := recover(); r != nil {
		// find out exactly what the error was and set err
		switch x := r.(type) {
		case string:
			err = errors.New(x)
		case error:
			err = x
		default:
			err = errors.New("Unknown panic in MembershipController")
		}

		// send MembershipCrashedMSG to the Central controller!
		membershipController.MsgOutQueue <- InternalMessage{Type: MembershipCrashedMSG, Payload: err}
	} else {
		// send MembershipClosedMSG to the Central controller!
		membershipController.MsgOutQueue <- InternalMessage{Type: MembershipClosedMSG, Payload: void{}}
	}
}

// replaceViewList is the method to use when replacing the 'viewList' with a new one.
// It not only replaces but also informs the Central controller. So, DO NOT MAKE
// CHANGES TO THE 'viewList' ELSEWHERE!
//
// The 'newViewList' argument is possibly modified.
func (membershipController *MembershipController) replaceViewList(newViewList *indexedset.IndexedSet) {
	toBeRemoved := indexedset.New()
	toBeAdded := newViewList
	viewList := membershipController.viewList

	for peer := range viewList.Iterate() {
		if !newViewList.IsMember(peer) {
			// if the peer is not in the intersection, then we need to remove it!
			toBeRemoved.Add(peer)
		} else {
			// if the peer is in the intersection, then we don't need to add it!
			toBeAdded.Remove(peer)
		}
	}
	for peer := range toBeRemoved.Iterate() {
		// TODO: send PeerRemoveMSG message to the Central controller!

		viewList.Remove(peer)
	}
	for peer := range toBeAdded.Iterate() {
		// TODO: send PeerAddMSG message to the Central controller!

		viewList.Add(peer)
	}
}

// membershipRound is a method for executing 1 round of membership exchange.
// Normally it is executed only periodically. However, if bootstrapping for
// the first time, the round is also executed.
func (membershipController *MembershipController) membershipRound() {
	// TODO: fill here
	pushProbability := membershipController.pushProbability

	membershipController.pushProbability = 1.0 - (1.0-pushProbability)*0.9
}

// bootstrap checks if the 'viewList' is empty and if so then refills it.
func (membershipController *MembershipController) bootstrap() {
	if membershipController.viewList.Len() == 0 {
		newViewList := indexedset.New()
		newViewList.Add(Peer{Addr: membershipController.bootstrapper})
		membershipController.replaceViewList(newViewList)
		membershipController.pushProbability = 0.0
		// force execute a round of membership exchange
		membershipController.membershipRound()
	}
}

func (membershipController *MembershipController) controllerRoutine() {
	defer membershipController.recover()
	membershipController.bootstrap()
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
		"\tpushProbability: %f,\n" +
		"\troundPeriod: %s,\n" +
		"\tpowConfig: %v,\n" +
		"\tviewList: %v,\n" +
		"\tviewListCap: %d,\n" +
		"\tsampleList: %v,\n" +
		"\tsampleListRemainingCap: %d,\n" +
		"\tpushRequests: %s,\n" +
		"\tpullReplies: %s,\n" +
		"\tpullPeers: %s,\n" +
		"}"
	return fmt.Sprintf(reprFormat,
		membershipController.bootstrapper,
		membershipController.p2pAddr,
		membershipController.alphaSize,
		membershipController.betaSize,
		membershipController.gammaSize,
		membershipController.pushProbability,
		membershipController.roundPeriod,
		membershipController.powConfig,
		membershipController.viewList,
		membershipController.viewListCap,
		membershipController.sampleList,
		membershipController.sampleListRemainingCap,
		membershipController.pushRequests,
		membershipController.pullReplies,
		membershipController.pullPeers,
	)
}

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
	"fmt"
	"io"
	"math"
	"math/big"
	mrand "math/rand"
	"time"
)

var membershipControllerHandlers map[InternalMessageType]func(*MembershipController, AnyMessage) error

// init is an initialization function for 'main' package, called by Go.
func init() {
	membershipControllerHandlers = map[InternalMessageType]func(*MembershipController, AnyMessage) error{}
	membershipControllerHandlers[PeerDisconnectedMSG] = (*MembershipController).peerDisconnectedHandler
	membershipControllerHandlers[ProbePeerReplyMSG] = (*MembershipController).probePeerReplyHandler
	membershipControllerHandlers[MembershipIncomingPushRequestMSG] = (*MembershipController).incomingPushRequestHandler
	membershipControllerHandlers[MembershipIncomingPullRequestMSG] = (*MembershipController).incomingPullRequestHandler
	membershipControllerHandlers[MembershipIncomingPullReplyMSG] = (*MembershipController).incomingPullReplyHandler
	membershipControllerHandlers[MembershipCloseMSG] = (*MembershipController).closeHandler
}

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
	pullReplies *indexedset.IndexedSet
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
				err = fmt.Errorf(x)
			case error:
				err = x
			default:
				err = fmt.Errorf("Unknown panic in MinWiseIndependentPermutation::Permute")
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

// NextSet is the method for introducing peer sampler with a set
// of peers. If at least one new peer is sampled, then returns nil.
//
// peerSet must be a set.Set of type 'Peer'.
func (peerSampler *PeerSampler) NextSet(peerSet set.Set) error {
	err := fmt.Errorf("peerSampler not changed yet")
	// Feed all of the new peers into the sampler.
	for elem := range peerSet.Iterate() {
		newPeer := elem.(Peer)
		curErr := peerSampler.Next(&newPeer)
		if curErr == nil {
			err = nil
		}
	}
	return err
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
		pullReplies:            indexedset.New(),
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
			err = fmt.Errorf(x)
		case error:
			err = x
		default:
			err = fmt.Errorf("Unknown panic in MembershipController")
		}

		// Clear the input queue.
		for len(membershipController.MsgInQueue) > 0 {
			<-membershipController.MsgInQueue
		}
		// send MembershipCrashedMSG to the Central controller!
		membershipController.MsgOutQueue <- InternalMessage{Type: MembershipCrashedMSG, Payload: err}
	}
}

// replaceViewList is the method to use when replacing the 'viewList' with a new one.
// It not only replaces but also informs the Central controller.
//
// The 'newViewList' argument is possibly modified.
func (membershipController *MembershipController) replaceViewList(newViewList *indexedset.IndexedSet) {
	toBeRemoved := indexedset.New()
	toBeAdded := newViewList
	viewList := membershipController.viewList

	for elem := range viewList.Iterate() {
		peer := elem.(Peer)
		if !newViewList.IsMember(peer) {
			// if the peer is not in the intersection, then we need to remove it!
			toBeRemoved.Add(peer)
		} else {
			// if the peer is in the intersection, then we don't need to add it!
			toBeAdded.Remove(peer)
		}
	}
	for elem := range toBeRemoved.Iterate() {
		peer := elem.(Peer)
		// send PeerRemoveMSG message to the Central controller!
		membershipController.MsgOutQueue <- InternalMessage{Type: PeerRemoveMSG, Payload: peer}

		viewList.Remove(peer)
	}
	for elem := range toBeAdded.Iterate() {
		peer := elem.(Peer)
		// send PeerAddMSG message to the Central controller!
		membershipController.MsgOutQueue <- InternalMessage{Type: PeerAddMSG, Payload: peer}

		viewList.Add(peer)
	}
}

// removePeer is the method to use when a remote peer goes down
// and everything related to that peer needs to be removed.
func (membershipController *MembershipController) removePeer(peer Peer) {
	if membershipController.viewList.IsMember(peer) {
		// Remove the peer from viewList.
		membershipController.viewList.Remove(peer)
		// Command the Central controller to remove the peer, since it cannot
		// modify its 'viewList' without our explicit instructions. This has
		// to be the case for the sake of eventual consistency between viewList's
		// of both controllers.
		membershipController.MsgOutQueue <- InternalMessage{Type: PeerRemoveMSG, Payload: peer}
	}
	// Check if the peer exists in sampleList.
	if membershipController.sampleList.IsMember(peer) {
		// Remove the peer from sampleList.
		value := membershipController.sampleList.GetValue(peer)
		peerSamplerSet := value.(set.Set)
		membershipController.sampleListRemainingCap += uint32(peerSamplerSet.Len())
		membershipController.sampleList.Remove(peer)
	}
	// Remove the peer from pushRequests.
	membershipController.pushRequests.Remove(peer)
	// Remove the peer from the pullReplies.
	membershipController.pullReplies.Remove(peer)
	// Remove the peer from the pullPeers.
	membershipController.pullPeers.Remove(peer)
}

// pushRound is the method for performing limited push requests
// during a membership round, as desribed in the BRAHMS paper.
func (membershipController *MembershipController) pushRound() {
	pushIndexes := mrand.Perm(membershipController.viewList.Len())[:membershipController.alphaSize]
	for _, i := range pushIndexes {
		if mrand.Float64() <= membershipController.pushProbability {
			ithElem := membershipController.viewList.ElemAtIndex(i)
			remotePeer := ithElem.(Peer)
			pushReq, err := NewMembershipPushRequestMSGPayload(
				Peer{membershipController.p2pAddr},
				remotePeer,
				membershipController.powConfig.hardness,
				membershipController.powConfig.repetition,
			)
			if err != nil {
				continue
			}

			// Send the push request message to the Central controller.
			membershipController.MsgOutQueue <- InternalMessage{Type: MembershipPushRequestMSG, Payload: pushReq}
		}
	}
	// Increase the pushProbability for the next time.
	membershipController.pushProbability = 1.0 - (1.0-membershipController.pushProbability)*0.9
}

// pullRound is the method for performing pull requests during
// a membership round, as desribed in the BRAHMS paper.
func (membershipController *MembershipController) pullRound() {
	membershipController.pullPeers = set.New()
	pullIndexes := mrand.Perm(membershipController.viewList.Len())[:membershipController.betaSize]
	for _, i := range pullIndexes {
		ithElem := membershipController.viewList.ElemAtIndex(i)
		peer := ithElem.(Peer)
		membershipController.pullPeers.Add(peer)

		// Send the pull request message to the Central controller.
		membershipController.MsgOutQueue <- InternalMessage{Type: MembershipPullRequestMSG, Payload: peer}
	}
}

// updateRound is the method for updating the old view list with the
// new one consisting of pushed, pulled and sampled peers as described
// in the BRAHMS paper.
func (membershipController *MembershipController) updateRound() {
	// Update the viewList only if there are no more than alphaSize push requests AND
	// either pushed or pulled peers are non-empty.
	if membershipController.pushRequests.Len() <= int(membershipController.alphaSize) &&
		(membershipController.pushRequests.Len() > 0 || membershipController.pullReplies.Len() > 0) {
		newViewList := indexedset.New()
		// Add up to alphaSize pushed peers into the new view list.
		for elem := range membershipController.pushRequests.Iterate() {
			peer := elem.(Peer)
			newViewList.Add(peer)
		}
		// Add up to betaSize pulled peers into the new view list.
		pullIndexes := mrand.Perm(membershipController.pullReplies.Len())[:membershipController.betaSize]
		for _, i := range pullIndexes {
			ithElem := membershipController.pullReplies.ElemAtIndex(i)
			peer := ithElem.(Peer)
			newViewList.Add(peer)
		}
		// Add up to gammaSize sampled peers into the new view list.
		sampleIndexes := mrand.Perm(membershipController.sampleList.Len())[:membershipController.gammaSize]
		for _, i := range sampleIndexes {
			ithElem, _ := membershipController.sampleList.KeyAtIndex(i)
			peer := ithElem.(Peer)
			newViewList.Add(peer)
		}
		// Replace the old view list with the new one.
		membershipController.replaceViewList(newViewList)
	}
}

// updateSampleRound is the method for updating the old peer samplers
// inside sampleList by introducing the new pushed and pulled peers.
func (membershipController *MembershipController) updateSampleRound() {
	newPeers := membershipController.pushRequests
	for elem := range membershipController.pullReplies.Iterate() {
		peer := elem.(Peer)
		newPeers.Add(peer)
	}
	membershipController.pushRequests = set.New()
	membershipController.pullReplies = indexedset.New()

	// If there is remaining capacity, create new peer samplers and
	// introduce new peers to the new peer samplers.
	newPeerSamplers := make([]*PeerSampler, 0)
	for i := uint32(0); i < membershipController.sampleListRemainingCap; i++ {
		peerSampler, err := NewPeerSampler()
		if err != nil {
			continue
		}
		err = peerSampler.NextSet(newPeers)
		// Check if the peer sampler has a different peer.
		if err == nil {
			newPeerSamplers = append(newPeerSamplers, peerSampler)
		}
	}
	// Reduce the remaining capacity for new peer samplers.
	membershipController.sampleListRemainingCap -= uint32(len(newPeerSamplers))

	// Introduce new peers to the old peer samplers.
	peersToRemove := make([]Peer, 0)
	for key, valueAndIndex := range membershipController.sampleList.Iterate() {
		oldPeer := key.(Peer)
		value := valueAndIndex.Value
		// value is a set.Set of *PeerSampler .
		peerSamplerSet := value.(set.Set)
		peerSamplersToRemove := make([]*PeerSampler, 0)
		for psElem := range peerSamplerSet.Iterate() {
			peerSampler := psElem.(*PeerSampler)
			err := peerSampler.NextSet(newPeers)
			// Check if the peer sampler has a different peer.
			if err == nil {
				peerSamplersToRemove = append(peerSamplersToRemove, peerSampler)
				newPeerSamplers = append(newPeerSamplers, peerSampler)
			}
		}
		// Remove the peer samplers that have a different peer.
		for _, peerSampler := range peerSamplersToRemove {
			peerSamplerSet.Remove(peerSampler)
		}
		// If peer samplers of a peer are all gone, then the peer
		// needs to be removed from the sampleList.
		if peerSamplerSet.Len() == 0 {
			peersToRemove = append(peersToRemove, oldPeer)
		}
	}
	// Remove peers without peer samplers from the sampleList.
	for _, oldPeer := range peersToRemove {
		membershipController.sampleList.Remove(oldPeer)
	}
	// Add new peer samplers back into the sampleList.
	for _, peerSampler := range newPeerSamplers {
		peer := peerSampler.Sample()
		value := membershipController.sampleList.GetValue(peer)
		if value == nil {
			peerSamplerSet := set.New().Add(peerSampler)
			membershipController.sampleList.Put(peer, peerSamplerSet)
		} else {
			peerSamplerSet := value.(set.Set)
			peerSamplerSet.Add(peerSampler)
		}
	}
}

// probePeerRound is executed to probe every peer in the sampleList.
func (membershipController *MembershipController) probePeerRound() {
	for elem := range membershipController.sampleList.Iterate() {
		peer := elem.(Peer)
		// Send probe peer request message to the Central controller.
		membershipController.MsgOutQueue <- InternalMessage{Type: ProbePeerRequestMSG, Payload: peer}
	}
}

// membershipRound is a method for executing 1 round of membership exchange.
// Normally it is executed only periodically. However, if bootstrapping for
// the first time, the round is also executed.
func (membershipController *MembershipController) membershipRound() {
	membershipController.pushRound()
	membershipController.pullRound()
	membershipController.updateRound()
	membershipController.updateSampleRound()

	membershipController.probePeerRound()
}

// bootstrap puts the bootstrapper peer into the viewList and starts
// a fresh membership round.
func (membershipController *MembershipController) bootstrap() {
	newViewList := indexedset.New().Add(Peer{Addr: membershipController.bootstrapper})
	membershipController.replaceViewList(newViewList)
	membershipController.pushProbability = 0.0
	// execute a round of push and pull with the bootstrapper peer
	membershipController.pushRound()
	membershipController.pullRound()
}

// peerDisconnectedHandler is the method called by controllerRoutine for when
// it receives an internal message of type PeerDisconnectedMSG.
func (membershipController *MembershipController) peerDisconnectedHandler(payload AnyMessage) error {
	peer, ok := payload.(Peer)
	if !ok {
		return nil
	}
	membershipController.removePeer(peer)

	return nil
}

// probePeerReplyHandler is the method called by controllerRoutine for when
// it receives an internal message of type ProbePeerReplyMSG.
func (membershipController *MembershipController) probePeerReplyHandler(payload AnyMessage) error {
	reply, ok := payload.(ProbePeerReplyMSGPayload)
	if !ok {
		return nil
	}
	if reply.ProbeResult == false {
		peer := reply.Probed
		membershipController.removePeer(peer)
	}

	return nil
}

// incomingPushRequestHandler is the method called by controllerRoutine for when
// it receives an internal message of type MembershipIncomingPushRequestMSG.
func (membershipController *MembershipController) incomingPushRequestHandler(payload AnyMessage) error {
	pr, ok := payload.(MembershipPushRequestMSGPayload)
	if !ok {
		return nil
	}
	if time.Now().UTC().Sub(pr.When) <= membershipController.powConfig.validityDuration &&
		pr.To.Addr == membershipController.p2pAddr {
		k := PoWThreshold(membershipController.powConfig.repetition, 256)
		hashVal, err := pr.HashVal(membershipController.powConfig.hardness)
		if err == nil && hashVal.Cmp(k) <= 0 {
			// If the pushed peer is valid, then add to pushRequests.
			if pr.From.ValidateAddr() == nil {
				membershipController.pushRequests.Add(pr.From)
			}
		}
	}

	return nil
}

// incomingPullRequestHandler is the method called by controllerRoutine for when
// it receives an internal message of type MembershipIncomingPullRequestMSG.
func (membershipController *MembershipController) incomingPullRequestHandler(payload AnyMessage) error {
	pr, ok := payload.(MembershipIncomingPullRequestMSGPayload)
	if !ok {
		return nil
	}
	reply := MembershipPullReplyMSGPayload{To: pr.From}
	for elem := range membershipController.viewList.Iterate() {
		peer := elem.(Peer)
		reply.ViewList = append(reply.ViewList, peer)
	}
	// Send the pull reply back to the Central controller.
	membershipController.MsgOutQueue <- InternalMessage{Type: MembershipPullReplyMSG, Payload: reply}

	return nil
}

// incomingPullReplyHandler is the method called by controllerRoutine for when
// it receives an internal message of type MembershipIncomingPullReplyMSG.
func (membershipController *MembershipController) incomingPullReplyHandler(payload AnyMessage) error {
	reply, ok := payload.(MembershipIncomingPullReplyMSGPayload)
	if !ok {
		return nil
	}
	// Check if we actually asked for this pull reply.
	if membershipController.pullPeers.IsMember(reply.From) {
		membershipController.pullPeers.Remove(reply.From)
		// Add all peers into the pullReplies.
		for _, peer := range reply.ViewList {
			// If the pushed peer is valid, then add to pullReplies.
			if peer.ValidateAddr() == nil {
				membershipController.pullReplies.Add(peer)
			}
		}
	}

	return nil
}

// closeHandler is the method called by controllerRoutine for when
// it receives an internal message of type MembershipCloseMSG.
func (membershipController *MembershipController) closeHandler(payload AnyMessage) error {
	_, ok := payload.(void)
	if !ok {
		return nil
	}
	// Clear the input queue.
	for len(membershipController.MsgInQueue) > 0 {
		<-membershipController.MsgInQueue
	}
	// send MembershipClosedMSG to the Central controller!
	membershipController.MsgOutQueue <- InternalMessage{Type: MembershipClosedMSG, Payload: void{}}
	// Signal for graceful closure.
	return &CloseError{}
}

func (membershipController *MembershipController) controllerRoutine() {
	defer membershipController.recover()
	membershipController.bootstrap()
	roundTicker := time.NewTicker(membershipController.roundPeriod)
	defer roundTicker.Stop()

	for done := false; !done; {
		// Check for the round ticker first.
		select {
		case <-roundTicker.C:
			membershipController.membershipRound()
		default:
			break
		}
		// Check for any incoming event.
		select {
		case <-roundTicker.C:
			membershipController.membershipRound()
		case im := <-membershipController.MsgInQueue:
			handler := membershipControllerHandlers[im.Type]
			err := handler(membershipController, im.Payload)
			if err != nil {
				switch err.(type) {
				case *CloseError:
					done = true
				default:
					break
				}
			}
		}
	}
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

package main

import (
	"datastruct/set"
	"fmt"
	"math"
	"mathutils"
	"time"
)

var gossiperControllerHandlers map[InternalMessageType]func(*Gossiper, AnyMessage) error

// init is an initialization function for 'main' package, called by Go.
func init() {
	gossiperControllerHandlers = map[InternalMessageType]func(*Gossiper, AnyMessage) error{}
	gossiperControllerHandlers[RandomPeerListReplyMSG] = (*Gossiper).randomPeerListReplyHandler
	gossiperControllerHandlers[GossipAnnounceMSG] = (*Gossiper).announceHandler
	gossiperControllerHandlers[GossipNotifyMSG] = (*Gossiper).notifyHandler
	gossiperControllerHandlers[GossipUnnofityMSG] = (*Gossiper).unnotifyHandler
	gossiperControllerHandlers[GossipValidationMSG] = (*Gossiper).validationHandler
	gossiperControllerHandlers[GossipIncomingPushMSG] = (*Gossiper).incomingPushHandler
	gossiperControllerHandlers[GossipIncomingPullRequestMSG] = (*Gossiper).incomingPullRequestHandler
	gossiperControllerHandlers[GossipIncomingPullReplyMSG] = (*Gossiper).incomingPullReplyHandler
	gossiperControllerHandlers[GossiperCloseMSG] = (*Gossiper).closeHandler
}

// MedianCounterConfig holds the configuration for the maximum counter
// value of a gossip item to be considered in state B and likewise in
// state C. The paper authors suggests O(loglog(n)) for both.
type MedianCounterConfig struct {
	bMax uint8
	cMax uint8
}

// GossiperNextRoundPullPeersType is the type of variable stored in
// Gossiper::nextRoundPullPeers.
type GossiperNextRoundPullPeersType Peer

// GossiperPullPeersType is the type of variable stored in
// Gossiper::pullPeers.
type GossiperPullPeersType Peer

// Gossiper is the struct for the goroutine that will be exclusively responsible for handling all p2p Gossip
// calls. It will both share items by Gossiping itself and it will also respond to the GossipPullRequest calls.
type Gossiper struct {
	// cacheSize is the maximum number of Gossip items to gossip at any time.
	cacheSize uint16
	// degree is the number of peers to gossip with per round.
	degree uint8
	// maxTTL is the maximum number of hops to propagate any Gossip item.
	maxTTL uint8
	// roundPeriod is the time duration between each membership round.
	roundPeriod time.Duration
	// mcConfig is the configuration for the "median-counter algorithm".
	mcConfig MedianCounterConfig
	// gossipList is going to contain all hot topics to propagate. Hence it is of size 'cache_size'.
	gossipList map[GossipItem]*GossipItemInfoGossiper
	// oldGossipList is going to contain all outdated gossips. If an incoming gossip is
	// in this map then it will be ignored and not propagated any further.
	oldGossipList map[GossipItem]*GossipItemInfoGossiper
	// apiClientsToNotify is a map of API clients to be notified upon receiving
	// a gossip item with a 'data type' that interests the client.
	apiClientsToNotify map[APIClient]*APIClientInfoGossiper
	// incomingGossips is a set of gossip that arrived as either push request or
	// pull reply since the last gossip round.
	incomingGossips map[GossipItem]*GossipItemInfoGossiper
	// nextRoundPullPeers are peers to whom the Gossiper will make a pull request
	// in the next gossip round.
	nextRoundPullPeers set.Set
	// pullPeers are peers to whom the Gossiper sent a gossip pull request and
	// is waiting for a pull reply. Any gossip pull reply from a peer outside of
	// this set will be ignored!
	pullPeers set.Set
	// MsgInQueue is the incoming message queue for
	// the Gossiper goroutine.
	MsgInQueue chan InternalMessage
	// MsgOutQueue is the outgoing message queue from
	// the Gossiper goroutine to the Central controller.
	MsgOutQueue chan InternalMessage
}

// NewGossiper is the constructor function for the Gossiper struct.
func NewGossiper(cacheSize uint16, degree, maxTTL uint8, roundPeriod time.Duration, maxPeers float64,
	inQ, outQ chan InternalMessage,
) (*Gossiper, error) {
	denominator := math.Log2(math.Max(2, float64(degree)))
	logN := math.Log2(maxPeers) / denominator
	loglogN := uint8(math.Max(1, math.Ceil(math.Log2(logN)/denominator)))
	return &Gossiper{
		cacheSize:          cacheSize,
		degree:             degree,
		maxTTL:             maxTTL,
		roundPeriod:        roundPeriod,
		mcConfig:           MedianCounterConfig{bMax: loglogN, cMax: loglogN},
		gossipList:         map[GossipItem]*GossipItemInfoGossiper{},
		oldGossipList:      map[GossipItem]*GossipItemInfoGossiper{},
		apiClientsToNotify: map[APIClient]*APIClientInfoGossiper{},
		incomingGossips:    map[GossipItem]*GossipItemInfoGossiper{},
		nextRoundPullPeers: set.New(),
		pullPeers:          set.New(),
		MsgInQueue:         inQ,
		MsgOutQueue:        outQ,
	}, nil
}

// recover method tries to catch a panic in controllerRoutine if it exists, then
// informs the Central controller about the crash.
func (gossiper *Gossiper) recover() {
	var err error
	if r := recover(); r != nil {
		// find out exactly what the error was and set err
		switch x := r.(type) {
		case string:
			err = fmt.Errorf(x)
		case error:
			err = x
		default:
			err = fmt.Errorf("Unknown panic in Gossiper")
		}

		// Clear the input queue.
		for len(gossiper.MsgInQueue) > 0 {
			<-gossiper.MsgInQueue
		}
		// send GossiperCrashedMSG to the Central controller!
		gossiper.MsgOutQueue <- InternalMessage{Type: GossiperCrashedMSG, Payload: err}
	}
}

// pushRound is the method for performing gossip push
// during a gossip round.
func (gossiper *Gossiper) pushRound() {
	itemsToRemove := make([]*GossipItem, 0)
	for item, info := range gossiper.gossipList {
		peerIndexes := make([]int, mathutils.Min(int(gossiper.degree), len(info.peerList)))
		// Set the peer indexes to gossip this item with as:
		// (ttl * degree) mod N, ..., (ttl * degree + degree - 1) mod N
		// where N == len(peerList) .
		for i := range peerIndexes {
			peerIndexes[i] = (int(info.s.ttl)*int(gossiper.degree) + i) % len(info.peerList)
		}
		// Gossip the item.
		for _, peerIndex := range peerIndexes {
			gossiper.MsgOutQueue <- InternalMessage{
				Type: GossipPushMSG,
				Payload: GossipPushMSGPayload{
					Item:    &item,
					State:   info.s.state,
					Counter: info.s.counter,
					To:      info.peerList[peerIndex],
				},
			}
		}
		// Change the state of gossip item to reflect this new round.
		info.s.ttl--
		if info.s.ttl == 0 {
			itemsToRemove = append(itemsToRemove, &item)
		} else if info.s.state == MedianCounterStateC {
			info.s.counter++
			if info.s.counter >= gossiper.mcConfig.cMax {
				itemsToRemove = append(itemsToRemove, &item)
			}
		}
	}
	// Remove the items to be removed into oldGossipList.
	for _, itemToRemove := range itemsToRemove {
		// Release peers allocated to this gossip item.
		releasedPeers := gossiper.gossipList[*itemToRemove].peerList
		gossiper.MsgOutQueue <- InternalMessage{
			Type:    RandomPeerListReleaseMSG,
			Payload: RandomPeerListReleaseMSGPayload{releasedPeers},
		}
		// Remove the item from gossipList into oldGossipList.
		delete(gossiper.gossipList, *itemToRemove)
		// Keep the old gossip items for maxTTL gossip rounds, then remove them entirely.
		gossiper.oldGossipList[*itemToRemove] = &GossipItemInfoGossiper{
			s: GossipItemState{state: MedianCounterStateD, ttl: gossiper.maxTTL},
		}
	}
}

// pullRound is the method for performing gossip pull requests
// during a gossip round.
func (gossiper *Gossiper) pullRound() {
	for elem := range gossiper.nextRoundPullPeers.Iterate() {
		peer := elem.(Peer)

		// Send the pull request message to the Central controller.
		gossiper.MsgOutQueue <- InternalMessage{Type: GossipPullRequestMSG, Payload: peer}
	}
	gossiper.pullPeers = gossiper.nextRoundPullPeers
	gossiper.nextRoundPullPeers = set.New()
	// Ask from the Central controller for more pull peer for the next round.
	gossiper.MsgOutQueue <- InternalMessage{
		Type:    RandomPeerListRequestMSG,
		Payload: RandomPeerListRequestMSGPayload{Related: nil, Num: int(gossiper.degree)},
	}
}

// notifyClients is the method for notifying clients that are interested
// in the given gossip item. DON'T GIVE nil GOSSIP ITEM!!!
func (gossiper *Gossiper) notifyClients(item *GossipItem) {
	// Inform any client of this new gossip item if they are interested.
	for client, cInfo := range gossiper.apiClientsToNotify {
		if cInfo.notifyDataTypes.IsMember(item.DataType) {
			// Send GossipNotificationMSG to the Central controller.
			gossiper.MsgOutQueue <- InternalMessage{
				Type:    GossipNotificationMSG,
				Payload: GossipNotificationMSGPayload{Who: client, Item: item, ID: cInfo.nextAvailableID}}
			cInfo.validationMap[cInfo.nextAvailableID] = item
			cInfo.nextAvailableID++
		}
	}
}

// updateRound is the method for updating the old gossip list with the
// newly arrived gossip items and their states according to the "median-
// counter algorithm". Also, notify clients about the gossips that they
// are interested in.
func (gossiper *Gossiper) updateRound() {
	for item, info := range gossiper.incomingGossips {
		// If the incoming gossip item is old, then ignore it.
		if _, isMember := gossiper.oldGossipList[item]; isMember {
			continue
		}
		// if I already have the incoming gossip item, just update my own state.
		if myInfo, isMember := gossiper.gossipList[item]; isMember {
			myInfo.UpdateItemInfo(info)
		} else {
			// Inform any client of this new gossip item if they are interested.
			gossiper.notifyClients(&item)
			// If we have space for new gossip items, add it.
			if len(gossiper.gossipList) < int(gossiper.cacheSize) {
				switch info.s.state {
				case MedianCounterStateB:
					gossiper.gossipList[item] = &GossipItemInfoGossiper{
						s: GossipItemState{state: MedianCounterStateB, counter: 1, medianRule: 0, ttl: gossiper.maxTTL},
					}
					// Ask for (degree * maxTTL) random peers for this gossip item.
					gossiper.MsgOutQueue <- InternalMessage{
						Type:    RandomPeerListRequestMSG,
						Payload: RandomPeerListRequestMSGPayload{Related: &item, Num: int(gossiper.degree) * int(gossiper.maxTTL)}}
				case MedianCounterStateC:
					gossiper.gossipList[item] = &GossipItemInfoGossiper{
						s: GossipItemState{state: MedianCounterStateC, counter: 0, medianRule: 0, ttl: gossiper.maxTTL},
					}
					// Ask for (degree * cMax) random peers for this gossip item, since it cannot be gossiped
					// for more than cMax more gossip rounds in state C.
					gossiper.MsgOutQueue <- InternalMessage{
						Type:    RandomPeerListRequestMSG,
						Payload: RandomPeerListRequestMSGPayload{Related: &item, Num: int(gossiper.degree) * int(gossiper.mcConfig.cMax)}}
				}
			}
		}
	}
	// Reset incomingGossips.
	gossiper.incomingGossips = map[GossipItem]*GossipItemInfoGossiper{}
	// Check for the median rule!
	for _, info := range gossiper.gossipList {
		switch info.s.state {
		case MedianCounterStateB:
			if info.s.medianRule > 0 {
				info.s.counter++
				if info.s.counter == gossiper.mcConfig.bMax {
					info.s.state = MedianCounterStateC
					info.s.counter = 0
				}
			}
			info.s.medianRule = 0
		}
	}
}

// updateOldGossipsRound is the method for reducing the time to live of
// all old gossips and remove them if it reaches 0.
func (gossiper *Gossiper) updateOldGossipsRound() {
	itemsToRemove := make([]*GossipItem, 0)
	for oldItem, info := range gossiper.oldGossipList {
		info.s.ttl--
		if info.s.ttl == 0 {
			itemsToRemove = append(itemsToRemove, &oldItem)
		}
	}
	// Remove old items with ttl == 0 .
	for _, oldItem := range itemsToRemove {
		delete(gossiper.oldGossipList, *oldItem)
	}
}

// gossipRound is a method for executing 1 round of gossip exchange.
// It is only executed periodically.
func (gossiper *Gossiper) gossipRound() {
	gossiper.pushRound()
	gossiper.pullRound()
	gossiper.updateRound()

	gossiper.updateOldGossipsRound()
}

// randomPeerListReplyHandler is the method called by controllerRoutine for when
// it receives an internal message of type RandomPeerListReplyMSG.
func (gossiper *Gossiper) randomPeerListReplyHandler(payload AnyMessage) error {
	reply, ok := payload.(RandomPeerListReplyMSGPayload)
	if !ok {
		return nil
	}
	// If the random peers are for pull request, add them to nextRoundPullPeers.
	if reply.Related == nil {
		for _, peer := range reply.RandomPeers {
			gossiper.nextRoundPullPeers.Add(peer)
		}
	} else if info, isMember := gossiper.gossipList[*reply.Related]; isMember {
		info.peerList = reply.RandomPeers
	} else {
		// These random peers are neither for pull nor for push requests. Just release them.
		gossiper.MsgOutQueue <- InternalMessage{
			Type:    RandomPeerListReleaseMSG,
			Payload: RandomPeerListReleaseMSGPayload{reply.RandomPeers}}
	}

	return nil
}

// announceHandler is the method called by controllerRoutine for when
// it receives an internal message of type GossipAnnounceMSG.
func (gossiper *Gossiper) announceHandler(payload AnyMessage) error {
	anno, ok := payload.(GossipAnnounceMSGPayload)
	if !ok {
		return nil
	}
	// Make sure the gossip item is not nil.
	if anno.Item == nil {
		return nil
	}
	// Inform any client of this new gossip item if they are interested.
	gossiper.notifyClients(anno.Item)
	// If gossipList doesn't have space for new gossip items,
	// then don't even consider the item.
	if len(gossiper.gossipList) >= int(gossiper.cacheSize) {
		return nil
	}
	// If the gossip item to announce is old OR if the
	// gossip item is already in the gossipList, then ignore it.
	_, isMember := gossiper.oldGossipList[*anno.Item]
	_, isMember2 := gossiper.gossipList[*anno.Item]
	if isMember || isMember2 {
		return nil
	}
	// Calculate the TTL for the gossip item.
	ttl := gossiper.maxTTL
	if anno.TTL != 0 {
		ttl = anno.TTL
		if ttl > gossiper.maxTTL {
			ttl = gossiper.maxTTL
		}
	}
	// Add the gossip item into the list of new gossips.
	gossiper.gossipList[*anno.Item] = &GossipItemInfoGossiper{
		s: GossipItemState{state: MedianCounterStateB, counter: 1, medianRule: 0, ttl: ttl}}
	// Ask for (degree * ttl) random peers for this gossip item.
	gossiper.MsgOutQueue <- InternalMessage{
		Type:    RandomPeerListRequestMSG,
		Payload: RandomPeerListRequestMSGPayload{Related: anno.Item, Num: int(gossiper.degree) * int(ttl)}}

	return nil
}

// notifyHandler is the method called by controllerRoutine for when
// it receives an internal message of type GossipNotifyMSG.
func (gossiper *Gossiper) notifyHandler(payload AnyMessage) error {
	ntf, ok := payload.(GossipNotifyMSGPayload)
	if !ok {
		return nil
	}
	// If the client is already registered, update its preferences.
	if info, isMember := gossiper.apiClientsToNotify[ntf.Who]; isMember {
		info.notifyDataTypes.Add(ntf.What)
	} else {
		// If the client is not registered, register it.
		gossiper.apiClientsToNotify[ntf.Who] = &APIClientInfoGossiper{
			notifyDataTypes: set.New().Add(ntf.What),
			validationMap:   map[uint16]*GossipItem{},
			nextAvailableID: 0}
	}

	return nil
}

// unnotifyHandler is the method called by controllerRoutine for when
// it receives an internal message of type GossipUnnofityMSG.
func (gossiper *Gossiper) unnotifyHandler(payload AnyMessage) error {
	client, ok := payload.(APIClient)
	if !ok {
		return nil
	}
	delete(gossiper.apiClientsToNotify, client)

	return nil
}

// validationHandler is the method called by controllerRoutine for when
// it receives an internal message of type GossipValidationMSG.
func (gossiper *Gossiper) validationHandler(payload AnyMessage) error {
	val, ok := payload.(GossipValidationMSGPayload)
	if !ok {
		return nil
	}
	// If the client who sent the GOSSIP VALIDATION is actually registered
	// and was sent the corresponding GOSSIP NOTIFICATION, then process it.
	// Otherwise, ignore the validation call.
	if info, isMember := gossiper.apiClientsToNotify[val.Who]; isMember {
		if item, isMember := info.validationMap[val.ID]; isMember {
			if !val.Valid {
				delete(gossiper.gossipList, *item)
				delete(gossiper.incomingGossips, *item)
				gossiper.oldGossipList[*item] = &GossipItemInfoGossiper{
					s: GossipItemState{state: MedianCounterStateD, ttl: gossiper.maxTTL},
				}
			}
			delete(info.validationMap, val.ID)
		}
	}

	return nil
}

// checkAndAddIncomingGossip performs sanity checks on the incoming gossip
// item. If found valid, the item is added to incomingGossips if either it
// doesn't exists or if it has a dominant state. Otherwise, it is ignored.
//
// Note that this method doesn't check the remaining capacity of the incomingGossips.
func (gossiper *Gossiper) checkAndAddIncomingGossip(itemExt *GossipItemExtended) error {
	// Perform sanity check for the pushed gossip item.
	if itemExt.Item == nil {
		return fmt.Errorf("itemExt.Item is nil")
	}
	var newInfo *GossipItemInfoGossiper = nil
	switch itemExt.State {
	case MedianCounterStateB:
		if itemExt.Counter < gossiper.mcConfig.bMax {
			newInfo = &GossipItemInfoGossiper{
				s: GossipItemState{
					state:      MedianCounterStateB,
					counter:    itemExt.Counter,
					medianRule: 0,
					ttl:        gossiper.maxTTL}}
		}
	case MedianCounterStateC:
		if itemExt.Counter < gossiper.mcConfig.cMax {
			newInfo = &GossipItemInfoGossiper{
				s: GossipItemState{
					state:      MedianCounterStateC,
					counter:    0,
					medianRule: 0,
					ttl:        gossiper.maxTTL}}
		}
	}
	if newInfo != nil {
		// If we already have this gossip item, then store whichever item
		// has a dominant state.
		if info, isMember := gossiper.incomingGossips[*itemExt.Item]; isMember {
			if info.s.Cmp(&newInfo.s) < 0 {
				gossiper.incomingGossips[*itemExt.Item] = newInfo
			}
		} else {
			// If the gossip item is not in the incomingGossips, just add it.
			gossiper.incomingGossips[*itemExt.Item] = newInfo
			return nil
		}
	}

	return fmt.Errorf("itemExt.Item is not added")
}

// incomingPushHandler is the method called by controllerRoutine for when
// it receives an internal message of type GossipIncomingPushMSG.
func (gossiper *Gossiper) incomingPushHandler(payload AnyMessage) error {
	itemExt, ok := payload.(GossipItemExtended)
	if !ok {
		return nil
	}
	// If incomingGossips is already full, ignore it.
	if len(gossiper.incomingGossips) >= int(gossiper.cacheSize) {
		return nil
	}
	return gossiper.checkAndAddIncomingGossip(&itemExt)
}

// incomingPullRequestHandler is the method called by controllerRoutine for when
// it receives an internal message of type GossipIncomingPullRequestMSG.
func (gossiper *Gossiper) incomingPullRequestHandler(payload AnyMessage) error {
	pr, ok := payload.(GossipIncomingPullRequestMSGPayload)
	if !ok {
		return nil
	}
	// Reply with a list of gossip items in our gossipList.
	itemList := make([]*GossipItemExtended, 0)
	for item, info := range gossiper.gossipList {
		itemList = append(itemList, &GossipItemExtended{Item: &item, State: info.s.state, Counter: info.s.counter})
	}
	// Send the GossipPullReplyMSG to the Central controller.
	gossiper.MsgOutQueue <- InternalMessage{
		Type: GossipPullReplyMSG, Payload: GossipPullReplyMSGPayload{To: pr.From, ItemList: itemList}}

	return nil
}

// incomingPullReplyHandler is the method called by controllerRoutine for when
// it receives an internal message of type GossipIncomingPullReplyMSG.
func (gossiper *Gossiper) incomingPullReplyHandler(payload AnyMessage) error {
	reply, ok := payload.(GossipIncomingPullReplyMSGPayload)
	if !ok {
		return nil
	}
	// Check whether we actually asked for this pull reply.
	if !gossiper.pullPeers.IsMember(reply.From) {
		return nil
	}
	// Process the incoming pull reply.
	upTo := int(gossiper.cacheSize) - len(gossiper.incomingGossips)
	for i := 0; i < len(reply.ItemList) && upTo > 0; i++ {
		if err := gossiper.checkAndAddIncomingGossip(reply.ItemList[i]); err == nil {
			upTo--
		}
	}
	gossiper.pullPeers.Remove(reply.From)

	return nil
}

// closeHandler is the method called by controllerRoutine for when
// it receives an internal message of type GossiperCloseMSG.
func (gossiper *Gossiper) closeHandler(payload AnyMessage) error {
	_, ok := payload.(void)
	if !ok {
		return nil
	}
	// Clear the input queue.
	for len(gossiper.MsgInQueue) > 0 {
		<-gossiper.MsgInQueue
	}
	// send GossiperClosedMSG to the Central controller!
	gossiper.MsgOutQueue <- InternalMessage{Type: GossiperClosedMSG, Payload: void{}}
	// Signal for graceful closure.
	return &CloseError{}
}

func (gossiper *Gossiper) controllerRoutine() {
	defer gossiper.recover()
	roundTicker := time.NewTicker(gossiper.roundPeriod)
	defer roundTicker.Stop()

	for done := false; !done; {
		// Check for the round ticker first.
		select {
		case <-roundTicker.C:
			gossiper.gossipRound()
		default:
			break
		}
		// Check for any incoming event.
		select {
		case <-roundTicker.C:
			gossiper.gossipRound()
		case im := <-gossiper.MsgInQueue:
			handler := gossiperControllerHandlers[im.Type]
			err := handler(gossiper, im.Payload)
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

// RunControllerGoroutine runs the Gossiper goroutine.
func (gossiper *Gossiper) RunControllerGoroutine() {
	go gossiper.controllerRoutine()
}

func (gossiper *Gossiper) String() string {
	reprFormat := "*Gossiper{\n" +
		"\tcacheSize: %d,\n" +
		"\tdegree: %d,\n" +
		"\tmaxTTL: %d,\n" +
		"\troundPeriod: %s,\n" +
		"\tmcConfig: %v,\n" +
		"\tgossipList: %s,\n" +
		"\toldGossipList: %s,\n" +
		"\tapiClientsToNotify: %s,\n" +
		"\tincomingGossips: %s,\n" +
		"\tnextRoundPullPeers: %s,\n" +
		"\tpullPeers: %s,\n" +
		"}"
	return fmt.Sprintf(reprFormat,
		gossiper.cacheSize,
		gossiper.degree,
		gossiper.maxTTL,
		gossiper.roundPeriod,
		gossiper.mcConfig,
		gossiper.gossipList,
		gossiper.oldGossipList,
		gossiper.apiClientsToNotify,
		gossiper.incomingGossips,
		gossiper.nextRoundPullPeers,
		gossiper.pullPeers,
	)
}

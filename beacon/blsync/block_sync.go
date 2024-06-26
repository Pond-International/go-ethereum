// Copyright 2023 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package blsync

import (
	"github.com/Pond-International/go-ethereum/beacon/light/request"
	"github.com/Pond-International/go-ethereum/beacon/light/sync"
	"github.com/Pond-International/go-ethereum/beacon/types"
	"github.com/Pond-International/go-ethereum/common"
	"github.com/Pond-International/go-ethereum/common/lru"
	"github.com/Pond-International/go-ethereum/event"
	"github.com/Pond-International/go-ethereum/log"
)

// beaconBlockSync implements request.Module; it fetches the beacon blocks belonging
// to the validated and prefetch heads.
type beaconBlockSync struct {
	recentBlocks *lru.Cache[common.Hash, *types.BeaconBlock]
	locked       map[common.Hash]request.ServerAndID
	serverHeads  map[request.Server]common.Hash
	headTracker  headTracker

	lastHeadInfo  types.HeadInfo
	chainHeadFeed event.FeedOf[types.ChainHeadEvent]
}

type headTracker interface {
	PrefetchHead() types.HeadInfo
	ValidatedHead() (types.SignedHeader, bool)
	ValidatedFinality() (types.FinalityUpdate, bool)
}

// newBeaconBlockSync returns a new beaconBlockSync.
func newBeaconBlockSync(headTracker headTracker) *beaconBlockSync {
	return &beaconBlockSync{
		headTracker:  headTracker,
		recentBlocks: lru.NewCache[common.Hash, *types.BeaconBlock](10),
		locked:       make(map[common.Hash]request.ServerAndID),
		serverHeads:  make(map[request.Server]common.Hash),
	}
}

func (s *beaconBlockSync) SubscribeChainHead(ch chan<- types.ChainHeadEvent) event.Subscription {
	return s.chainHeadFeed.Subscribe(ch)
}

// Process implements request.Module.
func (s *beaconBlockSync) Process(requester request.Requester, events []request.Event) {
	for _, event := range events {
		switch event.Type {
		case request.EvResponse, request.EvFail, request.EvTimeout:
			sid, req, resp := event.RequestInfo()
			blockRoot := common.Hash(req.(sync.ReqBeaconBlock))
			if resp != nil {
				s.recentBlocks.Add(blockRoot, resp.(*types.BeaconBlock))
			}
			if s.locked[blockRoot] == sid {
				delete(s.locked, blockRoot)
			}
		case sync.EvNewHead:
			s.serverHeads[event.Server] = event.Data.(types.HeadInfo).BlockRoot
		case request.EvUnregistered:
			delete(s.serverHeads, event.Server)
		}
	}
	s.updateEventFeed()
	// request validated head block if unavailable and not yet requested
	if vh, ok := s.headTracker.ValidatedHead(); ok {
		s.tryRequestBlock(requester, vh.Header.Hash(), false)
	}
	// request prefetch head if the given server has announced it
	if prefetchHead := s.headTracker.PrefetchHead().BlockRoot; prefetchHead != (common.Hash{}) {
		s.tryRequestBlock(requester, prefetchHead, true)
	}
}

func (s *beaconBlockSync) tryRequestBlock(requester request.Requester, blockRoot common.Hash, needSameHead bool) {
	if _, ok := s.recentBlocks.Get(blockRoot); ok {
		return
	}
	if _, ok := s.locked[blockRoot]; ok {
		return
	}
	for _, server := range requester.CanSendTo() {
		if needSameHead && (s.serverHeads[server] != blockRoot) {
			continue
		}
		id := requester.Send(server, sync.ReqBeaconBlock(blockRoot))
		s.locked[blockRoot] = request.ServerAndID{Server: server, ID: id}
		return
	}
}

func blockHeadInfo(block *types.BeaconBlock) types.HeadInfo {
	if block == nil {
		return types.HeadInfo{}
	}
	return types.HeadInfo{Slot: block.Slot(), BlockRoot: block.Root()}
}

func (s *beaconBlockSync) updateEventFeed() {
	head, ok := s.headTracker.ValidatedHead()
	if !ok {
		return
	}
	finality, ok := s.headTracker.ValidatedFinality() //TODO fetch directly if subscription does not deliver
	if !ok || head.Header.Epoch() != finality.Attested.Header.Epoch() {
		return
	}
	validatedHead := head.Header.Hash()
	headBlock, ok := s.recentBlocks.Get(validatedHead)
	if !ok {
		return
	}
	headInfo := blockHeadInfo(headBlock)
	if headInfo == s.lastHeadInfo {
		return
	}
	s.lastHeadInfo = headInfo

	// new head block and finality info available; extract executable data and send event to feed
	execBlock, err := headBlock.ExecutionPayload()
	if err != nil {
		log.Error("Error extracting execution block from validated beacon block", "error", err)
		return
	}
	s.chainHeadFeed.Send(types.ChainHeadEvent{
		BeaconHead: head.Header,
		Block:      execBlock,
		Finalized:  finality.Finalized.PayloadHeader.BlockHash(),
	})
}

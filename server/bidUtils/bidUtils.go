package bidUtils

import (
	"sync"
)

type BidinfoBroadcaster struct {
	UserChannels map[string]chan *BidInfo
	Mu           sync.Mutex
}

type BidInfo struct {
	Amount int32
	User   string
}

type HighestBidHolder struct {
	BidInfo *BidInfo
	Mu      sync.Mutex
}

func (bb *BidinfoBroadcaster) AddClient(user string, highestBid *BidInfo) bool {
	bb.Mu.Lock()
	defer bb.Mu.Unlock()

	if bb.UserChannels[user] == nil {
		bb.UserChannels[user] = make(chan *BidInfo)
		go Broadcast(bb.UserChannels[user], highestBid)
		return true
	}

	return false
}
func (bb *BidinfoBroadcaster) GetChannel(user string) chan *BidInfo {
	bb.Mu.Lock()
	defer bb.Mu.Unlock()

	return bb.UserChannels[user]
}

func (bb *BidinfoBroadcaster) BroadcastToAll(bidInfo *BidInfo) {
	bb.Mu.Lock()
	defer bb.Mu.Unlock()

	for u, c := range bb.UserChannels {
		if u == bidInfo.User {
			continue
		}

		go Broadcast(c, bidInfo)
	}
}

func Broadcast(c chan *BidInfo, bidInfo *BidInfo) {
	c <- bidInfo
}

func (bb *BidinfoBroadcaster) CloseAll() {
	bb.Mu.Lock()
	defer bb.Mu.Unlock()

	for _, c := range bb.UserChannels {
		select {
		case _ = <-c:
		default:
		}
		close(c)
	}

	bb.UserChannels = make(map[string]chan *BidInfo)
}

func (bb *BidinfoBroadcaster) GetAllUsers() []string {
	bb.Mu.Lock()
	defer bb.Mu.Unlock()

	allUsers := make([]string, 0)
	for user := range bb.UserChannels {
		allUsers = append(allUsers, user)
	}
	return allUsers
}

func (hb *HighestBidHolder) SetBid(Amount int32, User string) bool {
	hb.Mu.Lock()
	defer hb.Mu.Unlock()

	if hb.BidInfo.Amount >= Amount {
		return false
	}

	hb.BidInfo = &BidInfo{Amount: Amount, User: User}

	return true
}

func (hb *HighestBidHolder) GetHighestBid() *BidInfo {
	hb.Mu.Lock()
	defer hb.Mu.Unlock()

	return &BidInfo{Amount: hb.BidInfo.Amount, User: hb.BidInfo.User}
}

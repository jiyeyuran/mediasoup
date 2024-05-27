package rtc

import (
	"sync"
	"time"

	"github.com/google/btree"
)

const (
	MaxPacketAge      = 10000
	MaxNackPackets    = 1000
	DefaultRtt        = 100
	MaxNackRetries    = 10
	NackTimerInterval = 40 * time.Millisecond
)

type NackListener interface {
	OnNackGeneratorNackRequired(nackBatch []uint16)
	OnNackGeneratorKeyFrameRequired()
}

type NackInfo struct {
	createdAtMs uint64
	seq         uint16
	sendAtSeq   uint16
	retries     uint8
	sentAtMs    uint64
}

type NackFilter int

const (
	NackSeq NackFilter = iota
	NackTime
)

type NackGeneratorOption func(*NackGenerator)

func WithTimerInterval(d time.Duration) NackGeneratorOption {
	return func(ng *NackGenerator) {
		ng.timer.Stop()
		ng.timer = time.NewTimer(d)
	}
}

type NackGenerator struct {
	listener        NackListener
	sendNackDelayMs uint
	timer           *time.Timer
	rtt             uint32 // Round trip time (ms).
	started         bool
	lastSeq         uint16 // Seq number of last valid packet.
	nackList        map[uint16]NackInfo
	keyFrameList    *btree.BTreeG[uint16]
	recoveredList   *btree.BTreeG[uint16]
	quit            chan struct{}
	wg              sync.WaitGroup
}

func NewNackGenerator(listener NackListener, sendNackDelayMs uint, options ...NackGeneratorOption) *NackGenerator {
	ng := &NackGenerator{
		listener:        listener,
		sendNackDelayMs: sendNackDelayMs,
		timer:           time.NewTimer(NackTimerInterval),
		rtt:             DefaultRtt,
		nackList:        make(map[uint16]NackInfo),
		keyFrameList: btree.NewG(2, func(lhs, rhs uint16) bool {
			return IsSeqLowerThan(lhs, rhs)
		}),
		recoveredList: btree.NewG(2, func(lhs, rhs uint16) bool {
			return IsSeqLowerThan(lhs, rhs)
		}),
		quit: make(chan struct{}),
	}
	for _, option := range options {
		option(ng)
	}
	ng.wg.Add(1)
	go ng.runTimer()
	return ng
}

func (ng *NackGenerator) runTimer() {
	defer ng.wg.Done()
	for {
		select {
		case <-ng.timer.C:
			ng.onTimer()
		case <-ng.quit:
			return
		}
	}
}

// ReceivePacket returns true if this is a found nacked packet. False otherwise.
func (ng *NackGenerator) ReceivePacket(packet *RtpPacket, isRecovered bool) bool {
	seq := packet.GetSequenceNumber()
	isKeyFrame := packet.IsKeyFrame()

	if !ng.started {
		ng.started = true
		ng.lastSeq = seq

		if isKeyFrame {
			ng.keyFrameList.ReplaceOrInsert(seq)
		}

		return false
	}

	// Obviously never nacked, so ignore.
	if seq == ng.lastSeq {
		return false
	}

	// May be an out of order packet, or already handled retransmitted packet,
	// or a retransmitted packet.
	if IsSeqLowerThan(seq, ng.lastSeq) {
		if nackInfo, exists := ng.nackList[seq]; exists {
			// MS_DEBUG_DEV(
			// 	"NACKed packet received [ssrc:%" PRIu32 ", seq:%" PRIu16 ", recovered:%s]",
			// 	packet->GetSsrc(),
			// 	packet->GetSequenceNumber(),
			// 	isRecovered ? "true" : "false");
			delete(ng.nackList, seq)
			return nackInfo.retries != 0
		}
		// Out of order packet or already handled NACKed packet.
		if !isRecovered {
			// MS_WARN_DEV(
			// 	"ignoring older packet not present in the NACK list [ssrc:%" PRIu32 ", seq:%" PRIu16 "]",
			// 	packet->GetSsrc(),
			// 	packet->GetSequenceNumber());
		}
		return false
	}

	// If we are here it means that we may have lost some packets so seq is
	// newer than the latest seq seen.
	if isKeyFrame {
		ng.keyFrameList.ReplaceOrInsert(seq)
	}

	var dropItems []uint16
	ng.keyFrameList.AscendLessThan(seq-MaxPacketAge, func(item uint16) bool {
		dropItems = append(dropItems, item)
		return true
	})
	for _, item := range dropItems {
		ng.keyFrameList.Delete(item)
	}

	if isRecovered {
		ng.recoveredList.ReplaceOrInsert(seq)

		// Remove old ones so we don't accumulate recovered packets.
		dropItems = nil
		ng.recoveredList.AscendLessThan(seq-MaxPacketAge, func(item uint16) bool {
			dropItems = append(dropItems, item)
			return true
		})
		for _, item := range dropItems {
			ng.recoveredList.Delete(item)
		}
		// Do not let a packet pass if it's newer than last seen seq and came via
		// RTX.
		return false
	}

	ng.AddPacketsToNackList(ng.lastSeq+1, seq)
	ng.lastSeq = seq

	// Check if there are any nacks that are waiting for this seq number.
	nackBatch := ng.GetNackBatch(NackSeq)
	if len(nackBatch) > 0 {
		ng.listener.OnNackGeneratorNackRequired(nackBatch)
	}

	// This is important. Otherwise the running timer (filter:TIME) would be
	// interrupted and NACKs would never been sent more than once for each seq.
	ng.mayRunTimer()

	return false
}

func (ng *NackGenerator) AddPacketsToNackList(seqStart, seqEnd uint16) {
	for seq := seqStart; seq != seqEnd; seq++ {
		if _, exists := ng.nackList[seq]; !exists {
			ng.nackList[seq] = NackInfo{
				createdAtMs: uint64(time.Now().UnixMilli()),
				seq:         seq,
				sendAtSeq:   seq,
			}
		}
	}
}

func (ng *NackGenerator) GetNackBatch(filter NackFilter) []uint16 {
	nowMs := uint64(time.Now().UnixMilli())
	var nackBatch []uint16

	for seq, nackInfo := range ng.nackList {
		if ng.sendNackDelayMs > 0 && nowMs-nackInfo.createdAtMs < uint64(ng.sendNackDelayMs) {
			continue
		}

		if filter == NackSeq && nackInfo.sentAtMs == 0 {
			nackBatch = append(nackBatch, seq)
			nackInfo.retries++
			nackInfo.sentAtMs = nowMs

			if nackInfo.retries >= MaxNackRetries {
				delete(ng.nackList, seq)
			} else {
				ng.nackList[seq] = nackInfo
			}
		}
	}

	return nackBatch
}

func (ng *NackGenerator) Reset() {
	ng.nackList = make(map[uint16]NackInfo)
	ng.keyFrameList.Clear(false)
	ng.recoveredList.Clear(false)
	ng.started = false
	ng.lastSeq = 0
}

func (ng *NackGenerator) Close() {
	close(ng.quit)
	ng.wg.Wait()
	if !ng.timer.Stop() {
		<-ng.timer.C
	}
}

func (ng *NackGenerator) onTimer() {
	nackBatch := ng.GetNackBatch(NackTime)
	if len(nackBatch) > 0 {
		ng.listener.OnNackGeneratorNackRequired(nackBatch)
	}

	ng.mayRunTimer()
}

func (ng *NackGenerator) mayRunTimer() {
	if len(ng.nackList) > 0 {
		ng.timer.Reset(NackTimerInterval)
	} else {
		ng.timer.Stop()
	}
}

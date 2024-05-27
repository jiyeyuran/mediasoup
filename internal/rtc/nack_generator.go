package rtc

import (
	"log/slog"
	"time"

	"github.com/jiyeyuran/mediasoup/internal/pkg/set"
	"github.com/zhangyunhao116/skipmap"
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
	sentAtMs    uint64
	seq         uint16
	sendAtSeq   uint16
	retries     uint8
}

//go:generate stringer -type=NackFilter
type NackFilter int

const (
	NackFilterSeq NackFilter = iota
	NackFilterTime
)

type NackGenerator struct {
	listener        NackListener
	sendNackDelayMs uint
	timer           *SafeTimer
	rtt             uint32 // Round trip time (ms).
	started         bool
	lastSeq         uint16 // Seq number of last valid packet.
	nackList        *skipmap.FuncMap[uint16, *NackInfo]
	keyFrameList    *set.Set[uint16]
	recoveredList   *set.Set[uint16]
	logger          *slog.Logger
}

func NewNackGenerator(listener NackListener, sendNackDelayMs uint) *NackGenerator {
	less := func(a, b uint16) bool {
		return IsSeqLowerThan(a, b)
	}
	ng := &NackGenerator{
		listener:        listener,
		sendNackDelayMs: sendNackDelayMs,
		rtt:             DefaultRtt,
		nackList:        skipmap.NewFunc[uint16, *NackInfo](less),
		keyFrameList:    set.NewFunc(less),
		recoveredList:   set.NewFunc(less),
		logger:          slog.Default().With("typename", "NackGenerator"),
	}
	ng.timer = NewSafeTimer(NackTimerInterval, ng.onTimer)
	return ng
}

// ReceivePacket returns true if this is a found nacked packet. False otherwise.
func (ng *NackGenerator) ReceivePacket(packet *RtpPacket, isRecovered bool) bool {
	seq := packet.GetSequenceNumber()
	isKeyFrame := packet.IsKeyFrame()

	if !ng.started {
		ng.started = true
		ng.lastSeq = seq

		if isKeyFrame {
			ng.keyFrameList.Add(seq)
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
		if nackInfo, exists := ng.nackList.Load(seq); exists {
			ng.logger.Debug("NACKed packet received", "ssrc", packet.GetSsrc(),
				"seq", packet.GetSequenceNumber(), "recovered", isRecovered)

			ng.nackList.Delete(seq)
			return nackInfo.retries != 0
		}
		// Out of order packet or already handled NACKed packet.
		if !isRecovered {
			ng.logger.Warn("ignoring older packet not present in the NACK list",
				"ssrc", packet.GetSsrc(), "seq", packet.GetSequenceNumber())
		}
		return false
	}

	// If we are here it means that we may have lost some packets so seq is
	// newer than the latest seq seen.
	if isKeyFrame {
		ng.keyFrameList.Add(seq)
	}

	ng.keyFrameList.DeleteLessThan(seq - MaxPacketAge)

	if isRecovered {
		ng.recoveredList.Add(seq)

		// Remove old ones so we don't accumulate recovered packets.
		ng.recoveredList.DeleteLessThan(seq - MaxPacketAge)
		// Do not let a packet pass if it's newer than last seen seq and came via RTX.
		return false
	}

	ng.AddPacketsToNackList(ng.lastSeq+1, seq)
	ng.lastSeq = seq

	// Check if there are any nacks that are waiting for this seq number.
	nackBatch := ng.GetNackBatch(NackFilterSeq)
	if len(nackBatch) > 0 {
		ng.listener.OnNackGeneratorNackRequired(nackBatch)
	}

	// This is important. Otherwise the running timer (filter:TIME) would be
	// interrupted and NACKs would never been sent more than once for each seq.
	if !ng.timer.IsActive() {
		ng.mayRunTimer()
	}

	return false
}

func (ng *NackGenerator) AddPacketsToNackList(seqStart, seqEnd uint16) {
	// Remove old packets.
	ng.nackList.Range(func(key uint16, value *NackInfo) bool {
		if IsSeqLowerThan(key, seqEnd-MaxPacketAge) {
			ng.nackList.Delete(key)
			return true
		}
		return false
	})

	// If the nack list is too large, remove packets from the nack list until
	// the latest first packet of a keyframe. If the list is still too large,
	// clear it and request a keyframe.
	numNewNacks := seqEnd - seqStart

	if uint16(ng.nackList.Len())+numNewNacks > MaxNackPackets {
		for ng.RemoveNackItemsUntilKeyFrame() &&
			uint16(ng.nackList.Len())+numNewNacks > MaxNackPackets {
		}

		if uint16(ng.nackList.Len())+numNewNacks > MaxNackPackets {
			ng.logger.Warn("NACK list full, clearing it and requesting a key frame", "seqEnd", seqEnd)
			ng.clearNackList()
			ng.listener.OnNackGeneratorKeyFrameRequired()

			return
		}
	}

	for seq := seqStart; seq != seqEnd; seq++ {
		// Do not send NACK for packets that are already recovered by RTX.
		if ng.recoveredList.Contains(seq) {
			continue
		}
		ng.nackList.Store(seq, &NackInfo{
			createdAtMs: uint64(time.Now().UnixMilli()),
			seq:         seq,
			sendAtSeq:   seq,
		})
	}
}

func (ng *NackGenerator) RemoveNackItemsUntilKeyFrame() bool {
	for {
		first, ok := ng.keyFrameList.First()
		if !ok {
			break
		}
		found := false
		ng.nackList.Range(func(key uint16, value *NackInfo) bool {
			if IsSeqLowerThan(key, first) {
				ng.nackList.Delete(key)
				found = true
				return true
			}
			return false
		})
		if found {
			return true
		}
		// If this keyframe is so old it does not remove any packets from the list,
		// remove it from the list of keyframes and try the next keyframe.
		ng.keyFrameList.Remove(first)
	}

	return false
}

func (ng *NackGenerator) GetNackBatch(filter NackFilter) []uint16 {
	nowMs := uint64(time.Now().UnixMilli())
	var nackBatch []uint16

	ng.nackList.Range(func(seq uint16, nackInfo *NackInfo) bool {
		if ng.sendNackDelayMs > 0 && nowMs-nackInfo.createdAtMs < uint64(ng.sendNackDelayMs) {
			return true
		}

		if filter == NackFilterSeq && nackInfo.sentAtMs == 0 &&
			(nackInfo.sendAtSeq == ng.lastSeq || IsSeqHigherThan(ng.lastSeq, nackInfo.sendAtSeq)) {
			nackBatch = append(nackBatch, seq)
			nackInfo.retries++
			nackInfo.sentAtMs = nowMs

			if nackInfo.retries >= MaxNackRetries {
				ng.logger.Warn("sequence number removed from the NACK list due to max retries",
					"filter", filter, "seq", seq)
				ng.nackList.Delete(seq)
			}

			return true
		}

		if filter == NackFilterTime && (nackInfo.sentAtMs == 0 || nowMs-nackInfo.sentAtMs >= uint64(ng.rtt)) {
			nackBatch = append(nackBatch, seq)
			nackInfo.retries++
			nackInfo.sentAtMs = nowMs

			if nackInfo.retries >= MaxNackRetries {
				ng.logger.Warn("sequence number removed from the NACK list due to max retries",
					"filter", filter, "seq", seq)
				ng.nackList.Delete(seq)
			}

			return true
		}

		return true
	})

	return nackBatch
}

func (ng *NackGenerator) Reset() {
	ng.clearNackList()
	ng.keyFrameList.Clear()
	ng.recoveredList.Clear()
	ng.started = false
	ng.lastSeq = 0
}

func (ng *NackGenerator) Close() {
	ng.timer.Stop()
}

func (ng *NackGenerator) clearNackList() {
	ng.nackList = skipmap.NewFunc[uint16, *NackInfo](func(a, b uint16) bool {
		return IsSeqLowerThan(a, b)
	})
}

func (ng *NackGenerator) onTimer() {
	nackBatch := ng.GetNackBatch(NackFilterTime)
	if len(nackBatch) > 0 {
		ng.listener.OnNackGeneratorNackRequired(nackBatch)
	}

	ng.mayRunTimer()
}

func (ng *NackGenerator) mayRunTimer() {
	if ng.nackList.Len() > 0 {
		ng.timer.Reset(NackTimerInterval)
	} else {
		ng.timer.Stop()
	}
}

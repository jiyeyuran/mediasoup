package rtc

import "time"

type bufferItem struct {
	count uint64 // Count of items.
	time  uint64 // Time when the item was added (calculated by the time source).
}

type RateCalculator struct {
	windowSizeMs        uint64       // Window Size (in milliseconds).
	scale               float64      // Scale in which the rate is represented.
	windowItems         int          // Window Size (number of items).
	itemSizeMs          uint64       // Item Size calculated as: windowSizeMs / windowItems.
	buffer              []bufferItem // Buffer to keep data.
	newestItemStartTime uint64       // Time (in milliseconds) for last item in the time window.
	newestItemIndex     int          // Index for the last item in the time window.
	oldestItemStartTime uint64       // Time (in milliseconds) for oldest item in the time window.
	oldestItemIndex     int          // Index for the oldest item in the time window.
	totalCount          uint64       // Total count in the time window.
	bytes               uint64       // Total bytes transmitted.
	lastRate            uint32       // Last value calculated by GetRate().
	lastTime            uint64       // Last time GetRate() was called.
}

type rtpDataCounter struct {
	rate    RateCalculator // RateCalculator instance to use.
	packets uint64         // Count of packets.
}

func NewRateCalculator(windowSizeMs uint64, scale float64, windowItems int) *RateCalculator {
	itemSizeMs := windowSizeMs / uint64(windowItems)
	if itemSizeMs < 1 {
		itemSizeMs = 1
	}

	return &RateCalculator{
		windowSizeMs:    windowSizeMs,
		scale:           scale,
		windowItems:     windowItems,
		itemSizeMs:      itemSizeMs,
		buffer:          make([]bufferItem, windowItems),
		newestItemIndex: -1,
		oldestItemIndex: -1,
	}
}

func NewRtpDataCounter(windowSizeMs uint64) *rtpDataCounter {
	// Initialize rtpDataCounter with input arguments, and initialise rate to be
	// an instance of NewRateCalculator with defined windowSizeMs.
	return &rtpDataCounter{
		rate: *NewRateCalculator(windowSizeMs, 8000, 100),
	}
}

func (r *RateCalculator) Update(size, nowMs uint64) {
	// Ignore too old data. Should never happen.
	if nowMs < r.oldestItemStartTime {
		return
	}

	// Increase bytes.
	r.bytes += size

	r.removeOldData(nowMs)

	// If the elapsed time from the newest item start time is greater than the
	// item size (in milliseconds), increase the item index.
	if r.newestItemIndex < 0 || nowMs-r.newestItemStartTime >= r.itemSizeMs {
		r.newestItemIndex++
		r.newestItemStartTime = nowMs

		if r.newestItemIndex >= r.windowItems {
			r.newestItemIndex = 0
		}

		// Ensure the newest item index doesn't overlap with the oldest one
		if r.newestItemIndex == r.oldestItemIndex && r.oldestItemIndex != -1 {
			panic("newest index overlaps with the oldest one")
		}

		// Set the newest item.
		item := &r.buffer[r.newestItemIndex]
		item.count = size
		item.time = nowMs
	} else {
		// Update the newest item.
		item := &r.buffer[r.newestItemIndex]
		item.count += size
	}

	// Set the oldest item index and time, if not set.
	if r.oldestItemIndex < 0 {
		r.oldestItemIndex = r.newestItemIndex
		r.oldestItemStartTime = nowMs
	}

	r.totalCount += size

	// Reset lastRate and lastTime so GetRate() will calculate rate again even
	// if called with same now in the same loop iteration.
	r.lastRate = 0
	r.lastTime = 0
}

func (r *RateCalculator) GetRate(nowMs uint64) uint32 {
	if nowMs == r.lastTime {
		return r.lastRate
	}

	r.removeOldData(nowMs)

	scale := r.scale / float64(r.windowSizeMs)

	r.lastTime = nowMs
	r.lastRate = uint32(float64(r.totalCount)*scale + 0.5)

	return r.lastRate
}

func (r *RateCalculator) removeOldData(nowMs uint64) {
	// No item set.
	if r.newestItemIndex < 0 || r.oldestItemIndex < 0 {
		return
	}

	newOldestTime := nowMs - r.windowSizeMs

	// Oldest item already removed.
	if newOldestTime < r.oldestItemStartTime {
		return
	}

	// A whole window size time has elapsed since last entry. Reset the buffer.
	if newOldestTime >= r.newestItemStartTime {
		r.reset()
		return
	}

	for newOldestTime >= r.oldestItemStartTime {
		oldestItem := &r.buffer[r.oldestItemIndex]
		r.totalCount -= oldestItem.count
		oldestItem.count = 0
		oldestItem.time = 0

		if r.oldestItemIndex++; r.oldestItemIndex >= r.windowItems {
			r.oldestItemIndex = 0
		}
		newOldestItem := r.buffer[r.oldestItemIndex]
		r.oldestItemStartTime = newOldestItem.time
	}
}

func (r *RateCalculator) reset() {
	clear(r.buffer)
	r.newestItemIndex = -1
	r.oldestItemIndex = -1
	r.totalCount = 0
}

func (r *rtpDataCounter) Update(packet *rtpPacket) {
	nowMs := uint64(time.Now().UnixMilli())

	r.packets++
	r.rate.Update(packet.Size, nowMs)
}

func (r *rtpDataCounter) GetPacketCount() uint64 {
	return r.packets
}

func (r *rtpDataCounter) GetBytes() uint64 {
	return r.rate.bytes
}

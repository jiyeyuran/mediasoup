package mediasoup

import "time"

func getTimeMs() uint64 {
	return uint64(time.Now().UnixMilli())
}

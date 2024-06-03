package rtc

import (
	"time"

	"github.com/zhangyunhao116/skipmap"
)

const KeyFrameRetransmissionWaitTime = 1000 * time.Millisecond

type PendingKeyFrameInfoListener interface {
	OnKeyFrameRequestTimeout(keyFrameRequestInfo *PendingKeyFrameInfo)
}

type KeyFrameRequestDelayerListener interface {
	OnKeyFrameDelayTimeout(keyFrameRequestDelayer *KeyFrameRequestDelayer)
}

type KeyFrameRequestManagerListener interface {
	OnKeyFrameNeeded(keyFrameRequestManager *KeyFrameRequestManager, ssrc uint32)
}

type PendingKeyFrameInfo struct {
	listener       PendingKeyFrameInfoListener
	ssrc           uint32
	timer          *time.Timer
	timerDuration  time.Duration
	retryOnTimeout bool
	doneCh         chan struct{}
}

func NewPendingKeyFrameInfo(listener PendingKeyFrameInfoListener, ssrc uint32, timeout time.Duration) *PendingKeyFrameInfo {
	pkfi := &PendingKeyFrameInfo{
		listener:       listener,
		ssrc:           ssrc,
		timer:          time.NewTimer(timeout),
		timerDuration:  timeout,
		retryOnTimeout: true,
		doneCh:         make(chan struct{}),
	}

	// Start the timer with the specified wait time
	go pkfi.runTimer()

	return pkfi
}

func (pkfi *PendingKeyFrameInfo) runTimer() {
	defer pkfi.timer.Stop()

	select {
	case <-pkfi.timer.C:
		pkfi.listener.OnKeyFrameRequestTimeout(pkfi)
	case <-pkfi.doneCh:
	}
}

func (pkfi *PendingKeyFrameInfo) GetSsrc() uint32 {
	return pkfi.ssrc
}

func (pkfi *PendingKeyFrameInfo) SetRetryOnTimeout(notify bool) {
	pkfi.retryOnTimeout = notify
}

func (pkfi *PendingKeyFrameInfo) GetRetryOnTimeout() bool {
	return pkfi.retryOnTimeout
}

func (pkfi *PendingKeyFrameInfo) Restart() {
	pkfi.timer.Reset(pkfi.timerDuration)
}

func (pkfi *PendingKeyFrameInfo) Stop() {
	close(pkfi.doneCh)
}

type KeyFrameRequestDelayer struct {
	listener          KeyFrameRequestDelayerListener
	ssrc              uint32
	timer             *time.Timer
	keyFrameRequested bool
	doneCh            chan struct{}
}

func NewKeyFrameRequestDelayer(listener KeyFrameRequestDelayerListener, ssrc uint32, delay time.Duration) *KeyFrameRequestDelayer {
	kfrd := &KeyFrameRequestDelayer{
		listener: listener,
		ssrc:     ssrc,
		timer:    time.NewTimer(delay),
		doneCh:   make(chan struct{}),
	}
	// Start the timer with the specified delay
	go kfrd.runTimer()
	return kfrd
}

func (pkfi *KeyFrameRequestDelayer) runTimer() {
	defer pkfi.timer.Stop()

	select {
	case <-pkfi.timer.C:
		pkfi.listener.OnKeyFrameDelayTimeout(pkfi)
	case <-pkfi.doneCh:
	}
}

func (kfrd *KeyFrameRequestDelayer) GetSsrc() uint32 {
	return kfrd.ssrc
}

func (kfrd *KeyFrameRequestDelayer) GetKeyFrameRequested() bool {
	return kfrd.keyFrameRequested
}

func (kfrd *KeyFrameRequestDelayer) SetKeyFrameRequested(flag bool) {
	kfrd.keyFrameRequested = flag
}

func (kfrd *KeyFrameRequestDelayer) Stop() {
	close(kfrd.doneCh)
}

type KeyFrameRequestManager struct {
	listener                      KeyFrameRequestManagerListener
	keyFrameRequestDelay          time.Duration
	keyFrameRetransmissionWait    time.Duration
	mapSsrcPendingKeyFrameInfo    *skipmap.Uint32Map[*PendingKeyFrameInfo]
	mapSsrcKeyFrameRequestDelayer *skipmap.Uint32Map[*KeyFrameRequestDelayer]
}

func NewKeyFrameRequestManager(listener KeyFrameRequestManagerListener, keyFrameRequestDelay time.Duration, options ...func(*KeyFrameRequestManager)) *KeyFrameRequestManager {
	km := &KeyFrameRequestManager{
		listener:                      listener,
		keyFrameRequestDelay:          keyFrameRequestDelay,
		keyFrameRetransmissionWait:    KeyFrameRetransmissionWaitTime,
		mapSsrcPendingKeyFrameInfo:    skipmap.NewUint32[*PendingKeyFrameInfo](),
		mapSsrcKeyFrameRequestDelayer: skipmap.NewUint32[*KeyFrameRequestDelayer](),
	}
	for _, option := range options {
		option(km)
	}
	return km
}

func (kfrm *KeyFrameRequestManager) KeyFrameNeeded(ssrc uint32) {
	// Handle key frame request delay
	if kfrm.keyFrameRequestDelay > 0 {
		if kfrd, found := kfrm.mapSsrcKeyFrameRequestDelayer.Load(ssrc); found {
			// Enable the delayer and return
			kfrd.SetKeyFrameRequested(true)
			return
		} else {
			// Create a new delayer and continue
			kfrm.mapSsrcKeyFrameRequestDelayer.Store(ssrc, NewKeyFrameRequestDelayer(kfrm, ssrc, kfrm.keyFrameRequestDelay))
		}
	}

	// Check for pending key frame request
	if pkfi, found := kfrm.mapSsrcPendingKeyFrameInfo.Load(ssrc); found {
		// Re-request the key frame if not received on time
		pkfi.SetRetryOnTimeout(true)
		return
	}

	// Create a new pending key frame info and notify listener
	kfrm.mapSsrcPendingKeyFrameInfo.Store(ssrc, NewPendingKeyFrameInfo(kfrm, ssrc, kfrm.keyFrameRetransmissionWait))
	kfrm.listener.OnKeyFrameNeeded(kfrm, ssrc)
}

func (kfrm *KeyFrameRequestManager) ForceKeyFrameNeeded(ssrc uint32) {
	// Handle key frame request delay
	if kfrm.keyFrameRequestDelay > 0 {
		if kfrd, found := kfrm.mapSsrcKeyFrameRequestDelayer.LoadAndDelete(ssrc); found {
			kfrd.Stop()
		}
		// Create a new delayer
		kfrm.mapSsrcKeyFrameRequestDelayer.Store(ssrc, NewKeyFrameRequestDelayer(kfrm, ssrc, kfrm.keyFrameRequestDelay))
	}

	// Check for pending key frame request
	if pkfi, found := kfrm.mapSsrcPendingKeyFrameInfo.Load(ssrc); found {
		pkfi.SetRetryOnTimeout(true)
		pkfi.Restart()
	} else {
		// Create a new pending key frame info
		kfrm.mapSsrcPendingKeyFrameInfo.Store(ssrc, NewPendingKeyFrameInfo(kfrm, ssrc, kfrm.keyFrameRetransmissionWait))
	}

	// Notify listener about the key frame needed
	kfrm.listener.OnKeyFrameNeeded(kfrm, ssrc)
}

func (kfrm *KeyFrameRequestManager) KeyFrameReceived(ssrc uint32) {
	if pkfi, found := kfrm.mapSsrcPendingKeyFrameInfo.LoadAndDelete(ssrc); found {
		pkfi.Stop()
	}
}

func (kfrm *KeyFrameRequestManager) OnKeyFrameRequestTimeout(pkfi *PendingKeyFrameInfo) {
	pkfi, found := kfrm.mapSsrcPendingKeyFrameInfo.Load(pkfi.GetSsrc())
	if !found {
		return
	}

	if !pkfi.GetRetryOnTimeout() {
		pkfi.Stop()
		kfrm.mapSsrcPendingKeyFrameInfo.Delete(pkfi.GetSsrc())
		return
	}

	// Best effort in case the PLI/FIR was lost. Do not retry on timeout.
	pkfi.SetRetryOnTimeout(false)
	pkfi.Restart()
	// Requesting key frame on timeout
	kfrm.listener.OnKeyFrameNeeded(kfrm, pkfi.GetSsrc())
}

func (kfrm *KeyFrameRequestManager) OnKeyFrameDelayTimeout(kfrd *KeyFrameRequestDelayer) {
	kfrd, found := kfrm.mapSsrcKeyFrameRequestDelayer.LoadAndDelete(kfrd.GetSsrc())
	if !found {
		return
	}

	ssrc := kfrd.GetSsrc()
	keyFrameRequested := kfrd.GetKeyFrameRequested()

	kfrd.Stop()

	// Ask for a new key frame as normal if needed
	if keyFrameRequested {
		kfrm.KeyFrameNeeded(ssrc)
	}
}

func (kfrm *KeyFrameRequestManager) Stop() {
	kfrm.mapSsrcPendingKeyFrameInfo.Range(func(key uint32, value *PendingKeyFrameInfo) bool {
		value.Stop()
		return true
	})
	kfrm.mapSsrcKeyFrameRequestDelayer.Range(func(key uint32, value *KeyFrameRequestDelayer) bool {
		value.Stop()
		return true
	})
}

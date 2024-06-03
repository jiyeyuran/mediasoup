package rtc

import (
	"sync"
	"testing"
	"time"
)

func TestKeyFrameRequestManager(t *testing.T) {
	type keyFrameTest struct {
		name                          string
		repeatedRequests              int
		forceKeyFrame                 bool
		receiveKeyFrame               bool
		expectedOnKeyFrameNeededCalls int
	}

	tests := []keyFrameTest{
		{
			name:                          "key frame requested once, not received on time",
			repeatedRequests:              1,
			forceKeyFrame:                 false,
			receiveKeyFrame:               false,
			expectedOnKeyFrameNeededCalls: 2,
		},
		{
			name:                          "key frame requested many times, not received on time",
			repeatedRequests:              4,
			forceKeyFrame:                 false,
			receiveKeyFrame:               false,
			expectedOnKeyFrameNeededCalls: 2,
		},
		{
			name:                          "key frame is received on time",
			repeatedRequests:              1,
			forceKeyFrame:                 false,
			receiveKeyFrame:               true,
			expectedOnKeyFrameNeededCalls: 1,
		},
		{
			name:                          "key frame is forced, not received on time",
			repeatedRequests:              1,
			forceKeyFrame:                 true,
			receiveKeyFrame:               false,
			expectedOnKeyFrameNeededCalls: 3,
		},
		{
			name:                          "key frame is forced, received on time",
			repeatedRequests:              1,
			forceKeyFrame:                 true,
			receiveKeyFrame:               true,
			expectedOnKeyFrameNeededCalls: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			listener := NewTestKeyFrameRequestManagerListener()

			kfrm := NewKeyFrameRequestManager(listener, 5*time.Millisecond, func(kfrm *KeyFrameRequestManager) {
				kfrm.keyFrameRetransmissionWait = 5 * time.Millisecond
			})
			defer kfrm.Stop()

			for i := 0; i < test.repeatedRequests; i++ {
				kfrm.KeyFrameNeeded(1111)
				if test.forceKeyFrame {
					kfrm.ForceKeyFrameNeeded(1111)
				}
				if test.receiveKeyFrame {
					kfrm.KeyFrameReceived(1111)
				}
			}

			// Simulate waiting for operations to complete
			time.Sleep(20 * time.Millisecond)

			if got := listener.OnKeyFrameNeededTimesCalled; got != test.expectedOnKeyFrameNeededCalls {
				t.Errorf("expected %d OnKeyFrameNeeded calls, got %d", test.expectedOnKeyFrameNeededCalls, got)
			}
		})
	}
}

// Mock listener for testing
type TestKeyFrameRequestManagerListener struct {
	OnKeyFrameNeededTimesCalled int
	sync.Mutex
}

func NewTestKeyFrameRequestManagerListener() *TestKeyFrameRequestManagerListener {
	return &TestKeyFrameRequestManagerListener{}
}

func (l *TestKeyFrameRequestManagerListener) OnKeyFrameNeeded(kfrm *KeyFrameRequestManager, ssrc uint32) {
	l.Lock()
	defer l.Unlock()
	l.OnKeyFrameNeededTimesCalled++
}

func (l *TestKeyFrameRequestManagerListener) Reset() {
	l.Lock()
	defer l.Unlock()
	l.OnKeyFrameNeededTimesCalled = 0
}

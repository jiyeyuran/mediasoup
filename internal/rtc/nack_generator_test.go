package rtc

import (
	"testing"

	"github.com/jiyeyuran/mediasoup/internal/rtc/codecs"
	"github.com/stretchr/testify/require"
)

const SendNackDelay = 0 // In ms.

type TestNackGeneratorInput struct {
	seq              uint16
	isKeyFrame       bool
	firstNacked      uint16
	numNacked        int
	keyFrameRequired bool
	nackListSize     int
}

type TestPayloadDescriptorHandler struct {
	isKeyFrame bool
}

func NewTestPayloadDescriptorHandler(isKeyFrame bool) *TestPayloadDescriptorHandler {
	return &TestPayloadDescriptorHandler{isKeyFrame: isKeyFrame}
}

func (h *TestPayloadDescriptorHandler) Dump() {}

func (h *TestPayloadDescriptorHandler) Process(context *codecs.EncodingContext, data []byte) (bool, bool) {
	return true, true
}

func (h *TestPayloadDescriptorHandler) Restore(data []byte) {}

func (h *TestPayloadDescriptorHandler) GetSpatialLayer() uint8 {
	return 0
}

func (h *TestPayloadDescriptorHandler) GetTemporalLayer() uint8 {
	return 0
}

func (h *TestPayloadDescriptorHandler) IsKeyFrame() bool {
	return h.isKeyFrame
}

type TestNackGeneratorListener struct {
	t                         *testing.T
	nackRequiredTriggered     bool
	keyFrameRequiredTriggered bool
	currentInput              TestNackGeneratorInput
}

func (listener *TestNackGeneratorListener) OnNackGeneratorNackRequired(seqNumbers []uint16) {
	listener.nackRequiredTriggered = true

	firstNacked := seqNumbers[0]
	numNacked := len(seqNumbers)

	require.Equal(listener.t, listener.currentInput.firstNacked, firstNacked)
	require.Equal(listener.t, listener.currentInput.numNacked, numNacked)
}

func (listener *TestNackGeneratorListener) OnNackGeneratorKeyFrameRequired() {
	listener.keyFrameRequiredTriggered = true
	require.True(listener.t, listener.currentInput.keyFrameRequired)
}

func (listener *TestNackGeneratorListener) Reset(input TestNackGeneratorInput) {
	listener.currentInput = input
	listener.nackRequiredTriggered = false
	listener.keyFrameRequiredTriggered = false
}

func (listener *TestNackGeneratorListener) Check(t *testing.T, nackGenerator *NackGenerator) {
	require.Equal(t, listener.nackRequiredTriggered, listener.currentInput.numNacked > 0)
	require.Equal(t, listener.keyFrameRequiredTriggered, listener.currentInput.keyFrameRequired)
}

func TestNackGenerator(t *testing.T) {
	validate := func(t *testing.T, inputs []TestNackGeneratorInput) {
		listener := &TestNackGeneratorListener{
			t: t,
		}
		nackGenerator := NewNackGenerator(listener, SendNackDelay)

		for _, input := range inputs {
			listener.Reset(input)

			h := NewTestPayloadDescriptorHandler(input.isKeyFrame)
			packet := &RtpPacket{}
			packet.Unmarshal([]byte{0x80, 0x7b, 0x52, 0x0e, 0x5b, 0x6b, 0xca, 0xb5, 0x00, 0x00, 0x00, 0x02})

			packet.SetPayloadDescriptorHandler(h)

			packet.SequenceNumber = input.seq

			nackGenerator.ReceivePacket(packet, false)

			listener.Check(t, nackGenerator)
			require.Equal(t, input.nackListSize, nackGenerator.GetNackListForTest().Len())
		}

		nackGenerator.Close()
	}

	t.Run("no NACKs required", func(t *testing.T) {
		inputs := []TestNackGeneratorInput{
			{2371, false, 0, 0, false, 0},
			{2372, false, 0, 0, false, 0},
			{2373, false, 0, 0, false, 0},
			{2374, false, 0, 0, false, 0},
			{2375, false, 0, 0, false, 0},
			{2376, false, 0, 0, false, 0},
			{2377, false, 0, 0, false, 0},
			{2378, false, 0, 0, false, 0},
			{2379, false, 0, 0, false, 0},
			{2380, false, 0, 0, false, 0},
			{2254, false, 0, 0, false, 0},
			{2250, false, 0, 0, false, 0},
		}
		validate(t, inputs)
	})

	t.Run("generate NACK for missing ordered packet", func(t *testing.T) {
		inputs := []TestNackGeneratorInput{
			{2381, false, 0, 0, false, 0},
			{2383, false, 2382, 1, false, 1},
		}
		validate(t, inputs)
	})

	t.Run("sequence wrap generates no NACK", func(t *testing.T) {
		inputs := []TestNackGeneratorInput{
			{65534, false, 0, 0, false, 0},
			{65535, false, 0, 0, false, 0},
			{0, false, 0, 0, false, 0},
		}
		validate(t, inputs)
	})

	t.Run("generate NACK after sequence wrap", func(t *testing.T) {
		inputs := []TestNackGeneratorInput{
			{65534, false, 0, 0, false, 0},
			{65535, false, 0, 0, false, 0},
			{1, false, 0, 1, false, 1},
		}
		validate(t, inputs)
	})

	t.Run("generate NACK after sequence wrap, and yet another NACK", func(t *testing.T) {
		inputs := []TestNackGeneratorInput{
			{65534, false, 0, 0, false, 0},
			{65535, false, 0, 0, false, 0},
			{1, false, 0, 1, false, 1},
			{11, false, 2, 9, false, 10},
			{12, true, 0, 0, false, 10},
			{13, true, 0, 0, false, 10},
		}
		validate(t, inputs)
	})

	t.Run("intercalated missing packets", func(t *testing.T) {
		inputs := []TestNackGeneratorInput{
			{1, false, 0, 0, false, 0},
			{3, false, 2, 1, false, 1},
			{5, false, 4, 1, false, 2},
			{7, false, 6, 1, false, 3},
			{9, false, 8, 1, false, 4},
		}
		validate(t, inputs)
	})

	t.Run("non contiguous intercalated missing packets", func(t *testing.T) {
		inputs := []TestNackGeneratorInput{
			{1, false, 0, 0, false, 0},
			{3, false, 2, 1, false, 1},
			{7, false, 4, 3, false, 4},
			{9, false, 8, 1, false, 5},
		}
		validate(t, inputs)
	})

	t.Run("big jump", func(t *testing.T) {
		inputs := []TestNackGeneratorInput{
			{1, false, 0, 0, false, 0},
			{300, false, 2, 298, false, 298},
			{3, false, 0, 0, false, 297},
			{4, false, 0, 0, false, 296},
			{5, false, 0, 0, false, 295},
		}
		validate(t, inputs)
	})

	t.Run("Key Frame required. Nack list too large to be requested", func(t *testing.T) {
		inputs := []TestNackGeneratorInput{
			{1, false, 0, 0, false, 0},
			{3000, false, 0, 0, true, 0},
		}
		validate(t, inputs)
	})
}

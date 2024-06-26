package rtc

import (
	"github.com/jiyeyuran/mediasoup/internal/rtc/codecs"
	"github.com/pion/rtp"
)

type RtpPacket struct {
	rtp.Packet
	Size                     uint64
	payloadDescriptorHandler codecs.PayloadDescriptorHandler
}

func (p *RtpPacket) SetPayloadDescriptorHandler(handler codecs.PayloadDescriptorHandler) {
	p.payloadDescriptorHandler = handler
}

func (p *RtpPacket) ProcessPayload(context *codecs.EncodingContext, data []byte) (marker, ok bool) {
	if p.payloadDescriptorHandler != nil {
		return p.payloadDescriptorHandler.Process(context, data)
	}
	return
}

func (p *RtpPacket) RestorePayload() {
	if p.payloadDescriptorHandler != nil {
		p.payloadDescriptorHandler.Restore(p.Payload)
	}
}

func (p RtpPacket) GetSequenceNumber() uint16 {
	return p.SequenceNumber
}

func (p RtpPacket) IsKeyFrame() bool {
	if p.payloadDescriptorHandler != nil {
		return p.payloadDescriptorHandler.IsKeyFrame()
	}
	return false
}

func (p RtpPacket) GetSsrc() uint32 {
	// Implement this method
	return p.SSRC
}

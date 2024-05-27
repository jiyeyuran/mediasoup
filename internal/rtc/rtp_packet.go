package rtc

type RtpPacket struct {
	Size uint64
}

func (p *RtpPacket) GetSequenceNumber() uint16 {
	// Implement this method
	return 0
}

func (p *RtpPacket) IsKeyFrame() bool {
	// Implement this method
	return false
}

func (p *RtpPacket) GetSsrc() uint32 {
	// Implement this method
	return 0
}

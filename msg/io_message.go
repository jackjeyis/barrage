package msg

type ProtocolMessage struct {
	channel Channel
}

func NewProtocolMessage(ch Channel) *ProtocolMessage {
	return &ProtocolMessage{
		channel: ch,
	}
}

func (m *ProtocolMessage) Type() uint8 {
	return 0x00
}

func (m *ProtocolMessage) Serve(msg Message) {
	m.channel.Serve(msg)
}

func (m *ProtocolMessage) EncodeMessage(msg Message) {
	m.channel.EncodeMessage(msg)
}

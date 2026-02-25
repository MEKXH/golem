package bus

// MessageBus handles message routing between channels and agent
type MessageBus struct {
	inbound  chan *InboundMessage
	outbound chan *OutboundMessage
}

// NewMessageBus creates a new message bus
func NewMessageBus(bufferSize int) *MessageBus {
	return &MessageBus{
		inbound:  make(chan *InboundMessage, bufferSize),
		outbound: make(chan *OutboundMessage, bufferSize),
	}
}

// PublishInbound sends a message to the agent
func (b *MessageBus) PublishInbound(msg *InboundMessage) {
	b.inbound <- msg
}

// Inbound returns the inbound channel for consuming
func (b *MessageBus) Inbound() <-chan *InboundMessage {
	return b.inbound
}

// PublishOutbound sends a message to channels
func (b *MessageBus) PublishOutbound(msg *OutboundMessage) {
	b.outbound <- msg
}

// Outbound returns the outbound channel for consuming
func (b *MessageBus) Outbound() <-chan *OutboundMessage {
	return b.outbound
}

// Close closes both channels
func (b *MessageBus) Close() {
	close(b.inbound)
	close(b.outbound)
}

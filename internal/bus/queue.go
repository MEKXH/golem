package bus

// MessageBus 负责管理通道与 Agent 之间的消息路由。
type MessageBus struct {
	inbound  chan *InboundMessage  // 入站消息队列，Agent 从此消费
	outbound chan *OutboundMessage // 出站消息队列，外部通道从此消费
}

// NewMessageBus 创建一个新的消息总线实例。
func NewMessageBus(bufferSize int) *MessageBus {
	return &MessageBus{
		inbound:  make(chan *InboundMessage, bufferSize),
		outbound: make(chan *OutboundMessage, bufferSize),
	}
}

// PublishInbound 将消息发布到入站队列中供 Agent 处理。
func (b *MessageBus) PublishInbound(msg *InboundMessage) {
	b.inbound <- msg
}

// Inbound 返回入站消息的只读通道。
func (b *MessageBus) Inbound() <-chan *InboundMessage {
	return b.inbound
}

// PublishOutbound 将消息发布到出站队列中供外部通道消费。
func (b *MessageBus) PublishOutbound(msg *OutboundMessage) {
	b.outbound <- msg
}

// Outbound 返回出站消息的只读通道。
func (b *MessageBus) Outbound() <-chan *OutboundMessage {
	return b.outbound
}

// Close 关闭消息总线中的所有通道。
func (b *MessageBus) Close() {
	close(b.inbound)
	close(b.outbound)
}

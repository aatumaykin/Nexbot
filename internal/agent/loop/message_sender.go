package loop

import (
	"github.com/aatumaykin/nexbot/internal/agent"
	"github.com/aatumaykin/nexbot/internal/bus"
)

// AgentMessageSender implements agent.MessageSender through the message bus.
// This bridges the Agent Layer's MessageSender interface with the Bus Layer.
type AgentMessageSender struct {
	messageBus *bus.MessageBus
}

// NewAgentMessageSender creates a new AgentMessageSender instance.
func NewAgentMessageSender(messageBus *bus.MessageBus) *AgentMessageSender {
	return &AgentMessageSender{
		messageBus: messageBus,
	}
}

// SendMessage sends a message through the message bus.
// Implements agent.MessageSender interface.
func (a *AgentMessageSender) SendMessage(userID, channelType, sessionID, message string) error {
	event := bus.NewOutboundMessage(
		bus.ChannelType(channelType),
		userID,
		sessionID,
		message,
		nil, // no metadata
	)
	return a.messageBus.PublishOutbound(*event)
}

var _ agent.MessageSender = (*AgentMessageSender)(nil) // Compile-time interface check

package notification

import (
	"context"

	"github.com/fabric8-services/fabric8-wit/notification"
)

// FakeNotificationChannel is a simple Sender impl that records the notifications for later verification
type FakeNotificationChannel struct {
	Messages []notification.Message
}

// Send records each sent message in Messages
func (s *FakeNotificationChannel) Send(ctx context.Context, msg notification.Message) {
	s.Messages = append(s.Messages, msg)
}

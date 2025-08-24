package expo

import (
	"context"
	"time"
)

// PushClient defines the interface for sending push notifications
type PushClient interface {
	// PublishSingle sends a single push notification
	PublishSingle(ctx context.Context, msg *Message) ([]*MessageResponse, error)
	
	// Publish sends multiple push notifications at once
	Publish(ctx context.Context, msgs []*Message) ([]*MessageResponse, error)
	
	// GetPushReceipts fetches push receipts for the given ticket IDs
	GetPushReceipts(ctx context.Context, ticketIDs []string) (map[string]*PushReceipt, error)
	
	// SendPushNotificationsWithReceipts sends push notifications and waits for receipts
	SendPushNotificationsWithReceipts(ctx context.Context, messages []*Message, receiptDelay time.Duration) ([]*PushResult, error)
}

// Ensure Client implements PushClient interface
var _ PushClient = (*Client)(nil)

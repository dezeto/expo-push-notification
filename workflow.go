package expo

import (
	"context"
	"fmt"
	"time"
)

// PushResult contains the complete result of sending a push notification
type PushResult struct {
	TicketID    string
	Message     *Message
	PushTicket  *MessageResponse
	PushReceipt *PushReceipt
	Error       error
}

// IsSuccessful returns true if the push was successful (ticket OK and receipt OK)
func (r *PushResult) IsSuccessful() bool {
	return r.Error == nil &&
		r.PushTicket != nil && r.PushTicket.IsOk() &&
		r.PushReceipt != nil && r.PushReceipt.IsOk()
}

// ShouldRetryToken returns true if this token should be retried later
func (r *PushResult) ShouldRetryToken() bool {
	if r.PushReceipt != nil && r.PushReceipt.Details != nil {
		errorType := r.PushReceipt.Details["error"]
		// Don't retry for DeviceNotRegistered or permanent errors
		return errorType != string(ErrorMsgDeviceNotRegistered)
	}
	return r.Error != nil
}

// SendPushNotificationsWithReceipts sends push notifications and waits for receipts
// This implements the complete workflow recommended by Expo documentation
func (c *Client) SendPushNotificationsWithReceipts(ctx context.Context, messages []*Message, receiptDelay time.Duration) ([]*PushResult, error) {
	// Step 1: Send push notifications
	responses, err := c.Publish(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to send push notifications: %w", err)
	}

	// Step 2: Collect successful ticket IDs
	var ticketIDs []string
	results := make([]*PushResult, len(responses))

	for i, response := range responses {
		result := &PushResult{
			Message:    response.MessageItem,
			PushTicket: response,
		}

		if response.IsOk() && response.ID != "" {
			result.TicketID = response.ID
			ticketIDs = append(ticketIDs, response.ID)
		} else {
			result.Error = fmt.Errorf("push ticket error: %s", response.Message)
		}

		results[i] = result
	}

	// Step 3: Wait for receipts (recommended: 15 minutes)
	if receiptDelay == 0 {
		receiptDelay = 15 * time.Minute
	}

	if len(ticketIDs) == 0 {
		return results, nil
	}

	// Wait for receipts to be available
	select {
	case <-ctx.Done():
		return results, ctx.Err()
	case <-time.After(receiptDelay):
		// Continue to fetch receipts
	}

	// Step 4: Fetch push receipts
	receipts, err := c.GetPushReceipts(ctx, ticketIDs)
	if err != nil {
		return results, fmt.Errorf("failed to fetch push receipts: %w", err)
	}

	// Step 5: Match receipts to results
	for _, result := range results {
		if result.TicketID != "" {
			if receipt, exists := receipts[result.TicketID]; exists {
				result.PushReceipt = receipt

				// Check for specific errors in receipts
				if !receipt.IsOk() {
					result.Error = fmt.Errorf("push receipt error: %s", receipt.Message)
				}
			}
		}
	}

	return results, nil
}

// ValidateMessage validates a message according to Expo requirements
func ValidateMessage(msg *Message) error {
	if len(msg.To) == 0 {
		return fmt.Errorf("message must have at least one recipient")
	}

	// Check payload size (rough estimate - actual calculation would be more complex)
	// The documentation mentions 4096 bytes maximum
	estimatedSize := len(msg.Title) + len(msg.Body)
	if msg.Data != nil {
		for k, v := range msg.Data {
			estimatedSize += len(k) + len(v)
		}
	}
	if estimatedSize > 4000 { // Leave some buffer for JSON structure
		return fmt.Errorf("message payload too large (estimated %d bytes, maximum ~4000)", estimatedSize)
	}

	// Validate tokens
	for _, token := range msg.To {
		if !IsPushTokenValid(string(*token)) {
			return fmt.Errorf("invalid push token: %s", *token)
		}
	}

	return nil
}

// FilterInvalidTokens removes invalid tokens from messages and returns the count of removed tokens
func FilterInvalidTokens(messages []*Message) int {
	var removedCount int

	for _, msg := range messages {
		var validTokens []*Token
		for _, token := range msg.To {
			if IsPushTokenValid(string(*token)) {
				validTokens = append(validTokens, token)
			} else {
				removedCount++
			}
		}
		msg.To = validTokens
	}

	return removedCount
}

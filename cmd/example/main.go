// Package main provides examples of how to use the improved Expo Push Service client
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	expo "dezeto/expo-push-notification"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load("../../.env"); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	// Example 1: Basic usage with improved features
	basicExample()

	// Example 2: Complete workflow with receipt checking
	completeWorkflowExample()

	// Example 3: Handling different types of errors
	errorHandlingExample()
}

func basicExample() {
	fmt.Println("=== Basic Example ===")

	// Get access token from environment variable
	accessToken := os.Getenv("EXPO_ACCESS_TOKEN")
	if accessToken == "" {
		accessToken = "your-access-token-here" // fallback
		log.Println("Warning: EXPO_ACCESS_TOKEN not set, using placeholder")
	}

	// Create client with enhanced configuration
	client := expo.NewClient(
		expo.WithAccessToken(accessToken), // Use token from environment
		expo.WithGzipEnabled(true),        // Enable compression
		expo.WithRetryConfig(&expo.RetryConfig{ // Custom retry logic
			MaxRetries:      3,
			InitialInterval: 2 * time.Second,
			MaxInterval:     30 * time.Second,
			Multiplier:      2.0,
		}),
	)

	// Create message with enhanced fields
	token := expo.MustParseToken("ExponentPushToken[xxxxxxxxxxxxxxxxxxxxxx]")
	msg := &expo.Message{
		To:       []*expo.Token{token},
		Title:    "Hello World!",
		Body:     "This is a test notification",
		Sound:    "default",
		Badge:    1,
		Priority: expo.HighPriority,
		TTL:      3600, // 1 hour
		Data:     expo.Data{"customKey": "customValue"},

		// iOS-specific fields
		Subtitle:          "Test Subtitle",
		InterruptionLevel: "active",
		MutableContent:    true,

		// Android-specific fields
		ChannelID: "default",
		Icon:      "notification_icon",

		// Rich content support
		RichContent: map[string]string{
			"image": "https://example.com/notification-image.png",
		},
	}

	ctx := context.Background()
	responses, err := client.PublishSingle(ctx, msg)
	if err != nil {
		log.Printf("Error sending notification: %v", err)
		return
	}

	for _, response := range responses {
		if response.IsOk() {
			fmt.Printf("✅ Notification sent successfully. Ticket ID: %s\n", response.ID)
		} else {
			fmt.Printf("❌ Failed to send notification: %s\n", response.Message)
		}
	}
}

func completeWorkflowExample() {
	fmt.Println("\n=== Complete Workflow Example ===")

	// Get access token from environment variable
	accessToken := os.Getenv("EXPO_ACCESS_TOKEN")
	if accessToken == "" {
		accessToken = "your-access-token-here" // fallback
		log.Println("Warning: EXPO_ACCESS_TOKEN not set, using placeholder")
	}

	client := expo.NewClient(
		expo.WithAccessToken(accessToken),
		expo.WithGzipEnabled(true),
	)

	// Create multiple messages
	messages := []*expo.Message{
		{
			To:    []*expo.Token{expo.MustParseToken("ExponentPushToken[token1]")},
			Title: "Message 1",
			Body:  "First notification",
		},
		{
			To:    []*expo.Token{expo.MustParseToken("ExponentPushToken[token2]")},
			Title: "Message 2",
			Body:  "Second notification",
		},
	}

	ctx := context.Background()

	// Use the complete workflow that handles both tickets and receipts
	results, err := client.SendPushNotificationsWithReceipts(ctx, messages, 15*time.Minute)
	if err != nil {
		log.Printf("Error in push workflow: %v", err)
		return
	}

	// Process results
	for i, result := range results {
		fmt.Printf("Result %d:\n", i+1)
		if result.IsSuccessful() {
			fmt.Printf("  ✅ Successfully delivered\n")
		} else if result.ShouldRetryToken() {
			fmt.Printf("  🔄 Should retry later: %v\n", result.Error)
		} else {
			fmt.Printf("  ❌ Failed (don't retry): %v\n", result.Error)
		}
	}
}

func errorHandlingExample() {
	fmt.Println("\n=== Error Handling Example ===")

	// Get access token from environment variable
	accessToken := os.Getenv("EXPO_ACCESS_TOKEN")
	if accessToken == "" {
		accessToken = "your-access-token-here" // fallback
		log.Println("Warning: EXPO_ACCESS_TOKEN not set, using placeholder")
	}

	client := expo.NewClient(
		expo.WithAccessToken(accessToken),
	)

	ctx := context.Background()

	// Simulate getting receipts with various error conditions
	ticketIDs := []string{"receipt-id-1", "receipt-id-2", "receipt-id-3"}
	receipts, err := client.GetPushReceipts(ctx, ticketIDs)
	if err != nil {
		log.Printf("Error fetching receipts: %v", err)
		return
	}

	for ticketID, receipt := range receipts {
		fmt.Printf("Receipt %s: ", ticketID)

		if receipt.IsOk() {
			fmt.Println("✅ Delivered successfully")
		} else if receipt.IsDeviceNotRegistered() {
			fmt.Println("🚫 Device not registered - remove token from database")
		} else if receipt.Details != nil {
			errorType := receipt.Details["error"]
			switch errorType {
			case string(expo.ErrorMsgTooBig):
				fmt.Println("📦 Message too big - reduce payload size")
			case string(expo.ErrorMsgRateExceeded):
				fmt.Println("⏱️  Rate exceeded - implement backoff")
			case string(expo.ErrorMsgMismatchSenderID):
				fmt.Println("🔑 FCM credentials mismatch - check configuration")
			case string(expo.ErrorMsgInvalidCredentials):
				fmt.Println("🔐 Invalid credentials - regenerate certificates")
			default:
				fmt.Printf("❌ Unknown error: %s\n", receipt.Message)
			}
		}
	}
}

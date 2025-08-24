# Expo Push Notification Go Client

A comprehensive Go client library for sending push notifications using the Expo Push Notification service. This library provides enhanced features including retry logic, gzip compression, receipt checking, and robust error handling.

## Features

- ✅ **Send single or multiple push notifications**
- ✅ **Automatic retry logic with exponential backoff**
- ✅ **Gzip compression support**
- ✅ **Receipt checking and validation**
- ✅ **Comprehensive error handling**
- ✅ **Environment variable configuration**
- ✅ **Rich notification content support**
- ✅ **iOS and Android specific features**
- ✅ **Token validation and parsing**

## Installation

```bash
go get dezeto/expo-push-notification
```

## Quick Start

### Basic Setup

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    expo "dezeto/expo-push-notification"
)

func main() {
    // Create a new client
    client := expo.NewClient(
        expo.WithAccessToken("your-expo-access-token"),
        expo.WithGzipEnabled(true),
    )
    
    // Parse a push token
    token := expo.MustParseToken("ExponentPushToken[xxxxxxxxxxxxxxxxxxxxxx]")
    
    // Create a message
    message := &expo.Message{
        To:    []*expo.Token{token},
        Title: "Hello World!",
        Body:  "This is a test notification",
        Sound: "default",
        Badge: 1,
    }
    
    // Send the notification
    ctx := context.Background()
    responses, err := client.PublishSingle(ctx, message)
    if err != nil {
        log.Fatal(err)
    }
    
    for _, response := range responses {
        if response.IsOk() {
            fmt.Printf("✅ Notification sent! Ticket ID: %s\n", response.ID)
        } else {
            fmt.Printf("❌ Failed to send: %s\n", response.Message)
        }
    }
}
```

### Environment Configuration

Copy the sample environment file and configure your settings:

```bash
cp .env.sample .env
```

Then edit `.env` with your actual values:

```env
EXPO_ACCESS_TOKEN=your-expo-access-token-here
```

Then use it in your code:

```go
package main

import (
    "log"
    "os"
    
    "github.com/joho/godotenv"
    expo "dezeto/expo-push-notification"
)

func main() {
    // Load environment variables
    if err := godotenv.Load(); err != nil {
        log.Printf("Warning: Could not load .env file: %v", err)
    }
    
    // Create client with environment token
    accessToken := os.Getenv("EXPO_ACCESS_TOKEN")
    client := expo.NewClient(expo.WithAccessToken(accessToken))
    
    // ... rest of your code
}
```

## Configuration Options

The client supports various configuration options:

```go
client := expo.NewClient(
    expo.WithAccessToken("your-token"),
    expo.WithGzipEnabled(true),
    expo.WithRetryConfig(&expo.RetryConfig{
        MaxRetries:      3,
        InitialInterval: 2 * time.Second,
        MaxInterval:     30 * time.Second,
        Multiplier:      2.0,
    }),
    expo.WithHTTPClient(customHttpClient),
)
```

## Message Options

### Basic Message

```go
message := &expo.Message{
    To:    []*expo.Token{token},
    Title: "Hello",
    Body:  "World",
}
```

### Advanced Message with Platform-Specific Features

```go
message := &expo.Message{
    To:       []*expo.Token{token},
    Title:    "Advanced Notification",
    Body:     "This notification has advanced features",
    Sound:    "default",
    Badge:    1,
    Priority: expo.HighPriority,
    TTL:      3600, // 1 hour
    Data:     expo.Data{"customKey": "customValue"},
    
    // iOS-specific
    Subtitle:          "iOS Subtitle",
    InterruptionLevel: "active",
    MutableContent:    true,
    
    // Android-specific
    ChannelID: "default",
    Icon:      "notification_icon",
    
    // Rich content
    RichContent: map[string]string{
        "image": "https://example.com/image.png",
    },
}
```

## Sending Multiple Notifications

```go
messages := []*expo.Message{
    {
        To:    []*expo.Token{token1},
        Title: "Message 1",
        Body:  "First notification",
    },
    {
        To:    []*expo.Token{token2},
        Title: "Message 2",
        Body:  "Second notification",
    },
}

responses, err := client.Publish(ctx, messages)
```

## Complete Workflow with Receipt Checking

```go
// Send notifications and automatically check receipts
results, err := client.SendPushNotificationsWithReceipts(ctx, messages, 15*time.Minute)
if err != nil {
    log.Printf("Error: %v", err)
    return
}

for i, result := range results {
    if result.IsSuccessful() {
        fmt.Printf("✅ Message %d delivered successfully\n", i+1)
    } else if result.ShouldRetryToken() {
        fmt.Printf("🔄 Message %d should be retried: %v\n", i+1, result.Error)
    } else {
        fmt.Printf("❌ Message %d failed permanently: %v\n", i+1, result.Error)
    }
}
```

## Error Handling

The library provides comprehensive error handling:

```go
receipts, err := client.GetPushReceipts(ctx, ticketIDs)
if err != nil {
    log.Printf("Error fetching receipts: %v", err)
    return
}

for ticketID, receipt := range receipts {
    if receipt.IsOk() {
        fmt.Printf("✅ %s: Delivered\n", ticketID)
    } else if receipt.IsDeviceNotRegistered() {
        fmt.Printf("🚫 %s: Device not registered - remove token\n", ticketID)
    } else if receipt.Details != nil {
        switch receipt.Details["error"] {
        case string(expo.ErrorMsgTooBig):
            fmt.Printf("📦 %s: Message too big\n", ticketID)
        case string(expo.ErrorMsgRateExceeded):
            fmt.Printf("⏱️  %s: Rate exceeded\n", ticketID)
        case string(expo.ErrorMsgMismatchSenderID):
            fmt.Printf("🔑 %s: FCM credentials mismatch\n", ticketID)
        case string(expo.ErrorMsgInvalidCredentials):
            fmt.Printf("🔐 %s: Invalid credentials\n", ticketID)
        }
    }
}
```

## Token Validation

```go
// Parse and validate tokens
token, err := expo.ParseToken("ExponentPushToken[xxxxxxxxxxxxxxxxxxxxxx]")
if err != nil {
    log.Printf("Invalid token: %v", err)
    return
}

// Or use MustParseToken for tokens you know are valid
token := expo.MustParseToken("ExponentPushToken[xxxxxxxxxxxxxxxxxxxxxx]")
```

## API Reference

### Client Methods

- `NewClient(opts ...Option) *Client` - Create a new client
- `PublishSingle(ctx, message) ([]*MessageResponse, error)` - Send a single notification
- `Publish(ctx, messages) ([]*MessageResponse, error)` - Send multiple notifications
- `GetPushReceipts(ctx, ticketIDs) (map[string]*PushReceipt, error)` - Get delivery receipts
- `SendPushNotificationsWithReceipts(ctx, messages, timeout) ([]*NotificationResult, error)` - Complete workflow

### Configuration Options

- `WithAccessToken(token string)` - Set Expo access token
- `WithGzipEnabled(enabled bool)` - Enable/disable gzip compression
- `WithRetryConfig(config *RetryConfig)` - Configure retry behavior
- `WithHTTPClient(client *http.Client)` - Use custom HTTP client

### Error Types

- `ErrorMsgDeviceNotRegistered` - Token is invalid, remove from database
- `ErrorMsgTooBig` - Message exceeds 4KB limit
- `ErrorMsgRateExceeded` - Too many messages sent too quickly
- `ErrorMsgMismatchSenderID` - FCM configuration issue
- `ErrorMsgInvalidCredentials` - Invalid push credentials

## Running the Example

1. Clone the repository
2. Copy `.env.sample` to `.env` and add your Expo access token:
   ```bash
   cp .env.sample .env
   ```
3. Edit `.env` and replace `your-expo-access-token-here` with your actual token
4. Run the example:

```bash
cd cmd/example
go run main.go
```

## Getting an Expo Access Token

1. Create an account at [expo.dev](https://expo.dev)
2. Create a new project or select an existing one
3. Go to your project settings
4. Generate an access token in the "Access Tokens" section
5. Add the token to your `.env` file

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For issues related to this library, please create an issue on GitHub.
For Expo-specific questions, refer to the [Expo Push Notification documentation](https://docs.expo.dev/push-notifications/overview/).

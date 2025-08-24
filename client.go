package expo

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type Client struct {
	cnf *Config
}

func NewClient(opts ...Option) *Client {
	c := &Config{}
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
	withDefaults(c)
	return &Client{c}
}

// Publish sends a single push notification
// @param msg: A Message object
// @return an array of MessageResponse objects which contains the results.
// @return error if any requests failed
func (c *Client) PublishSingle(ctx context.Context, msg *Message) ([]*MessageResponse, error) {
	responses, err := c.publish(ctx, []*Message{msg})
	if err != nil {
		return nil, err
	}
	return responses, nil
}

// PublishMultiple sends multiple push notifications at once
// @param msgs: An array of Message objects.
// @return an array of MessageResponse objects which contains the results.
// @return error if the request failed
func (c *Client) Publish(ctx context.Context, msgs []*Message) ([]*MessageResponse, error) {
	return c.publish(ctx, msgs)
}

func (c *Client) publish(ctx context.Context, msgs []*Message) ([]*MessageResponse, error) {
	// Validate the messages
	for _, message := range msgs {
		if len(message.To) == 0 {
			return nil, errors.New("no recipients")
		}
		for _, recipient := range message.To {
			if recipient == nil || *recipient == "" {
				return nil, errors.New("invalid push token")
			}
		}
	}

	// Limit to 100 notifications per request as per Expo documentation
	const maxNotificationsPerRequest = 100
	if len(msgs) > maxNotificationsPerRequest {
		return nil, fmt.Errorf("too many notifications: %d (maximum is %d)", len(msgs), maxNotificationsPerRequest)
	}

	url := fmt.Sprintf("%s%s/push/send", c.cnf.Host, c.cnf.ApiURL)
	jsonBytes, err := json.Marshal(msgs)
	if err != nil {
		return nil, err
	}

	// Apply gzip compression if enabled
	var requestBody []byte = jsonBytes
	if c.cnf.EnableGzip {
		var buf bytes.Buffer
		gzWriter := gzip.NewWriter(&buf)
		if _, err := gzWriter.Write(jsonBytes); err != nil {
			return nil, err
		}
		if err := gzWriter.Close(); err != nil {
			return nil, err
		}
		requestBody = buf.Bytes()
	}

	// Use retry logic for the HTTP request
	resp, err := c.WithRetry(ctx, c.cnf.RetryConfig, func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
		if err != nil {
			return nil, err
		}

		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Accept", "application/json")
		req.Header.Add("Accept-Encoding", "gzip, deflate")

		if c.cnf.EnableGzip {
			req.Header.Add("Content-Encoding", "gzip")
		}

		if c.cnf.AccessToken != "" {
			req.Header.Add("Authorization", "Bearer "+c.cnf.AccessToken)
		}

		return c.cnf.HttpClient.Do(req)
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err = checkStatus(resp); err != nil {
		return nil, err
	}

	var r *Response
	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		return nil, err
	}
	if r.Errors != nil {
		return nil, errors.New("invalid request")
	}
	if r.Data == nil {
		return nil, NewServerError("invalid server response", resp, r, nil)
	}

	// Expand the messages to match the API's response structure
	var expandedMessages []*Message
	for _, msg := range msgs {
		for range msg.To {
			expandedMessages = append(expandedMessages, msg)
		}
	}

	if len(expandedMessages) != len(r.Data) {
		errMsg := fmt.Sprintf("mismatched response length. Expected %d receipts but only received %d", len(expandedMessages), len(r.Data))
		return nil, NewServerError(errMsg, resp, r, nil)
	}
	// data will contain an array of push tickets in the same order in which the messages were sent
	// assign each response to its corresponding message
	for i := range r.Data {
		r.Data[i].MessageItem = expandedMessages[i]
	}
	return r.Data, nil
}

// GetPushReceipts fetches push receipts for the given ticket IDs
// @param ctx: Context for the request
// @param ticketIDs: Array of ticket IDs from previous push responses
// @return map of ticket ID to PushReceipt
// @return error if the request failed
func (c *Client) GetPushReceipts(ctx context.Context, ticketIDs []string) (map[string]*PushReceipt, error) {
	if len(ticketIDs) == 0 {
		return make(map[string]*PushReceipt), nil
	}

	// The API accepts maximum 1000 receipt IDs per request
	const maxReceiptsPerRequest = 1000
	if len(ticketIDs) > maxReceiptsPerRequest {
		return nil, fmt.Errorf("too many ticket IDs: %d (maximum is %d)", len(ticketIDs), maxReceiptsPerRequest)
	}

	url := fmt.Sprintf("%s%s/push/getReceipts", c.cnf.Host, c.cnf.ApiURL)
	reqBody := &PushReceiptRequest{IDs: ticketIDs}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	if c.cnf.AccessToken != "" {
		req.Header.Add("Authorization", "Bearer "+c.cnf.AccessToken)
	}

	resp, err := c.cnf.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err = checkStatus(resp); err != nil {
		return nil, err
	}

	var receiptResp *PushReceiptResponse
	err = json.NewDecoder(resp.Body).Decode(&receiptResp)
	if err != nil {
		return nil, err
	}

	if receiptResp.Errors != nil {
		return nil, NewServerError("error fetching receipts", resp, nil, receiptResp.Errors)
	}

	return receiptResp.Data, nil
}

func checkStatus(resp *http.Response) error {
	if resp.StatusCode >= http.StatusOK && resp.StatusCode <= 299 {
		return nil
	}
	return fmt.Errorf("invalid response (%d %s)", resp.StatusCode, resp.Status)
}

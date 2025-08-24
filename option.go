package expo

import "net/http"

type Config struct {
	Host        string
	ApiURL      string
	AccessToken string
	HttpClient  *http.Client
	EnableGzip  bool
	RetryConfig *RetryConfig
}

type Option func(*Config)

func WithHost(host string) Option {
	return func(c *Config) {
		c.Host = host
	}
}

func WithApiURL(apiURL string) Option {
	return func(c *Config) {
		c.ApiURL = apiURL
	}
}

func WithAccessToken(accessToken string) Option {
	return func(c *Config) {
		c.AccessToken = accessToken
	}
}

func WithGzipEnabled(enabled bool) Option {
	return func(c *Config) {
		c.EnableGzip = enabled
	}
}

func WithRetryConfig(retryConfig *RetryConfig) Option {
	return func(c *Config) {
		c.RetryConfig = retryConfig
	}
}

func WithHttpClient(httpClient *http.Client) Option {
	return func(c *Config) {
		c.HttpClient = httpClient
	}
}

func withDefaults(c *Config) {
	if c.Host == "" {
		c.Host = "https://exp.host"
	}
	if c.ApiURL == "" {
		c.ApiURL = "/--/api/v2"
	}
	if c.HttpClient == nil {
		c.HttpClient = &http.Client{}
	}
	if c.RetryConfig == nil {
		c.RetryConfig = DefaultRetryConfig()
	}
}

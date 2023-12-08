package http_manager

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"sync"
)

type PypiHttpClient struct {
	BaseUrl string
	Client  *http.Client
}

// Join appends paths to the base URL and returns the resulting URL.
func (c *PypiHttpClient) Join(params ...string) string {
	u, _ := url.Parse(c.BaseUrl)

	// Append the provided path segments to the URL
	for _, param := range params {
		u.Path = path.Join(u.Path, param)
	}

	return u.String()
}

func (c *PypiHttpClient) Get(url string) (resp *http.Response, err error) {
	return c.Client.Get(url)
}

// HTTPClientManager manages HTTP clients for different hostnames.
type HTTPClientManager struct {
	clients map[string]*PypiHttpClient
	mu      sync.Mutex
}

// NewHTTPClientManager creates a new HTTPClientManager.
func NewHTTPClientManager() *HTTPClientManager {
	return &HTTPClientManager{
		clients: make(map[string]*PypiHttpClient),
	}
}

// GetAllBaseURLs returns a slice of all base URLs without credentials.
func (m *HTTPClientManager) GetAllBaseURLs() []*PypiHttpClient {
	m.mu.Lock()
	defer m.mu.Unlock()

	var baseURLs []*PypiHttpClient

	for _, client := range m.clients {
		baseURLs = append(baseURLs, client)
	}

	return baseURLs
}

// AddURL adds a URL with optional basic authentication to the HTTPClientManager.
func (m *HTTPClientManager) AddURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}

	// Extract the hostname from the URL
	hostname := u.Hostname()

	// Extract username and password from the URL if provided
	username := u.User.Username()
	password, _ := u.User.Password()

	// Remove credentials from the URL
	u.User = nil
	baseURLWithoutCreds := u.String()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Create a new HTTP client with basic authentication if provided
	client := &http.Client{}
	if username != "" && password != "" {
		client.Transport = &basicAuthTransport{
			Username: username,
			Password: password,
		}
	}

	// Create a new PypiHttpClient and associate it with the base URL without credentials
	m.clients[hostname] = &PypiHttpClient{
		BaseUrl: baseURLWithoutCreds,
		Client:  client,
	}

	return nil
}

// GetClient returns the HTTP client associated with a given hostname from a URL.
func (m *HTTPClientManager) GetClient(rawURL string) (*PypiHttpClient, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	// Extract the hostname from the URL
	hostname := u.Hostname()

	m.mu.Lock()
	defer m.mu.Unlock()

	PypiHttpClient, ok := m.clients[hostname]
	if !ok {
		return nil, fmt.Errorf("HTTP client for hostname %s not found", hostname)
	}

	return PypiHttpClient, nil
}

func (m *HTTPClientManager) Get(url string) (resp *http.Response, err error) {
	client, err := m.GetClient(url)
	if err != nil {
		return http.Get(url)
	} else {
		return client.Client.Get(url)
	}
}

// BasicAuthTransport is a custom HTTP transport that adds basic authentication to requests.
type basicAuthTransport struct {
	Username string
	Password string
}

// RoundTrip adds basic authentication headers to the request.
func (t *basicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(t.Username, t.Password)
	return http.DefaultTransport.RoundTrip(req)
}

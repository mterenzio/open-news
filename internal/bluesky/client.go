package bluesky

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents a Bluesky API client
type Client struct {
	baseURL    string
	httpClient *http.Client
	session    *Session
}

// Session represents an authenticated Bluesky session
type Session struct {
	AccessJWT  string `json:"accessJwt"`
	RefreshJWT string `json:"refreshJwt"`
	DID        string `json:"did"`
	Handle     string `json:"handle"`
}

// Post represents a Bluesky post
type Post struct {
	URI       string    `json:"uri"`
	CID       string    `json:"cid"`
	Author    Author    `json:"author"`
	Record    Record    `json:"record"`
	ReplyCount int      `json:"replyCount"`
	RepostCount int     `json:"repostCount"`
	LikeCount  int      `json:"likeCount"`
	IndexedAt  time.Time `json:"indexedAt"`
}

// Author represents a post author
type Author struct {
	DID         string `json:"did"`
	Handle      string `json:"handle"`
	DisplayName string `json:"displayName,omitempty"`
	Avatar      string `json:"avatar,omitempty"`
}

// Record represents the content of a post
type Record struct {
	Type      string     `json:"$type"`
	Text      string     `json:"text"`
	CreatedAt time.Time  `json:"createdAt"`
	Facets    []Facet    `json:"facets,omitempty"`
	Embed     *Embed     `json:"embed,omitempty"`
}

// Facet represents a facet in a post (links, mentions, etc.)
type Facet struct {
	Index    ByteSlice   `json:"index"`
	Features []Feature   `json:"features"`
}

// ByteSlice represents a byte range
type ByteSlice struct {
	ByteStart int `json:"byteStart"`
	ByteEnd   int `json:"byteEnd"`
}

// Feature represents a feature in a facet
type Feature struct {
	Type string `json:"$type"`
	URI  string `json:"uri,omitempty"`
	DID  string `json:"did,omitempty"`
	Tag  string `json:"tag,omitempty"`
}

// Embed represents embedded content in a post
type Embed struct {
	Type     string          `json:"$type"`
	External *ExternalEmbed  `json:"external,omitempty"`
	Images   []ImageEmbed    `json:"images,omitempty"`
}

// ExternalEmbed represents an external link embed
type ExternalEmbed struct {
	URI         string `json:"uri"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Thumb       string `json:"thumb,omitempty"`
}

// ImageEmbed represents an image embed
type ImageEmbed struct {
	Image *ImageRef `json:"image"`
	Alt   string    `json:"alt"`
}

// ImageRef represents an image reference
type ImageRef struct {
	Type string `json:"$type"`
	Ref  string `json:"ref"`
	Size int    `json:"size"`
}

// Timeline represents a timeline response
type Timeline struct {
	Feed   []Post `json:"feed"`
	Cursor string `json:"cursor,omitempty"`
}

// NewClient creates a new Bluesky client
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://bsky.social"
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateSession authenticates with Bluesky and creates a session
func (c *Client) CreateSession(identifier, password string) error {
	reqBody := map[string]string{
		"identifier": identifier,
		"password":   password,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/xrpc/com.atproto.server.createSession",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var session Session
	if err := json.Unmarshal(body, &session); err != nil {
		return err
	}

	c.session = &session
	return nil
}

// GetTimeline retrieves the authenticated user's timeline
func (c *Client) GetTimeline(limit int, cursor string) (*Timeline, error) {
	if c.session == nil {
		return nil, fmt.Errorf("not authenticated")
	}

	url := fmt.Sprintf("%s/xrpc/app.bsky.feed.getTimeline?limit=%d", c.baseURL, limit)
	if cursor != "" {
		url += "&cursor=" + cursor
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.session.AccessJWT)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get timeline: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var timeline Timeline
	if err := json.Unmarshal(body, &timeline); err != nil {
		return nil, err
	}

	return &timeline, nil
}

// GetProfile retrieves a user's profile
func (c *Client) GetProfile(handle string) (*Author, error) {
	url := fmt.Sprintf("%s/xrpc/app.bsky.actor.getProfile?actor=%s", c.baseURL, handle)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if c.session != nil {
		req.Header.Set("Authorization", "Bearer "+c.session.AccessJWT)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get profile: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var profile Author
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, err
	}

	return &profile, nil
}

// ExtractLinks extracts URLs from a post's text and embeds
func ExtractLinks(post *Post) []string {
	var links []string

	// Extract from facets
	for _, facet := range post.Record.Facets {
		for _, feature := range facet.Features {
			if feature.Type == "app.bsky.richtext.facet#link" && feature.URI != "" {
				links = append(links, feature.URI)
			}
		}
	}

	// Extract from embeds
	if post.Record.Embed != nil && post.Record.Embed.External != nil {
		links = append(links, post.Record.Embed.External.URI)
	}

	return links
}

// FollowsResponse represents the response from getFollows
type FollowsResponse struct {
	Subject string   `json:"subject"`
	Follows []Author `json:"follows"`
	Cursor  string   `json:"cursor,omitempty"`
}

// GetFollows retrieves the list of accounts a user follows
func (c *Client) GetFollows(actor string, limit int, cursor string) (*FollowsResponse, error) {
	url := fmt.Sprintf("%s/xrpc/app.bsky.graph.getFollows?actor=%s&limit=%d", c.baseURL, actor, limit)
	if cursor != "" {
		url += "&cursor=" + cursor
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if c.session != nil {
		req.Header.Set("Authorization", "Bearer "+c.session.AccessJWT)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get follows: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var follows FollowsResponse
	if err := json.Unmarshal(body, &follows); err != nil {
		return nil, err
	}

	return &follows, nil
}

package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	walog "github.com/sorens/warchiver/internal/log"
)

const maxAPIErrorBody = 64 * 1024

// Client is the Battle.net API client
type Client struct {
	httpClient  *http.Client
	region      Region
	game        GameProperty
	accessToken string
	baseURL     string
	limiter     *RateLimiter
	discovered  map[string]bool
}

// EndpointResponse describes a successful API call (failures return *APIError).
type EndpointResponse struct {
	ID   string
	Path string
}

func NewClient(options *ClientOptions) (*Client, error) {
	if options == nil {
		return nil, fmt.Errorf("client required region and access token missing")
	}
	if options.Region == "" || options.AccessToken == "" {
		return nil, fmt.Errorf("client required region and access token missing")
	}
	baseURL, err := options.Region.BaseURL()
	if err != nil {
		return nil, err
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConnsPerHost = 8 // default 2 churns under the 4-worker archive pool
	c := &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   2 * time.Minute,
		},
		region:      options.Region,
		game:        options.Game,
		baseURL:     baseURL,
		accessToken: options.AccessToken,
		limiter:     options.Limiter,
		discovered:  map[string]bool{},
	}
	if c.limiter == nil {
		c.limiter = DefaultRateLimiter()
	}

	return c, nil
}

func (c *Client) RequestEndpoint(ctx context.Context, dir string, ep *Endpoint) (*EndpointResponse, error) {
	namespace := c.endpointNamespace(ep)

	endpointDir, err := ep.ID()
	if err != nil {
		return nil, err
	}
	dst, err := c.request(ctx, dir, namespace, endpointDir, ep.Filename)
	if err != nil {
		return nil, err
	}

	id, err := ep.ID()
	if err != nil {
		return nil, err
	}

	return &EndpointResponse{
		ID:   id,
		Path: dst,
	}, nil
}

func (c *Client) RequestEndpointBytes(ctx context.Context, ep *Endpoint) ([]byte, error) {
	namespace := c.endpointNamespace(ep)
	endpointPath, err := ep.ID()
	if err != nil {
		return nil, err
	}
	return c.requestBytes(ctx, namespace, endpointPath, ep.Filename)
}

func (c *Client) endpointNamespace(ep *Endpoint) string {
	game := c.game
	if game == "" {
		game = GameRetail
	}
	if game == GameRetail {
		return fmt.Sprintf(ep.Namespace, c.region)
	}
	return fmt.Sprintf(ep.Namespace, string(game)+"-"+string(c.region))
}

func (c *Client) RequestMedia(ctx context.Context, dir string, ep *Endpoint) (*EndpointResponse, error) {
	namespace := c.endpointNamespace(ep)

	endpointDir, err := ep.ID()
	if err != nil {
		return nil, err
	}

	u, err := validateMediaURL(ep.Url)
	if err != nil {
		return nil, fmt.Errorf("invalid media URL: %w", err)
	}
	filename := filepath.Base(u.Path)

	dst, err := c.requestUrl(ctx, dir, namespace, endpointDir, u.String(), filename)
	if err != nil {
		return nil, err
	}

	id, err := ep.ID()
	if err != nil {
		return nil, err
	}

	return &EndpointResponse{
		ID:   id,
		Path: dst,
	}, nil
}

// ctx
// dir: directory where data is written
// namespace: for request
// path: for request, also used when writing data
// filename: file where data is written
func (c *Client) request(ctx context.Context, dir, namespace, path, filename string) (string, error) {
	dst := filepath.Join(dir, c.archivePath(path), filename)
	dstDir := filepath.Dir(dst)

	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %q: %w", dstDir, err)
	}

	f, err := os.Create(dst)
	if err != nil {
		walog.Ctx(ctx).Debug("file create failure: %s: %s", dst, err)
		return "", fmt.Errorf("failed creating file %q: %w", dst, err)
	}

	if _, err := c.requestInto(ctx, namespace, path, filename, dst, f); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(dst)
		return "", fmt.Errorf("failed closing file %q: %w", dst, err)
	}
	return dst, nil
}

func (c *Client) requestBytes(ctx context.Context, namespace, path, filename string) ([]byte, error) {
	var data bytes.Buffer
	if _, err := c.requestInto(ctx, namespace, path, filename, path, &data); err != nil {
		return nil, err
	}
	return data.Bytes(), nil
}

func (c *Client) requestInto(ctx context.Context, namespace, path, filename, output string, dest io.Writer) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return 0, fmt.Errorf("invalid path: %w", err)
	}
	q := u.Query()
	q.Set("namespace", namespace)
	q.Set("locale", "en_US")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", "application/json")

	waitStart := time.Now()
	if err := c.limiter.Wait(ctx); err != nil {
		return 0, fmt.Errorf("rate limiter: %w", err)
	}
	walog.Ctx(ctx).Debug("api limiter wait: duration=%s", time.Since(waitStart))
	defer c.limiter.Done()

	walog.Ctx(ctx).Verbose("request started:  %s", u.Path)
	started := time.Now()
	resp, err := c.doWithRetry(ctx, req, u.Path)
	if err != nil {
		return 0, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	header := ParseHeader(resp.Header)
	if header.BNSchemaRevision != "" {
		if _, err := strconv.ParseInt(header.BNSchemaRevision, 10, 64); err != nil {
			return 0, fmt.Errorf("failed to parse schema revision header: %w", err)
		}
	}
	if resp.StatusCode != http.StatusOK {
		walog.Ctx(ctx).Debug("request completed: path=%s status=%d latency=%s", u.Path, resp.StatusCode, time.Since(started))
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxAPIErrorBody))
		walog.Ctx(ctx).Verbose("request failed: [%d]: %s", resp.StatusCode, u.Path)
		if resp.StatusCode == http.StatusUnauthorized {
			walog.Ctx(ctx).Debug("api: 401 unauthorized: path=%s token_prefix=%s", u.Path, tokenPrefix(c.accessToken))
		}
		return 0, &APIError{StatusCode: resp.StatusCode, Path: path, Filename: filename, Body: string(body)}
	}

	bytesWritten, err := io.Copy(dest, resp.Body)
	if err != nil {
		walog.Ctx(ctx).Debug("response write failure: %s: %s", output, err)
		return bytesWritten, fmt.Errorf("failed writing response %q: %w", output, err)
	}
	walog.Ctx(ctx).Verbose("request finished: %s", output)
	walog.Ctx(ctx).Debug("request completed: path=%s status=%d latency=%s bytes=%d", u.Path, resp.StatusCode, time.Since(started), bytesWritten)
	return bytesWritten, nil
}

func (c *Client) archivePath(path string) string {
	if c.game == "" {
		return path
	}
	if strings.HasPrefix(path, "/profile/wow/") {
		return strings.Replace(path, "/profile/wow/", "/profile/wow/"+string(c.game)+"/", 1)
	}
	if strings.HasPrefix(path, "/data/wow/") {
		return strings.Replace(path, "/data/wow/", "/data/"+string(c.game)+"/", 1)
	}
	if strings.HasPrefix(path, "/profile/user/wow") {
		return strings.Replace(path, "/profile/user/wow", "/account/"+string(c.game), 1)
	}
	if strings.HasPrefix(path, "/profile/wow/protected-character/") {
		return strings.Replace(path, "/profile/wow/protected-character/", "/profile/wow/"+string(c.game)+"/protected-character/", 1)
	}
	return path
}

func (c *Client) requestUrl(ctx context.Context, dir, namespace, path, requestUrl, filename string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	_ = namespace
	dst := filepath.Join(dir, c.archivePath(path), filename)
	dstDir := filepath.Dir(dst)

	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %q: %w", dstDir, err)
	}

	u, err := validateMediaURL(requestUrl)
	if err != nil {
		return "", fmt.Errorf("invalid url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	waitStart := time.Now()
	if err := c.limiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("rate limiter: %w", err)
	}
	walog.Ctx(ctx).Debug("api limiter wait: duration=%s", time.Since(waitStart))
	defer c.limiter.Done()

	walog.Ctx(ctx).Verbose("request started:  %s", u.String())
	started := time.Now()

	mediaClient := *c.httpClient
	mediaClient.CheckRedirect = func(req *http.Request, _ []*http.Request) error {
		if _, err := validateMediaURL(req.URL.String()); err != nil {
			return fmt.Errorf("blocked media redirect: %w", err)
		}
		// Never forward credentials if a custom transport/client adds them.
		req.Header.Del("Authorization")
		return nil
	}
	resp, err := c.doWithRetryClient(ctx, &mediaClient, req, u.String())
	if err != nil {
		return "", fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		walog.Ctx(ctx).Debug("request completed: path=%s status=%d latency=%s", u.Path, resp.StatusCode, time.Since(started))
		// read all if an error
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxAPIErrorBody))
		walog.Ctx(ctx).Verbose("request failed: [%d]: %s", resp.StatusCode, requestUrl)
		if resp.StatusCode == http.StatusUnauthorized {
			walog.Ctx(ctx).Verbose("api: 401 unauthorized: url=%s token_prefix=%s", requestUrl, tokenPrefix(c.accessToken))
		}

		return "", &APIError{
			StatusCode: resp.StatusCode,
			Path:       path,
			Filename:   filename,
			Body:       string(body),
		}
	}

	f, err := os.Create(dst)
	if err != nil {
		walog.Ctx(ctx).Debug("file create failure: %s: %s", dst, err)
		return "", fmt.Errorf("failed creating file %q: %w", dst, err)
	}
	defer f.Close()

	bytesWritten, err := io.Copy(f, resp.Body)
	if err != nil {
		os.Remove(f.Name())
		walog.Ctx(ctx).Debug("file write failure: %s: %s", dst, err)
		return "", fmt.Errorf("failed writing to file %q: %w", dst, err)
	}

	walog.Ctx(ctx).Verbose("request finished: %s", dst)
	walog.Ctx(ctx).Debug("request completed: path=%s status=%d latency=%s bytes=%d", u.Path, resp.StatusCode, time.Since(started), bytesWritten)
	return dst, nil
}

func (c *Client) doWithRetry(ctx context.Context, req *http.Request, path string) (*http.Response, error) {
	return c.doWithRetryClient(ctx, c.httpClient, req, path)
}
func (c *Client) doWithRetryClient(ctx context.Context, hc *http.Client, req *http.Request, path string) (*http.Response, error) {
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
		}
		resp, err := hc.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if attempt == 3 {
				return nil, err
			}
			time.Sleep(time.Duration(100*(1<<attempt)) * time.Millisecond)
			continue
		}
		if resp.StatusCode != http.StatusTooManyRequests && (resp.StatusCode < 500 || resp.StatusCode > 599) {
			return resp, nil
		}
		if attempt == 3 {
			return resp, nil
		}
		wait := time.Duration(100*(1<<attempt)) * time.Millisecond
		if v := resp.Header.Get("Retry-After"); v != "" {
			if sec, e := strconv.Atoi(v); e == nil {
				wait = time.Duration(sec) * time.Second
			}
		}
		if wait > 30*time.Second {
			wait = 30 * time.Second
		}
		resp.Body.Close()
		wait += time.Duration(rand.Int63n(int64(100 * time.Millisecond)))
		t := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			t.Stop()
			return nil, ctx.Err()
		case <-t.C:
		}
		req = req.Clone(ctx)
	}
	return nil, fmt.Errorf("request retries exhausted: %s", path)
}

// validateMediaURL accepts only HTTPS Blizzard media origins. Media is public,
// so callers must never send the user's API bearer token to this destination.
func validateMediaURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Port() != "" && u.Port() != "443" {
		return nil, fmt.Errorf("media URL must use HTTPS without credentials or a non-HTTPS port")
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if ip, err := netip.ParseAddr(host); err == nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return nil, fmt.Errorf("private media host")
		}
		return nil, fmt.Errorf("media host is not a Blizzard origin")
	}
	if !(host == "blizzard.com" || strings.HasSuffix(host, ".blizzard.com") || host == "worldofwarcraft.com" || strings.HasSuffix(host, ".worldofwarcraft.com")) {
		return nil, fmt.Errorf("media host %q is not a Blizzard origin", host)
	}
	u.Host = host
	return u, nil
}

func (c *Client) RequestProfile(ctx context.Context, dir, path, filename string) (string, error) {
	return c.request(ctx, dir, c.ProfileNamespace(), path, filename)
}

func (c *Client) RequestCharacterEndpoint(ctx context.Context, dir, realm, character, endpoint string, filename string) (string, error) {
	path := fmt.Sprintf("/profile/wow/character/%s/%s",
		url.PathEscape(realm),
		url.PathEscape(character))

	if endpoint != "" {
		path += "/" + endpoint
	}

	return c.RequestProfile(ctx, dir, path, filename)
}

func (c *Client) RequestAccountEndpoint(ctx context.Context, dir, endpoint string, filename string) (string, error) {
	path := "/profile/user/wow"
	if endpoint != "" {
		path += "/" + endpoint
	}

	return c.RequestProfile(ctx, dir, path, filename)
}

// ProfileNamespace returns the regional profile-* namespace.
func (c *Client) ProfileNamespace() string {
	return fmt.Sprintf("profile-%s", c.region)
}

func (c *Client) Region() Region {
	return c.region
}

func tokenPrefix(token string) string {
	const n = 8
	if len(token) == 0 {
		return "(empty)"
	}
	if len(token) <= n {
		return "(short)"
	}
	return token[:n] + "..."
}

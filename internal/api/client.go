// Package api is a minimal HTTP client for the MeteoSwiss STAC collection
// "ch.meteoschweiz.ogd-local-forecasting" hosted on data.geo.admin.ch.
//
// It can list collection items, locate the most recent forecast-run asset for
// a given parameter, and download asset files (CSV).
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"
)

// DefaultBaseURL is the data.geo.admin.ch STAC base URL.
const DefaultBaseURL = "https://data.geo.admin.ch"

// CollectionID is the STAC collection used for local point forecasts.
const CollectionID = "ch.meteoschweiz.ogd-local-forecasting"

// Asset filenames follow vnut12.lssw.YYYYMMDDHHmm.<param>.csv.
var assetNameRE = regexp.MustCompile(`^vnut12\.lssw\.(\d{12})\.([a-z0-9]+)\.csv$`)

// Client talks to the data.geo.admin.ch STAC API.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient returns a Client with sensible timeouts.
func NewClient() *Client {
	return &Client{
		BaseURL: DefaultBaseURL,
		HTTPClient: &http.Client{
			// Per-request timeout protects against silent connection hangs.
			// Large files can take a while; this is a hard upper bound.
			Timeout: 10 * time.Minute,
		},
	}
}

// MetaParametersURL returns the canonical URL of the parameter metadata CSV.
func (c *Client) MetaParametersURL() string {
	return fmt.Sprintf("%s/%s/ogd-local-forecasting_meta_parameters.csv", c.BaseURL, CollectionID)
}

// MetaPointsURL returns the canonical URL of the point metadata CSV.
func (c *Client) MetaPointsURL() string {
	return fmt.Sprintf("%s/%s/ogd-local-forecasting_meta_point.csv", c.BaseURL, CollectionID)
}

// stacItemsResponse mirrors the small subset of the STAC API we need.
type stacItemsResponse struct {
	Features []stacFeature `json:"features"`
	Links    []struct {
		Rel  string `json:"rel"`
		Href string `json:"href"`
	} `json:"links"`
}

type stacFeature struct {
	ID         string                `json:"id"`
	Properties stacFeatureProperties `json:"properties"`
	Assets     map[string]stacAsset  `json:"assets"`
}

type stacFeatureProperties struct {
	DateTime time.Time `json:"datetime"`
	Updated  time.Time `json:"updated"`
}

type stacAsset struct {
	Type    string    `json:"type"`
	Href    string    `json:"href"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}

// ListAllItems retrieves every item in the collection, following pagination
// links. With the current dataset this is a handful of features.
func (c *Client) ListAllItems(ctx context.Context) ([]stacFeature, error) {
	first := fmt.Sprintf("%s/api/stac/v1/collections/%s/items?limit=100", c.BaseURL, CollectionID)
	var all []stacFeature
	next := first
	for next != "" {
		slog.Info("fetching stac items page", "url", next)
		resp, err := c.getJSON(ctx, next)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Features...)
		next = ""
		for _, l := range resp.Links {
			if l.Rel == "next" {
				next = l.Href
				break
			}
		}
	}
	slog.Info("stac items collected", "feature_count", len(all))
	return all, nil
}

func (c *Client) getJSON(ctx context.Context, u string) (*stacItemsResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("new request %s: %w", u, err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", u, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("get %s: status %d: %s", u, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out stacItemsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode %s: %w", u, err)
	}
	return &out, nil
}

// LatestAsset describes the most-recent forecast-run CSV for a parameter.
type LatestAsset struct {
	Parameter string
	FeatureID string
	RunTime   time.Time // UTC
	Filename  string
	Href      string
}

// LatestAssets fetches every collection item and returns the asset with the
// highest run timestamp for each requested parameter. Errors if any requested
// parameter has no matching asset.
func (c *Client) LatestAssets(ctx context.Context, params []string) (map[string]LatestAsset, error) {
	features, err := c.ListAllItems(ctx)
	if err != nil {
		return nil, err
	}
	return findLatestAssets(features, params)
}

// findLatestAssets is the pure-function core of LatestAssets, separated for
// unit testing.
func findLatestAssets(features []stacFeature, params []string) (map[string]LatestAsset, error) {
	want := make(map[string]struct{}, len(params))
	for _, p := range params {
		want[p] = struct{}{}
	}
	best := make(map[string]LatestAsset)

	for _, f := range features {
		for name, a := range f.Assets {
			m := assetNameRE.FindStringSubmatch(name)
			if m == nil {
				continue
			}
			runStr, param := m[1], m[2]
			if _, ok := want[param]; !ok {
				continue
			}
			run, err := time.ParseInLocation("200601021504", runStr, time.UTC)
			if err != nil {
				return nil, fmt.Errorf("parse run time in asset %q: %w", name, err)
			}
			cur, seen := best[param]
			if !seen || run.After(cur.RunTime) {
				best[param] = LatestAsset{
					Parameter: param,
					FeatureID: f.ID,
					RunTime:   run,
					Filename:  name,
					Href:      a.Href,
				}
			}
		}
	}

	var missing []string
	for _, p := range params {
		if _, ok := best[p]; !ok {
			missing = append(missing, p)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("no asset found for parameters: %s", strings.Join(missing, ", "))
	}
	return best, nil
}

// Download streams the asset at href into dst.
func (c *Client) Download(ctx context.Context, href string, dst io.Writer) (int64, error) {
	parsed, err := url.Parse(href)
	if err != nil {
		return 0, fmt.Errorf("parse url %q: %w", href, err)
	}
	slog.Info("downloading asset", "host", parsed.Host, "path", parsed.Path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, href, nil)
	if err != nil {
		return 0, fmt.Errorf("new request %s: %w", href, err)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("get %s: %w", href, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return 0, fmt.Errorf("get %s: status %d: %s", href, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	n, err := io.Copy(dst, resp.Body)
	if err != nil {
		return n, fmt.Errorf("download %s: %w", href, err)
	}
	slog.Info("download complete",
		"file", path.Base(parsed.Path),
		"bytes", n,
	)
	return n, nil
}

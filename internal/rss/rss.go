package rss

import (
	"bekadoux/gator/internal/common"
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
)

type Feed struct {
	Channel struct {
		Title       string     `xml:"title"`
		Link        string     `xml:"link"`
		Description string     `xml:"description"`
		Item        []FeedItem `xml:"item"`
	} `xml:"channel"`
}

type FeedItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func FetchFeed(ctx context.Context, feedURL string) (*Feed, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return &Feed{}, fmt.Errorf("could not create request: %w", err)
	}

	req.Header.Set("User-Agent", "gator")

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return &Feed{}, fmt.Errorf("request failed: %w", err)
	}
	defer common.CloseWithError(&err, res.Body, "response body")

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return &Feed{}, fmt.Errorf("could not read response body: %w", err)
	}

	feed := &Feed{}
	err = xml.Unmarshal(data, feed)
	if err != nil {
		return &Feed{}, fmt.Errorf("unmarshal failure: %w", err)
	}

	feed = unescapeFeed(feed)

	return feed, nil
}

func unescapeFeed(feed *Feed) *Feed {
	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)
	for i, item := range feed.Channel.Item {
		feed.Channel.Item[i].Title = html.UnescapeString(item.Title)
		feed.Channel.Item[i].Description = html.UnescapeString(item.Description)
	}

	return feed
}

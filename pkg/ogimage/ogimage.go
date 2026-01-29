package ogimage

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

func extractOGImage(body io.Reader) (string, error) {
	doc, err := html.Parse(body)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	var ogImageURL string
	var findOGImage func(*html.Node)
	findOGImage = func(n *html.Node) {
		if ogImageURL != "" {
			return
		}

		if n.Type == html.ElementNode && n.Data == "meta" {
			var property, content string
			for _, attr := range n.Attr {
				switch attr.Key {
				case "property", "name":
					property = attr.Val
				case "content":
					content = attr.Val
				}
			}

			switch property {
			case "og:image", "og:image:url", "og:image:secure_url",
				"twitter:image", "twitter:image:src":
				if content != "" {
					ogImageURL = content
					return
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findOGImage(c)
		}
	}

	findOGImage(doc)

	if ogImageURL == "" {
		return "", fmt.Errorf("no og:image found")
	}

	return ogImageURL, nil
}

func resolveURL(baseURL, targetURL string) (string, error) {
	if strings.HasPrefix(targetURL, "http://") || strings.HasPrefix(targetURL, "https://") {
		return targetURL, nil
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse base URL: %w", err)
	}

	target, err := url.Parse(targetURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse target URL: %w", err)
	}

	return base.ResolveReference(target).String(), nil
}

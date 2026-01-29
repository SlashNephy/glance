package ogimage

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func Handler(httpClient *http.Client, userAgent string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		targetURL := r.URL.Query().Get("url")
		if targetURL == "" {
			http.Error(w, "url parameter is required", http.StatusBadRequest)
			return
		}

		parsedURL, err := url.Parse(targetURL)
		if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
			http.Error(w, "invalid url parameter", http.StatusBadRequest)
			return
		}

		htmlRequest, err := http.NewRequestWithContext(r.Context(), http.MethodGet, parsedURL.String(), nil)
		if err != nil {
			http.Error(w, "failed to create request", http.StatusInternalServerError)
			return
		}

		htmlRequest.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		htmlRequest.Header.Set("User-Agent", userAgent)

		htmlResponse, err := httpClient.Do(htmlRequest)
		if err != nil {
			http.Error(w, "failed to fetch URL", http.StatusBadGateway)
			return
		}
		defer htmlResponse.Body.Close()

		if htmlResponse.StatusCode != http.StatusOK {
			http.Error(w, fmt.Sprintf("target URL returned status %d", htmlResponse.StatusCode), http.StatusBadGateway)
			return
		}

		ogImageURL, err := extractOGImage(htmlResponse.Body)
		if err != nil {
			http.Error(w, "no og:image found", http.StatusNotFound)
			return
		}

		resolvedImageURL, err := resolveURL(targetURL, ogImageURL)
		if err != nil {
			http.Error(w, "failed to resolve image URL", http.StatusInternalServerError)
			return
		}

		imageRequest, err := http.NewRequestWithContext(r.Context(), http.MethodGet, resolvedImageURL, nil)
		if err != nil {
			http.Error(w, "failed to create image request", http.StatusInternalServerError)
			return
		}

		imageRequest.Header.Set("User-Agent", userAgent)
		imageRequest.Header.Set("Referer", targetURL)

		imageResponse, err := httpClient.Do(imageRequest)
		if err != nil {
			http.Error(w, "failed to fetch image", http.StatusBadGateway)
			return
		}
		defer imageResponse.Body.Close()

		if imageResponse.StatusCode != http.StatusOK {
			http.Error(w, fmt.Sprintf("image URL returned status %d", imageResponse.StatusCode), http.StatusBadGateway)
			return
		}

		contentType := imageResponse.Header.Get("Content-Type")
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}

		contentLength := imageResponse.Header.Get("Content-Length")
		if contentLength != "" {
			w.Header().Set("Content-Length", contentLength)
		}

		w.Header().Set("Cache-Control", "public, max-age=86400") // 24h

		if _, err = io.Copy(w, imageResponse.Body); err != nil {
			http.Error(w, "failed to copy image response", http.StatusInternalServerError)
			return
		}
	}
}

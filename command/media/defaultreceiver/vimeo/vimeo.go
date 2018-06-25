package vimeo

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/oliverpool/go-chromecast"
	"github.com/oliverpool/go-chromecast/command/media"
	"github.com/oliverpool/go-chromecast/command/media/defaultreceiver"
)

type App struct {
	*media.App
}

func LaunchAndConnect(client chromecast.Client, statuses ...chromecast.Status) (App, error) {
	app, err := defaultreceiver.LaunchAndConnect(client, statuses...)
	return App{app}, err
}

func (a App) Load(id string, options ...media.Option) (<-chan []byte, error) {
	item := media.Item{
		ContentID:   id,
		ContentType: "video/mp4",
		StreamType:  "BUFFERED",
	}
	return a.App.Load(item, options...)
}

func URLLoader(rawurl string, options ...media.Option) (func(client chromecast.Client, statuses ...chromecast.Status) (<-chan []byte, error), error) {
	iframe, err := ExtractIframe(rawurl)
	if err != nil {
		return nil, err
	}
	mp4, err := ExtractMp4(rawurl, iframe)
	if err != nil {
		return nil, err
	}

	return func(client chromecast.Client, statuses ...chromecast.Status) (<-chan []byte, error) {
		app, err := LaunchAndConnect(client, statuses...)
		if err != nil {
			return nil, err
		}
		return app.Load(mp4, options...)
	}, nil
}

func ExtractIframe(rawurl string) (string, error) {
	{
		u, err := url.Parse(rawurl)
		if err != nil {
			return "", fmt.Errorf("could not parse url '%s': %v", rawurl, err)
		}
		if !u.IsAbs() {
			return "", fmt.Errorf("url '%s' is not absolute", rawurl)
		}
	}

	resp, err := http.Get(rawurl)
	if err != nil {
		return "", fmt.Errorf("could not fetch url '%s': %v", rawurl, err)
	}

	defer resp.Body.Close()
	id, err := extractIframeFromPage(resp.Body)
	if err != nil {
		return "", fmt.Errorf("could extract iframe-url from response '%s': %v", rawurl, err)
	}
	return id, nil
}

func ExtractMp4(rawurl, iframe string) (string, error) {
	req, err := http.NewRequest("GET", iframe, nil)
	if err != nil {
		return "", fmt.Errorf("could not prepare iframe '%s' request: %v", iframe, err)
	}
	req.Header.Set("Referer", rawurl)

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return "", fmt.Errorf("could not get iframe '%s': %v", iframe, err)
	}

	defer resp.Body.Close()
	id, err := extractMp4FromIframe(resp.Body)
	if err != nil {
		return "", fmt.Errorf("could extract iframe-url from response '%s': %v", rawurl, err)
	}
	return id, nil
}

func extractIframeFromPage(body io.Reader) (string, error) {
	scanner := bufio.NewScanner(body)
	scanner.Split(bufio.ScanWords)

	for scanner.Scan() {
		s := scanner.Text()
		if strings.HasPrefix(s, "vimeo-id") {
			reg, err := regexp.Compile("[^0-9]+")
			if err != nil {
				return "", err
			}
			return "https://player.vimeo.com/video/" + reg.ReplaceAllString(s, ""), nil
		}
	}
	return "", errors.New("no vimeo-id found")
}

func extractMp4FromIframe(body io.Reader) (string, error) {
	scanner := bufio.NewScanner(body)
	scanner.Split(splitOnString("\""))

	bestQuality := ""

	for scanner.Scan() {
		s := scanner.Text()
		if strings.HasSuffix(s, ".mp4") {
			bestQuality = s
		}
	}
	if bestQuality == "" {
		return "", errors.New("no .mp4 vimeo-src found")
	}
	return bestQuality, nil
}

func splitOnString(delimiter string) bufio.SplitFunc {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		// Return nothing if at end of file and no data passed
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		// Find the index of the delimiter
		if i := strings.Index(string(data), delimiter); i >= 0 {
			return i + 1, data[0:i], nil
		}

		// If at end of file with data return the data
		if atEOF {
			return len(data), data, nil
		}

		return
	}
}

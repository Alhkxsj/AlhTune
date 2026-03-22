package core

import (
	"net/http"
	"time"

	"github.com/guohuiyuan/music-lib/model"
)

// ValidatePlayable checks if a song is playable by testing its download URL
func ValidatePlayable(song *model.Song) bool {
	if song == nil || song.ID == "" || song.Source == "" {
		return false
	}

	if song.Source == "soda" || song.Source == "fivesing" {
		return false
	}

	fn := GetDownloadFunc(song.Source)
	if fn == nil {
		return false
	}

	urlStr, err := fn(&model.Song{ID: song.ID, Source: song.Source})
	if err != nil || urlStr == "" {
		return false
	}

	req, err := BuildSourceRequest("GET", urlStr, song.Source, "bytes=0-1")
	if err != nil {
		return false
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200 || resp.StatusCode == 206
}

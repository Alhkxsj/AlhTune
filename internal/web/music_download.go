package web

import (
	"bytes"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/Alhkxsj/AlhTune/core"
	"github.com/Alhkxsj/AlhTune/internal/errors"
	"github.com/guohuiyuan/music-lib/model"
	"github.com/guohuiyuan/music-lib/soda"
)

func handleDownload(c *gin.Context) {
	song, baseFilename, ok := parseDownloadParams(c)
	if !ok {
		return
	}

	embedMeta := c.Query("embed") == "1" && strings.TrimSpace(c.GetHeader("Range")) == ""
	if embedMeta {
		downloadWithEmbeddedMeta(c, song, baseFilename)
		return
	}
	if song.Source == "soda" {
		downloadSodaStream(c, song, baseFilename)
		return
	}
	downloadProxyStream(c, song, baseFilename)
}

func parseDownloadParams(c *gin.Context) (*model.Song, string, bool) {
	id := c.Query("id")
	source := c.Query("source")
	name := c.Query("name")
	artist := c.Query("artist")
	coverURL := strings.TrimSpace(c.Query("cover"))
	extra := parseSongExtraQuery(c.Query("extra"))

	if id == "" || source == "" {
		c.String(400, "Missing params")
		return nil, "", false
	}
	if name == "" {
		name = "Unknown"
	}
	if artist == "" {
		artist = "Unknown"
	}

	song := &model.Song{ID: id, Source: source, Name: name, Artist: artist, Cover: coverURL, Extra: extra}
	baseFilename := fmt.Sprintf("%s - %s", name, artist)
	return song, baseFilename, true
}

// downloadWithEmbeddedMeta fetches audio data, embeds metadata (lyrics/cover), and returns the result.
func downloadWithEmbeddedMeta(c *gin.Context, song *model.Song, baseFilename string) {
	audioData, err := fetchAudioData(song)
	if err != nil {
		respondDownloadError(c, err)
		return
	}

	lyric := fetchLyric(song)
	coverData, coverMime := fetchCover(song)

	ext := core.DetectAudioExt(audioData)
	
	embedConfig := MetadataEmbedConfig{
		AudioData: audioData,
		Song:      song,
		Lyric:     lyric,
		CoverData: coverData,
		CoverMime: coverMime,
		Extension: ext,
	}
	finalData := maybeEmbedMetadata(c, embedConfig)

	filename := fmt.Sprintf("%s.%s", baseFilename, ext)
	setDownloadHeader(c, filename)
	c.Data(200, core.AudioMimeByExt(ext), finalData)
}

// downloadSodaStream fetches and decrypts soda audio, then serves it with range support.
func downloadSodaStream(c *gin.Context, song *model.Song, baseFilename string) {
	audioData, err := fetchSodaAudio(song)
	if err != nil {
		respondDownloadError(c, err)
		return
	}
	ext := core.DetectAudioExt(audioData)
	filename := fmt.Sprintf("%s.%s", baseFilename, ext)
	setDownloadHeader(c, filename)
	http.ServeContent(c.Writer, c.Request, filename, time.Now(), bytes.NewReader(audioData))
}

// downloadProxyStream proxies the upstream audio response directly to the client.
func downloadProxyStream(c *gin.Context, song *model.Song, baseFilename string) {
	dlFunc := core.GetDownloadFunc(song.Source)
	if dlFunc == nil {
		c.String(400, "Unknown source")
		return
	}
	downloadURL, err := dlFunc(song)
	if err != nil {
		c.String(404, "Failed to get URL")
		return
	}

	req, reqErr := core.BuildSourceRequest("GET", downloadURL, song.Source, c.GetHeader("Range"))
	if reqErr != nil {
		c.String(502, "Upstream request error")
		return
	}
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		c.String(502, "Upstream stream error")
		return
	}
	defer resp.Body.Close()

	copyResponseHeaders(c, resp)
	ext := resolveAudioExt(resp.Header.Get("Content-Type"), downloadURL)
	filename := fmt.Sprintf("%s.%s", baseFilename, ext)
	setDownloadHeader(c, filename)
	c.Status(resp.StatusCode)
	_, _ = io.Copy(c.Writer, resp.Body)
}

// --- audio data fetchers ---

// downloadError carries an HTTP status code with an error message.
type downloadError struct {
	code    int
	message string
}

func (e *downloadError) Error() string { return e.message }

func respondDownloadError(c *gin.Context, err error) {
	if de, ok := err.(*downloadError); ok {
		c.String(de.code, de.message)
		return
	}
	c.String(500, err.Error())
}

func fetchAudioData(song *model.Song) ([]byte, error) {
	if song.Source == "soda" {
		return fetchSodaAudio(song)
	}

	dlFunc := core.GetDownloadFunc(song.Source)
	if dlFunc == nil {
		return nil, &downloadError{400, "Unknown source"}
	}
	downloadURL, err := dlFunc(song)
	if err != nil {
		return nil, &downloadError{404, "Failed to get URL"}
	}
	audioData, _, err := core.FetchBytesWithMime(downloadURL, song.Source)
	if err != nil {
		return nil, &downloadError{502, "Upstream stream error"}
	}
	return audioData, nil
}

func fetchSodaAudio(song *model.Song) ([]byte, error) {
	cookie := core.CM.Get("soda")
	sodaInst := soda.New(cookie)
	info, err := sodaInst.GetDownloadInfo(song)
	if err != nil {
		return nil, &downloadError{502, "Soda info error"}
	}
	encryptedData, _, err := core.FetchBytesWithMime(info.URL, "soda")
	if err != nil {
		return nil, &downloadError{502, "Soda stream error"}
	}
	decrypted, err := soda.DecryptAudio(encryptedData, info.PlayAuth)
	if err != nil {
		return nil, &downloadError{500, "Decrypt failed"}
	}
	return decrypted, nil
}

func fetchLyric(song *model.Song) string {
	lyricFn := core.GetLyricFuncFromSource(song.Source)
	if lyricFn == nil {
		return ""
	}
	lyric, _ := lyricFn(&model.Song{ID: song.ID, Source: song.Source})
	return lyric
}

func fetchCover(song *model.Song) ([]byte, string) {
	if song.Cover == "" {
		return nil, ""
	}
	data, mime, _ := core.FetchBytesWithMime(song.Cover, song.Source)
	return data, mime
}

// --- metadata embedding ---

type MetadataEmbedConfig struct {
	AudioData  []byte
	Song       *model.Song
	Lyric      string
	CoverData  []byte
	CoverMime  string
	Extension  string
}

var supportedEmbedFormats = map[string]bool{
	"mp3": true, "flac": true, "m4a": true, "wma": true,
}

func maybeEmbedMetadata(c *gin.Context, config MetadataEmbedConfig) []byte {
	if !supportedEmbedFormats[config.Extension] || (config.Lyric == "" && len(config.CoverData) == 0) {
		return config.AudioData
	}

	embedded, err := core.EmbedSongMetadata(config.AudioData, config.Song, config.Lyric, config.CoverData, config.CoverMime)
	if err == nil {
		return embedded
	}
	if stderrors.Is(err, errors.ErrFFmpegNotFound) {
		c.Header("X-MusicDL-Warning", "ffmpeg not found, metadata embedding skipped")
	} else {
		c.Header("X-MusicDL-Warning", "metadata embedding failed, using original audio")
	}
	return config.AudioData
}

// --- response helpers ---

var skipResponseHeaders = map[string]bool{
	"Transfer-Encoding":          true,
	"Date":                       true,
	"Access-Control-Allow-Origin": true,
}

func copyResponseHeaders(c *gin.Context, resp *http.Response) {
	for k, v := range resp.Header {
		if !skipResponseHeaders[k] {
			c.Writer.Header()[k] = v
		}
	}
}

var knownAudioExts = map[string]bool{
	"mp3": true, "flac": true, "ogg": true, "m4a": true,
}

func resolveAudioExt(contentType, downloadURL string) string {
	if ext := core.DetectAudioExtByContentType(contentType); ext != "" {
		return ext
	}
	if parsedURL, err := url.Parse(downloadURL); err == nil {
		suffix := strings.ToLower(strings.TrimPrefix(path.Ext(parsedURL.Path), "."))
		if knownAudioExts[suffix] {
			return suffix
		}
	}
	return "mp3"
}

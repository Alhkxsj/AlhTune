package core

import (
	"bytes"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dhowden/tag"
	"github.com/Alhkxsj/AlhTune/internal/errors"
	"github.com/Alhkxsj/AlhTune/internal/utils"
	"github.com/guohuiyuan/music-lib/model"
)

// IsDurationClose checks if two durations are similar
func IsDurationClose(a, b int) bool {
	return utils.IsDurationClose(a, b)
}

// IntAbs returns the absolute value of an integer
func IntAbs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// CalcSongSimilarity calculates similarity between two songs
func CalcSongSimilarity(name, artist, candName, candArtist string) float64 {
	return utils.CalcSongSimilarity(name, artist, candName, candArtist)
}

// coverMimeMap maps file extensions to MIME types
var coverMimeMap = map[string]string{
	"png":  "image/png",
	"webp": "image/webp",
	"gif":  "image/gif",
}

// normalizeCoverMime normalizes cover image MIME type
func normalizeCoverMime(coverMime string) string {
	coverMime = strings.TrimSpace(strings.ToLower(coverMime))
	if coverMime == "" {
		return "image/jpeg"
	}

	for key, mime := range coverMimeMap {
		if strings.Contains(coverMime, key) {
			return mime
		}
	}
	return "image/jpeg"
}

// FetchBytesWithMime fetches bytes from URL and returns content type
func FetchBytesWithMime(urlStr string, source string) ([]byte, string, error) {
	if urlStr == "" {
		return nil, "", fmt.Errorf("fetch failed: empty URL for source %s", source)
	}

	req, err := BuildSourceRequest("GET", urlStr, source, "")
	if err != nil {
		return nil, "", fmt.Errorf("fetch failed: build request for source %s: %w", source, err)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("fetch failed: download from source %s: %w", source, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("fetch failed: source %s returned status %d", source, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("fetch failed: read response from source %s: %w", source, err)
	}

	if len(data) == 0 {
		return nil, "", fmt.Errorf("fetch failed: source %s returned empty response", source)
	}

	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	if idx := strings.Index(contentType, ";"); idx >= 0 {
		contentType = strings.TrimSpace(contentType[:idx])
	}

	return data, contentType, nil
}

// EmbedSongMetadata embeds song metadata into audio file
func EmbedSongMetadata(
	audioData []byte,
	song *model.Song,
	lyric string,
	coverData []byte,
	coverMime string,
) ([]byte, error) {
	if len(audioData) == 0 {
		return nil, stderrors.New("empty audio data")
	}

	ext := DetectAudioExt(audioData)
	if song != nil && song.Ext != "" {
		songExt := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(song.Ext, ".")))
		switch songExt {
		case AudioExtMP3, AudioExtFLAC, AudioExtM4A, AudioExtWMA:
			ext = songExt
		}
	}

	title := ""
	artist := ""
	if song != nil {
		title = strings.TrimSpace(song.Name)
		artist = strings.TrimSpace(song.Artist)
	}
	lyric = strings.TrimSpace(lyric)

	supportedExts := map[string]bool{
		AudioExtMP3: true, AudioExtFLAC: true, AudioExtM4A: true, AudioExtWMA: true,
	}
	if !supportedExts[ext] {
		return audioData, nil
	}

	if title == "" && artist == "" && lyric == "" && len(coverData) == 0 {
		return audioData, nil
	}

	_, _ = tag.ReadFrom(bytes.NewReader(audioData))

	argsConfig := FFmpegArgsConfig{
		Extension: ext,
		Title:     title,
		Artist:    artist,
		Lyric:     lyric,
		CoverPath: "",
		HasCover:  len(coverData) > 0,
	}
	
	return embedAudioMetadataByFFmpeg(audioData, coverData, normalizeCoverMime(coverMime), argsConfig)
}

// embedAudioMetadataByFFmpeg uses FFmpeg to embed metadata
func embedAudioMetadataByFFmpeg(
	audioData []byte,
	coverData []byte,
	coverMime string,
	argsConfig FFmpegArgsConfig,
) ([]byte, error) {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, errors.ErrFFmpegNotFound
	}

	inPath, err := writeTempFile(audioData, "gomusicdl-in-*."+argsConfig.Extension)
	if err != nil {
		return nil, err
	}
	defer os.Remove(inPath)

	outPath, err := createTempFile("gomusicdl-out-*."+argsConfig.Extension)
	if err != nil {
		return nil, err
	}
	defer os.Remove(outPath)

	coverPath, hasCover, err := prepareCoverFile(coverData, coverMime)
	if err != nil {
		return nil, err
	}
	if hasCover {
		defer os.Remove(coverPath)
	}

	argsConfig.InPath = inPath
	argsConfig.OutPath = outPath
	argsConfig.CoverPath = coverPath
	argsConfig.HasCover = hasCover
	
	args := buildFFmpegArgs(argsConfig)

	cmd := exec.Command(ffmpegPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg metadata embed failed: %v, output: %s", err, strings.TrimSpace(string(out)))
	}

	return readAndValidateOutput(outPath)
}

func writeTempFile(data []byte, pattern string) (string, error) {
	file, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	path := file.Name()
	
	if _, err := file.Write(data); err != nil {
		file.Close()
		return "", err
	}
	file.Close()
	
	return path, nil
}

func createTempFile(pattern string) (string, error) {
	file, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	path := file.Name()
	file.Close()
	return path, nil
}

func prepareCoverFile(coverData []byte, coverMime string) (string, bool, error) {
	if len(coverData) == 0 {
		return "", false, nil
	}

	coverExt := ".jpg"
	if strings.Contains(coverMime, "png") {
		coverExt = ".png"
	}

	coverPath, err := writeTempFile(coverData, "gomusicdl-cover-*"+coverExt)
	if err != nil {
		return "", false, err
	}

	return coverPath, true, nil
}

func readAndValidateOutput(path string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, stderrors.New("embedded output is empty")
	}
	return data, nil
}

// FFmpegArgsConfig holds configuration for FFmpeg command arguments
type FFmpegArgsConfig struct {
	InPath    string
	OutPath   string
	Extension string
	Title     string
	Artist    string
	Lyric     string
	CoverPath string
	HasCover  bool
}

// buildFFmpegArgs builds FFmpeg command arguments
func buildFFmpegArgs(config FFmpegArgsConfig) []string {
	args := []string{"-y", "-hide_banner", "-loglevel", "error", "-i", config.InPath}

	if config.HasCover {
		args = append(args, "-i", config.CoverPath)
		args = append(args, "-map", "0:a:0", "-map", "1:v:0")
	} else {
		args = append(args, "-map", "0:a:0")
	}

	args = append(args, "-c:a", "copy")

	if config.HasCover {
		args = append(args,
			"-c:v", "copy",
			"-disposition:v:0", "attached_pic",
			"-metadata:s:v:0", "title=Album cover",
			"-metadata:s:v:0", "comment=Cover (front)",
		)
	}

	if config.Title != "" {
		args = append(args, "-metadata", "title="+config.Title)
	}
	if config.Artist != "" {
		args = append(args, "-metadata", "artist="+config.Artist)
	}
	if config.Lyric != "" {
		args = append(args, "-metadata", "lyrics="+config.Lyric)
	}

	if config.Extension == AudioExtMP3 {
		args = append(args, "-id3v2_version", "3", "-write_id3v1", "1")
	}

	args = append(args, config.OutPath)
	return args
}

package cli

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/Alhkxsj/AlhTune/core"
	"github.com/guohuiyuan/music-lib/model"
	"github.com/guohuiyuan/music-lib/soda"
	musicutils "github.com/guohuiyuan/music-lib/utils"
)

const pageSize = 15

var sourceReferers = map[string]string{
	"bilibili": "https://www.bilibili.com/",
	"migu":     "http://music.migu.cn/",
	"qq":       "http://y.qq.com",
}

var skipSwitchSources = map[string]bool{"soda": true, "fivesing": true}

var supportedEmbedFormats = map[string]bool{
	"mp3": true, "flac": true, "m4a": true, "wma": true,
}

var defaultRecommendSources = []string{"netease", "qq", "kugou", "kuwo"}

// --- HTTP helpers ---

func newSourceRequest(method, url, source string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", core.UACommon)
	if ref, ok := sourceReferers[source]; ok {
		req.Header.Set("Referer", ref)
	}
	return req, nil
}

// --- Audio fetch ---

func fetchAudioData(song *model.Song) ([]byte, error) {
	if song.Source == "soda" {
		return fetchSodaAudio(song)
	}
	return fetchSourceAudio(song)
}

func fetchSodaAudio(song *model.Song) ([]byte, error) {
	cookie := cookieManager.Get("soda")
	sodaInst := soda.New(cookie)
	info, err := sodaInst.GetDownloadInfo(song)
	if err != nil {
		return nil, err
	}
	req, _ := http.NewRequest("GET", info.URL, nil)
	req.Header.Set("User-Agent", core.UACommon)
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	encryptedData, _ := io.ReadAll(resp.Body)
	return soda.DecryptAudio(encryptedData, info.PlayAuth)
}

func fetchSourceAudio(song *model.Song) ([]byte, error) {
	dlFunc := core.GetDownloadFunc(song.Source)
	if dlFunc == nil {
		return nil, fmt.Errorf("不支持的源: %s", song.Source)
	}
	urlStr, err := dlFunc(song)
	if err != nil {
		return nil, err
	}
	if urlStr == "" {
		return nil, fmt.Errorf("下载链接为空")
	}
	req, err := newSourceRequest("GET", urlStr, song.Source)
	if err != nil {
		return nil, err
	}
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// --- Song probing ---

func probeSourceURL(song *model.Song) (*http.Response, error) {
	dlFunc := core.GetDownloadFunc(song.Source)
	if dlFunc == nil {
		return nil, fmt.Errorf("unsupported source")
	}
	urlStr, err := dlFunc(song)
	if err != nil || urlStr == "" {
		return nil, fmt.Errorf("no download URL")
	}
	req, err := newSourceRequest("GET", urlStr, song.Source)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Range", "bytes=0-1")
	return (&http.Client{Timeout: 5 * time.Second}).Do(req)
}

func probeSongDetails(song *model.Song) {
	resp, err := probeSourceURL(song)
	if err != nil {
		song.IsInvalid = true
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 206 {
		song.IsInvalid = true
		return
	}
	size := parseContentRange(resp)
	if size <= 0 {
		return
	}
	song.Size = size
	if song.Duration > 0 {
		song.Bitrate = int((size * 8) / int64(song.Duration) / 1000)
	}
}

func parseContentRange(resp *http.Response) int64 {
	cr := resp.Header.Get("Content-Range")
	if parts := strings.Split(cr, "/"); len(parts) == 2 {
		var size int64
		_, _ = fmt.Sscanf(parts[1], "%d", &size)
		return size
	}
	return resp.ContentLength
}

func probeSongsBatch(songs []model.Song) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)
	
	for i := range songs {
		if songs[i].Size == 0 {
			wg.Add(1)
			go probeSongWorker(&wg, sem, &songs[i])
		}
	}
	wg.Wait()
}

func probeSongWorker(wg *sync.WaitGroup, sem chan struct{}, song *model.Song) {
	defer wg.Done()
	sem <- struct{}{}
	defer func() { <-sem }()
	probeSongDetails(song)
}

func validatePlayable(song *model.Song) bool {
	if song == nil || song.ID == "" || song.Source == "" {
		return false
	}
	if skipSwitchSources[song.Source] {
		return false
	}
	resp, err := probeSourceURL(song)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200 || resp.StatusCode == 206
}

// --- UI utilities ---

func paginate(total, cursor int) (start, end int) {
	if total <= pageSize {
		return 0, total
	}
	start = 0
	if cursor >= pageSize {
		start = cursor - pageSize + 1
	}
	end = start + pageSize
	if end > total {
		end = total
	}
	return
}

func renderCell(text string, width int, style lipgloss.Style) string {
	return style.Width(width).MaxHeight(1).Render(text)
}

func truncate(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen-1]) + "…"
	}
	return s
}

func getSourceDisplay(s []string) string {
	if len(s) == 0 {
		return "默认源"
	}
	return strings.Join(s, ", ")
}

func formatSongFileName(song *model.Song) string {
	return fmt.Sprintf("%s - %s",
		musicutils.SanitizeFilename(song.Name),
		musicutils.SanitizeFilename(song.Artist))
}

func validIndex(cursor, length int) bool {
	return length > 0 && cursor >= 0 && cursor < length
}

func intAbs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func reloadCookies() {
	if err := cookieManager.Load(cookieFile); err != nil {
		fmt.Printf("Warning: failed to load cookies: %v\n", err)
	}
}

package cli

import (
	stderrors "errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Alhkxsj/AlhTune/core"
	"github.com/Alhkxsj/AlhTune/internal/errors"
	"github.com/Alhkxsj/AlhTune/internal/utils"
	"github.com/guohuiyuan/music-lib/model"
	musicutils "github.com/guohuiyuan/music-lib/utils"
)

// --- Search commands ---

func searchCmd(keyword, searchType string, sources []string) tea.Cmd {
	return func() tea.Msg {
		if strings.HasPrefix(keyword, "http") {
			return searchByURL(keyword)
		}
		targets := resolveSearchSources(sources, searchType)
		if searchType == "playlist" {
			return searchPlaylists(keyword, targets)
		}
		return searchSongs(keyword, targets)
	}
}

func resolveSearchSources(sources []string, searchType string) []string {
	if len(sources) > 0 {
		return sources
	}
	if searchType == "playlist" {
		return core.GetPlaylistSourceNames()
	}
	return core.GetDefaultSourceNames()
}

func searchByURL(keyword string) tea.Msg {
	src := core.DetectSource(keyword)
	if src == "" {
		return searchErrorMsg(fmt.Errorf("不支持该链接的解析，或无法识别来源"))
	}
	if parseFn := core.GetParseFunc(src); parseFn != nil {
		if song, err := parseFn(keyword); err == nil {
			probeSongDetails(song)
			return searchResultMsg([]model.Song{*song})
		}
	}
	if parsePlFn := core.GetParsePlaylistFunc(src); parsePlFn != nil {
		if _, songs, err := parsePlFn(keyword); err == nil && len(songs) > 0 {
			probeSongsBatch(songs)
			return searchResultMsg(songs)
		}
	}
	return searchErrorMsg(fmt.Errorf("解析失败: 暂不支持 %s 平台的此链接类型或解析出错", src))
}

func searchSongs(keyword string, sources []string) tea.Msg {
	var (
		allSongs []model.Song
		wg       sync.WaitGroup
		mu       sync.Mutex
	)

	for _, src := range sources {
		fn := core.GetSearchFunc(src)
		if fn == nil {
			continue
		}
		wg.Add(1)
		go func(s string, f func(string) ([]model.Song, error)) {
			defer wg.Done()
			res, err := f(keyword)
			if err != nil || len(res) == 0 {
				return
			}
			for i := range res {
				res[i].Source = s
			}
			if len(res) > 10 {
				res = res[:10]
			}
			mu.Lock()
			allSongs = append(allSongs, res...)
			mu.Unlock()
		}(src, fn)
	}
	wg.Wait()

	if len(allSongs) == 0 {
		return searchErrorMsg(fmt.Errorf("未找到结果"))
	}
	return searchResultMsg(allSongs)
}

func searchPlaylists(keyword string, sources []string) tea.Msg {
	var (
		allPlaylists []model.Playlist
		wg           sync.WaitGroup
		mu           sync.Mutex
	)
	
	searchFunc := func(src string, fn func(string) ([]model.Playlist, error)) {
		defer wg.Done()
		res, err := fn(keyword)
		if err != nil {
			return
		}
		for i := range res {
			res[i].Source = src
		}
		mu.Lock()
		allPlaylists = append(allPlaylists, res...)
		mu.Unlock()
	}
	
	for _, src := range sources {
		fn := core.GetPlaylistSearchFunc(src)
		if fn == nil {
			continue
		}
		wg.Add(1)
		go searchFunc(src, fn)
	}
	
	wg.Wait()
	if len(allPlaylists) == 0 {
		return searchErrorMsg(fmt.Errorf("未找到歌单"))
	}
	return playlistResultMsg(allPlaylists)
}

func recommendPlaylistsCmd(sources []string) tea.Cmd {
	return func() tea.Msg {
		targets := resolveRecommendSources(sources)
		allPlaylists := fetchRecommendPlaylists(targets)
		
		if len(allPlaylists) == 0 {
			return searchErrorMsg(fmt.Errorf("未找到推荐歌单"))
		}
		return playlistResultMsg(allPlaylists)
	}
}

func resolveRecommendSources(sources []string) []string {
	if len(sources) > 0 {
		return sources
	}
	return defaultRecommendSources
}

func fetchRecommendPlaylists(sources []string) []model.Playlist {
	var (
		allPlaylists []model.Playlist
		wg           sync.WaitGroup
		mu           sync.Mutex
	)
	
	fetchFunc := func(src string, fn func() ([]model.Playlist, error)) {
		defer wg.Done()
		res, err := fn()
		if err != nil || len(res) == 0 {
			return
		}
		for i := range res {
			res[i].Source = src
		}
		mu.Lock()
		allPlaylists = append(allPlaylists, res...)
		mu.Unlock()
	}
	
	for _, src := range sources {
		fn := core.GetRecommendFunc(src)
		if fn == nil {
			continue
		}
		wg.Add(1)
		go fetchFunc(src, fn)
	}
	
	wg.Wait()
	return allPlaylists
}

func fetchPlaylistSongsCmd(id, source string) tea.Cmd {
	return func() tea.Msg {
		fn := core.GetPlaylistDetailFunc(source)
		if fn == nil {
			return searchErrorMsg(fmt.Errorf("go source %s not support playlist detail", source))
		}
		songs, err := fn(id)
		if err != nil {
			return searchErrorMsg(err)
		}
		if len(songs) == 0 {
			return searchErrorMsg(fmt.Errorf("歌单为空"))
		}
		probeSongsBatch(songs)
		return searchResultMsg(songs)
	}
}

// --- Download commands ---

func downloadNextCmd(queue []model.Song, outDir string, withCover, withLyrics bool) tea.Cmd {
	return func() tea.Msg {
		if len(queue) == 0 {
			return nil
		}
		target := queue[0]
		err := downloadSongWithCookie(&target, outDir, withCover, withLyrics)
		return downloadOneFinishedMsg{err: err, song: target}
	}
}

func downloadSongWithCookie(song *model.Song, outDir string, withCover, withLyrics bool) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	audioData, err := fetchAudioData(song)
	if err != nil {
		return err
	}
	audioData = tryEmbedMetadata(audioData, song, withCover, withLyrics)
	ext := core.DetectAudioExt(audioData)
	filePath := filepath.Join(outDir, formatSongFileName(song)+"."+ext)
	return os.WriteFile(filePath, audioData, 0644)
}

func tryEmbedMetadata(audioData []byte, song *model.Song, withCover, withLyrics bool) []byte {
	ext := core.DetectAudioExt(audioData)
	if !supportedEmbedFormats[ext] {
		return audioData
	}

	lyric := fetchLyricIfEnabled(song, withLyrics)
	coverData, coverMime := fetchCoverIfEnabled(song, withCover)
	if lyric == "" && len(coverData) == 0 {
		return audioData
	}

	embedded, err := core.EmbedSongMetadata(audioData, song, lyric, coverData, coverMime)
	if err == nil {
		return embedded
	}
	if stderrors.Is(err, errors.ErrFFmpegNotFound) {
		fmt.Printf("⚠ 未检测到 ffmpeg，已跳过歌词/封面嵌入，仍会正常下载音频\n")
	} else {
		fmt.Printf("⚠ 音频元数据嵌入失败，已使用原始音频继续保存: %v\n", err)
	}
	return audioData
}

func fetchLyricIfEnabled(song *model.Song, enabled bool) string {
	if !enabled {
		return ""
	}
	if lrcFunc := core.GetLyricFuncFromSource(song.Source); lrcFunc != nil {
		if lrc, err := lrcFunc(song); err == nil {
			return lrc
		}
	}
	return ""
}

func fetchCoverIfEnabled(song *model.Song, enabled bool) ([]byte, string) {
	if !enabled || song.Cover == "" {
		return nil, ""
	}
	data, err := musicutils.Get(song.Cover)
	if err != nil || len(data) == 0 {
		return nil, ""
	}
	mime := http.DetectContentType(data)
	if idx := strings.Index(mime, ";"); idx >= 0 {
		mime = strings.TrimSpace(mime[:idx])
	}
	return data, mime
}

// --- Switch source commands ---

func switchSourceCmd(index int, song model.Song) tea.Cmd {
	return func() tea.Msg {
		newSong, err := findBestSwitchSong(song)
		return switchSourceResultMsg{index: index, song: newSong, err: err}
	}
}

type switchCandidate struct {
	song    model.Song
	score   float64
	durDiff int
}

type sourceSearchConfig struct {
	wg         *sync.WaitGroup
	mu         *sync.Mutex
	fn         func(string) ([]model.Song, error)
	keyword    string
	current    model.Song
	source     string
	candidates *[]switchCandidate
}

func findBestSwitchSong(current model.Song) (model.Song, error) {
	if current.Name == "" || current.Source == "" {
		return model.Song{}, fmt.Errorf("缺少歌名或来源")
	}
	candidates := searchSwitchCandidates(current)
	if len(candidates) == 0 {
		return model.Song{}, fmt.Errorf("未找到可换源结果")
	}
	sortSwitchCandidates(candidates)
	return selectPlayableCandidate(candidates)
}

func searchSwitchCandidates(current model.Song) []switchCandidate {
	keyword := buildSearchKeyword(current)
	
	var (
		candidates []switchCandidate
		wg         sync.WaitGroup
		mu         sync.Mutex
	)
	
	sources := getValidSwitchSources(current.Source)
	for _, src := range sources {
		fn := core.GetSearchFunc(src)
		if fn == nil {
			continue
		}
		wg.Add(1)
		config := &sourceSearchConfig{
			wg:         &wg,
			mu:         &mu,
			fn:         fn,
			keyword:    keyword,
			current:    current,
			source:     src,
			candidates: &candidates,
		}
		go searchFromSource(config)
	}
	wg.Wait()
	return candidates
}

func buildSearchKeyword(current model.Song) string {
	if current.Artist != "" {
		return current.Name + " " + current.Artist
	}
	return current.Name
}

func getValidSwitchSources(currentSource string) []string {
	var validSources []string
	for _, src := range core.GetAllSourceNames() {
		if src == "" || src == currentSource || skipSwitchSources[src] {
			continue
		}
		validSources = append(validSources, src)
	}
	return validSources
}

func searchFromSource(config *sourceSearchConfig) {
	defer config.wg.Done()
	results := scoreCandidatesFromSource(config.fn, config.keyword, config.current, config.source)
	if len(results) > 0 {
		config.mu.Lock()
		*config.candidates = append(*config.candidates, results...)
		config.mu.Unlock()
	}
}

const maxCandidatesPerSource = 8

func scoreCandidatesFromSource(fn func(string) ([]model.Song, error), keyword string, current model.Song, source string) []switchCandidate {
	res, err := fn(keyword)
	if err != nil || len(res) == 0 {
		if current.Artist != "" {
			res, _ = fn(current.Name)
		}
	}
	if len(res) == 0 {
		return nil
	}

	return processSearchResults(res, current, source)
}

func processSearchResults(res []model.Song, current model.Song, source string) []switchCandidate {
	limit := min(len(res), maxCandidatesPerSource)
	var candidates []switchCandidate

	for i := range limit {
		cand := res[i]
		cand.Source = source
		score := utils.CalcSongSimilarity(current.Name, current.Artist, cand.Name, cand.Artist)
		if score <= 0 {
			continue
		}

		durDiff := 0
		if current.Duration > 0 && cand.Duration > 0 {
			durDiff = intAbs(current.Duration - cand.Duration)
			if !utils.IsDurationClose(current.Duration, cand.Duration) {
				continue
			}
		}
		candidates = append(candidates, switchCandidate{song: cand, score: score, durDiff: durDiff})
	}
	return candidates
}

func sortSwitchCandidates(candidates []switchCandidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return candidates[i].durDiff < candidates[j].durDiff
		}
		return candidates[i].score > candidates[j].score
	})
}

func selectPlayableCandidate(candidates []switchCandidate) (model.Song, error) {
	for _, cand := range candidates {
		if validatePlayable(&cand.song) {
			return cand.song, nil
		}
	}
	return model.Song{}, fmt.Errorf("无可播放的换源结果")
}

// --- Play commands ---

func playSongCmd(song model.Song, outDir string) tea.Cmd {
	return func() tea.Msg {
		filePath, err := downloadAndSaveAudio(&song, outDir)
		if err != nil {
			return playFinishedMsg{err: err}
		}
		return startAudioPlayer(filePath)
	}
}

func downloadAndSaveAudio(song *model.Song, outDir string) (string, error) {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", err
	}
	audioData, err := fetchAudioData(song)
	if err != nil {
		return "", err
	}
	ext := core.DetectAudioExt(audioData)
	filePath := filepath.Join(outDir, formatSongFileName(song)+"."+ext)
	if err := os.WriteFile(filePath, audioData, 0644); err != nil {
		return "", err
	}
	return filePath, nil
}

func startAudioPlayer(filePath string) tea.Msg {
	cmd := exec.Command("paplay", filePath)
	if err := cmd.Start(); err != nil {
		return playFinishedMsg{filePath: filePath, err: err}
	}
	return playStartedMsg{filePath: filePath, process: cmd.Process}
}

func resumePlayCmd(filePath string) tea.Cmd {
	return func() tea.Msg {
		return startAudioPlayer(filePath)
	}
}

func playLocalSongCmd(song model.Song) tea.Cmd {
	return func() tea.Msg {
		if song.URL == "" {
			return playFinishedMsg{err: fmt.Errorf("本地文件路径为空")}
		}
		return startAudioPlayer(song.URL)
	}
}

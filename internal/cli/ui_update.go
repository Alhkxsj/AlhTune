package cli

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/guohuiyuan/music-lib/model"
)

// --- Common helpers ---

func (m modelState) goBackToInput() (tea.Model, tea.Cmd) {
	m.state = stateInput
	m.textInput.SetValue("")
	m.textInput.Focus()
	return m, textinput.Blink
}

func (m modelState) updateSpinnerAndProgress(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		spinnerModel, cmd := m.spinner.Update(msg)
		m.spinner = spinnerModel //nolint:staticcheck // Required by bubbletea pattern
		return cmd
	case progress.FrameMsg:
		_, cmd := m.progress.Update(msg)
		return cmd
	}
	return nil
}

func (m modelState) startDownload(songs []model.Song) (tea.Model, tea.Cmd) {
	m.downloadQueue = songs
	m.totalToDl = len(songs)
	m.downloaded = 0
	m.state = stateDownloading
	if m.statusMsg == "" {
		m.statusMsg = "正在准备下载..."
	}
	return m, tea.Batch(
		m.spinner.Tick,
		downloadNextCmd(m.downloadQueue, m.outDir, m.withCover, m.withLyrics),
	)
}

// --- Input state ---

func (m modelState) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		return m.handleInputKey(km)
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m modelState) handleInputKey(km tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(km.String()) {
	case "tab":
		m.toggleSearchType()
		return m, nil
	case "enter":
		return m.handleEnterKey()
	case "esc":
		return m, tea.Quit
	case "w":
		return m.startRecommend()
	case "o":
		return m.openFavorites()
	case "l":
		return m.openLocal()
	}
	return m, nil
}

func (m modelState) handleEnterKey() (tea.Model, tea.Cmd) {
	if val := m.textInput.Value(); strings.TrimSpace(val) != "" {
		return m.startSearch()
	}
	return m, nil
}

func (m *modelState) toggleSearchType() {
	if m.searchType == "song" {
		m.searchType = "playlist"
		m.textInput.Placeholder = "输入歌单关键词或链接..."
	} else {
		m.searchType = "song"
		m.textInput.Placeholder = "输入歌名、歌手或链接 (Tab 切换)..."
	}
}

func (m modelState) startSearch() (tea.Model, tea.Cmd) {
	reloadCookies()
	m.state = stateLoading
	m.songs = nil
	m.playlists = nil
	return m, tea.Batch(m.spinner.Tick, searchCmd(m.textInput.Value(), m.searchType, m.sources))
}

func (m modelState) startRecommend() (tea.Model, tea.Cmd) {
	reloadCookies()
	m.state = stateLoading
	m.searchType = "playlist"
	m.songs = nil
	m.playlists = nil
	m.statusMsg = "正在获取每日推荐歌单..."
	return m, tea.Batch(m.spinner.Tick, recommendPlaylistsCmd(m.sources))
}

func (m modelState) openFavorites() (tea.Model, tea.Cmd) {
	m.state = stateFavorites
	m.cursor = 0
	m.favorites = fm.get()
	if len(m.favorites) == 0 {
		m.statusMsg = "暂无收藏歌曲"
	} else {
		m.statusMsg = fmt.Sprintf("共 %d 首收藏歌曲", len(m.favorites))
	}
	return m, nil
}

func (m modelState) openLocal() (tea.Model, tea.Cmd) {
	m.state = stateLocal
	m.cursor = 0
	lm.scan(m.outDir)
	m.localSongs = lm.get()
	if len(m.localSongs) == 0 {
		m.statusMsg = "暂无本地音乐"
	} else {
		m.statusMsg = fmt.Sprintf("共 %d 首本地音乐", len(m.localSongs))
	}
	return m, nil
}

// --- Loading state ---

func (m modelState) updateLoading(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case searchResultMsg:
		return m.handleSearchResult(msg)
	case playlistResultMsg:
		m.playlists = msg
		m.state = statePlaylistResult
		m.cursor = 0
		m.statusMsg = fmt.Sprintf("找到 %d 个歌单。回车查看详情。", len(m.playlists))
		return m, textinput.Blink
	case searchErrorMsg:
		m.state = stateInput
		m.statusMsg = fmt.Sprintf("搜索失败: %v", msg)
		return m, textinput.Blink
	}
	return m, nil
}

func (m modelState) handleSearchResult(songs searchResultMsg) (tea.Model, tea.Cmd) {
	m.songs = songs
	m.state = stateList
	m.cursor = 0
	m.selected = make(map[int]struct{})

	if len(m.songs) == 1 && strings.HasPrefix(m.textInput.Value(), "http") {
		m.selected[0] = struct{}{}
		m.statusMsg = fmt.Sprintf("解析成功: %s。按回车下载。", m.songs[0].Name)
		return m, nil
	}
	if m.searchType == "playlist" {
		m.statusMsg = fmt.Sprintf("歌单解析完成，包含 %d 首歌曲。空格选择，回车下载。", len(m.songs))
	} else {
		m.statusMsg = fmt.Sprintf("找到 %d 首歌曲。空格选择，回车下载。", len(m.songs))
	}
	return m, nil
}

func handleNavigationKeys(m modelState, maxIndex int) (modelState, tea.Cmd) {
	km, ok := m.lastKeyMsg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	
	switch km.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < maxIndex-1 {
			m.cursor++
		}
	}
	return m, nil
}

// --- Playlist result state ---

func (m modelState) updatePlaylistResult(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	m.lastKeyMsg = km
	
	switch km.String() {
	case "up", "k", "down", "j":
		return handleNavigationKeys(m, len(m.playlists))
	case "q":
		return m, tea.Quit
	case "esc", "b":
		return m.goBackToInput()
	case "enter":
		return m.handlePlaylistEnter()
	}
	return m, nil
}

func (m modelState) handlePlaylistEnter() (tea.Model, tea.Cmd) {
	if len(m.playlists) == 0 {
		return m, nil
	}
	
	target := m.playlists[m.cursor]
	m.state = stateLoading
	m.statusMsg = fmt.Sprintf("正在获取歌单 [%s] 详情...", target.Name)
	return m, tea.Batch(m.spinner.Tick, fetchPlaylistSongsCmd(target.ID, target.Source))
}

// --- List state ---

func (m modelState) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	m.lastKeyMsg = km

	return m.handleListKey(km.String())
}

func (m modelState) handleListKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k", "down", "j":
		return handleNavigationKeys(m, len(m.songs))
	case " ":
		return m.toggleSelection(), nil
	case "a":
		m.toggleAllSelection()
		return m, nil
	case "q":
		return m, tea.Quit
	case "esc", "b":
		return m.goBackToInput()
	case "enter":
		return m.handleListDownload()
	case "r":
		return m.handleListSwitch()
	case "p":
		return m.handleListPlay()
	case "o", "O":
		return m.handleListFavoriteAndDownload()
	default:
		return m, nil
	}
}

func (m modelState) toggleSelection() modelState {
	if _, ok := m.selected[m.cursor]; ok {
		delete(m.selected, m.cursor)
	} else {
		m.selected[m.cursor] = struct{}{}
	}
	return m
}

func (m *modelState) toggleAllSelection() {
	if len(m.selected) == len(m.songs) && len(m.songs) > 0 {
		m.selected = make(map[int]struct{})
		m.statusMsg = "已取消全部选择"
	} else {
		for i := range m.songs {
			m.selected[i] = struct{}{}
		}
		m.statusMsg = fmt.Sprintf("已选中全部 %d 首歌曲", len(m.songs))
	}
}

func (m modelState) handleListDownload() (tea.Model, tea.Cmd) {
	if len(m.selected) == 0 {
		m.selected[m.cursor] = struct{}{}
	}
	var queue []model.Song
	for idx := range m.selected {
		if validIndex(idx, len(m.songs)) {
			queue = append(queue, m.songs[idx])
		}
	}
	m.statusMsg = ""
	return m.startDownload(queue)
}

func (m modelState) handleListSwitch() (tea.Model, tea.Cmd) {
	if !validIndex(m.cursor, len(m.songs)) {
		return m, nil
	}
	if len(m.selected) == 0 {
		m.selected[m.cursor] = struct{}{}
	}
	m.switchQueue = m.switchQueue[:0]
	for idx := range m.selected {
		if validIndex(idx, len(m.songs)) {
			m.switchQueue = append(m.switchQueue, idx)
		}
	}
	if len(m.switchQueue) == 0 {
		return m, nil
	}
	m.switchTotal = len(m.switchQueue)
	m.switched = 0
	m.state = stateSwitching
	m.statusMsg = fmt.Sprintf("正在换源... 0/%d", m.switchTotal)
	firstIdx := m.switchQueue[0]
	return m, tea.Batch(
		m.spinner.Tick,
		m.progress.SetPercent(0),
		switchSourceCmd(firstIdx, m.songs[firstIdx]),
	)
}

func (m modelState) handleListPlay() (tea.Model, tea.Cmd) {
	if !validIndex(m.cursor, len(m.songs)) {
		return m, nil
	}
	song := m.songs[m.cursor]
	m.state = statePlaying
	m.playingSong = &song
	m.playlistCursor = m.cursor
	m.statusMsg = fmt.Sprintf("正在准备播放: %s - %s", song.Name, song.Artist)
	return m, tea.Batch(m.spinner.Tick, playSongCmd(song, m.outDir))
}

func (m modelState) handleListFavoriteAndDownload() (tea.Model, tea.Cmd) {
	if !validIndex(m.cursor, len(m.songs)) {
		return m, nil
	}
	song := m.songs[m.cursor]
	fm.add(song)
	fm.save()
	m.statusMsg = fmt.Sprintf("已收藏: %s - %s", song.Name, song.Artist)
	return m.startDownload([]model.Song{song})
}

// --- Downloading state ---

func (m modelState) updateDownloading(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd := m.updateSpinnerAndProgress(msg)
	if cmd != nil {
		return m, cmd
	}

	if result, ok := msg.(downloadOneFinishedMsg); ok {
		return m.handleDownloadResult(result)
	}
	return m, nil
}

func (m modelState) handleDownloadResult(msg downloadOneFinishedMsg) (tea.Model, tea.Cmd) {
	m.downloaded++
	if msg.err != nil {
		m.statusMsg = fmt.Sprintf("❌ 失败: %s - %s (%v)", msg.song.Name, msg.song.Artist, msg.err)
	} else {
		m.statusMsg = fmt.Sprintf("已完成: %s - %s", msg.song.Name, msg.song.Artist)
	}

	pct := float64(m.downloaded) / float64(m.totalToDl)
	if len(m.downloadQueue) > 0 {
		m.downloadQueue = m.downloadQueue[1:]
	}

	if m.downloaded >= m.totalToDl {
		m.state = stateList
		m.selected = make(map[int]struct{})
		m.statusMsg = fmt.Sprintf("✅ 任务结束，共下载 %d 首歌曲", m.downloaded)
		return m, nil
	}
	return m, tea.Batch(
		m.progress.SetPercent(pct),
		downloadNextCmd(m.downloadQueue, m.outDir, m.withCover, m.withLyrics),
	)
}

// --- Switching state ---

func (m modelState) updateSwitching(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd := m.updateSpinnerAndProgress(msg)
	if cmd != nil {
		return m, cmd
	}

	if result, ok := msg.(switchSourceResultMsg); ok {
		return m.handleSwitchResult(result)
	}
	return m, nil
}

func (m modelState) handleSwitchResult(msg switchSourceResultMsg) (tea.Model, tea.Cmd) {
	m.switched++
	if msg.err == nil && validIndex(msg.index, len(m.songs)) {
		m.songs[msg.index] = msg.song
	}

	if m.switched >= m.switchTotal {
		return m.finishSwitching()
	}
	m.statusMsg = fmt.Sprintf("正在换源... %d/%d", m.switched, m.switchTotal)
	if len(m.switchQueue) > 0 {
		m.switchQueue = m.switchQueue[1:]
	}
	if len(m.switchQueue) == 0 {
		return m.finishSwitching()
	}

	pct := float64(m.switched) / float64(m.switchTotal)
	nextIdx := m.switchQueue[0]
	return m, tea.Batch(
		m.progress.SetPercent(pct),
		switchSourceCmd(nextIdx, m.songs[nextIdx]),
	)
}

func (m modelState) finishSwitching() (tea.Model, tea.Cmd) {
	m.state = stateList
	m.statusMsg = fmt.Sprintf("换源完成: %d/%d", m.switched, m.switchTotal)
	m.selected = make(map[int]struct{})
	m.switchQueue = nil
	return m, m.progress.SetPercent(1)
}

// --- Favorites state ---

func (m modelState) updateFavorites(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch strings.ToLower(km.String()) {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.favorites)-1 {
			m.cursor++
		}
	case "enter":
		return m.handleFavoriteDownload()
	case "d":
		return m.handleFavoriteDelete()
	case "b":
		return m.goBackToInput()
	}
	return m, nil
}

func (m modelState) handleFavoriteDownload() (tea.Model, tea.Cmd) {
	if !validIndex(m.cursor, len(m.favorites)) {
		return m, nil
	}
	m.statusMsg = ""
	return m.startDownload([]model.Song{m.favorites[m.cursor]})
}

func (m modelState) handleFavoriteDelete() (tea.Model, tea.Cmd) {
	if !validIndex(m.cursor, len(m.favorites)) {
		return m, nil
	}
	song := m.favorites[m.cursor]
	fm.remove(m.cursor)
	fm.save()
	m.favorites = fm.get()
	if m.cursor >= len(m.favorites) && len(m.favorites) > 0 {
		m.cursor = len(m.favorites) - 1
	}
	m.statusMsg = fmt.Sprintf("已删除: %s - %s", song.Name, song.Artist)
	return m, nil
}

// --- Local state ---

func (m modelState) updateLocal(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch strings.ToLower(km.String()) {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.localSongs)-1 {
			m.cursor++
		}
	case "b":
		return m.goBackToInput()
	case "p":
		if !validIndex(m.cursor, len(m.localSongs)) {
			return m, nil
		}
		song := m.localSongs[m.cursor]
		m.state = statePlaying
		m.playingSong = &song
		m.statusMsg = fmt.Sprintf("正在播放: %s - %s", song.Name, song.Artist)
		return m, tea.Batch(m.spinner.Tick, playLocalSongCmd(song))
	}
	return m, nil
}

// --- Playing state ---

func (m modelState) updatePlaying(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "p":
			return m.handlePlayPause()
		case "m":
			return m.handlePlayModeToggle()
		case "b":
			return m.handlePlayStop()
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case playFinishedMsg:
		return m.handlePlayFinished(msg)
	case playStartedMsg:
		m.playingProcess = msg.process
		m.playingFilePath = msg.filePath
		m.statusMsg = fmt.Sprintf("正在播放: %s", m.playingSong.Name)
		return m, nil
	}
	return m, nil
}

func (m modelState) handlePlayPause() (tea.Model, tea.Cmd) {
	if m.playingProcess != nil {
		_ = m.playingProcess.Kill()
		m.playingProcess = nil
		m.isPaused = true
		m.statusMsg = "已暂停"
		return m, nil
	}
	if m.playingFilePath != "" {
		return m, resumePlayCmd(m.playingFilePath)
	}
	return m, nil
}

func (m modelState) handlePlayModeToggle() (tea.Model, tea.Cmd) {
	m.playMode = m.playMode.next()
	m.statusMsg = fmt.Sprintf("播放模式已切换: %s", m.playMode.displayName())
	return m, nil
}

func (m modelState) handlePlayStop() (tea.Model, tea.Cmd) {
	return m.stopPlaying("已返回"), nil
}

func (m modelState) stopPlaying(statusMsg string) modelState {
	if m.playingProcess != nil {
		_ = m.playingProcess.Kill()
		m.playingProcess = nil
	}
	m.state = stateList
	m.statusMsg = statusMsg
	m.playingSong = nil
	m.playingFilePath = ""
	m.isPaused = false
	return m
}

func (m modelState) handlePlayFinished(msg playFinishedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		return m.stopPlaying(fmt.Sprintf("播放失败: %v", msg.err)), nil
	}

	switch m.playMode {
	case playModeLoop:
		if m.playingSong != nil {
			m.statusMsg = fmt.Sprintf("单曲循环: %s - %s", m.playingSong.Name, m.playingSong.Artist)
			return m, playSongCmd(*m.playingSong, m.outDir)
		}
	case playModeSequential:
		return m.playNextSequential()
	case playModeShuffle:
		return m.playNextShuffle()
	}
	return m.stopPlaying(fmt.Sprintf("播放完成: %s", msg.filePath)), nil
}

func (m modelState) resetPlaying(statusMsg string) modelState {
	return m.stopPlaying(statusMsg)
}

func (m modelState) playNextSequential() (tea.Model, tea.Cmd) {
	if len(m.songs) == 0 {
		return m.resetPlaying("播放列表为空"), nil
	}
	m.playlistCursor = (m.playlistCursor + 1) % len(m.songs)
	next := m.songs[m.playlistCursor]
	m.playingSong = &next
	m.statusMsg = fmt.Sprintf("顺序播放下一首: %s - %s", next.Name, next.Artist)
	return m, playSongCmd(next, m.outDir)
}

func (m modelState) playNextShuffle() (tea.Model, tea.Cmd) {
	if len(m.songs) == 0 {
		return m.resetPlaying("播放列表为空"), nil
	}
	newCursor := m.playlistCursor
	for newCursor == m.playlistCursor && len(m.songs) > 1 {
		newCursor = rand.Intn(len(m.songs))
	}
	m.playlistCursor = newCursor
	next := m.songs[m.playlistCursor]
	m.playingSong = &next
	m.statusMsg = fmt.Sprintf("随机播放: %s - %s", next.Name, next.Artist)
	return m, playSongCmd(next, m.outDir)
}

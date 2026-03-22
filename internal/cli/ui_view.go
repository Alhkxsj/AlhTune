package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m modelState) View() string {
	var s strings.Builder

	s.WriteString(m.renderHeader())
	s.WriteString("\n\n")
	s.WriteString(m.renderCurrentView())
	s.WriteString(m.renderStatusBar())

	return s.String()
}

func (m modelState) renderHeader() string {
	header := lipgloss.NewStyle().
		Foreground(textColor).
		Background(secondaryColor).
		Bold(true).
		Padding(0, 2).
		Render(" 🎵 Go Music DL ")
	return header
}

func (m modelState) renderCurrentView() string {
	renderFuncs := map[sessionState]func() string{
		stateInput:          m.viewInput,
		stateLoading:        m.viewLoading,
		stateList:           m.viewList,
		statePlaylistResult: m.viewPlaylistResult,
		stateDownloading:    m.viewDownloading,
		stateSwitching:      m.viewSwitching,
		statePlaying:        m.viewPlaying,
		stateFavorites:      m.viewFavorites,
		stateLocal:          m.viewLocal,
	}
	
	if fn, ok := renderFuncs[m.state]; ok {
		return fn()
	}
	return ""
}

func (m modelState) renderStatusBar() string {
	return "\n\n" + statusBarStyle.Render(fmt.Sprintf(" Mode: %s | Sources: %d ", m.searchType, len(m.sources)))
}

func (m modelState) viewInput() string {
	var s strings.Builder
	s.WriteString(lipgloss.NewStyle().Bold(true).Render("请输入搜索关键字:") + "\n")
	s.WriteString(m.textInput.View())

	modeLabel := "单曲"
	if m.searchType == "playlist" {
		modeLabel = "歌单"
	}

	infoLines := []string{
		fmt.Sprintf("(当前源：%v)", getSourceDisplay(m.sources)),
		fmt.Sprintf("(当前模式：%s搜索)", modeLabel),
		"(按 Enter 搜索，Tab 切换，w 每日推荐，o 收藏，l 本地，Ctrl+C 退出)",
	}

	for _, line := range infoLines {
		s.WriteString(lipgloss.NewStyle().Foreground(subtleColor).Render("\n"+line))
	}

	if cookieManager.Count() > 0 {
		loadedSources := cookieManager.GetSources()
		hint := fmt.Sprintf("\n(已加载 Cookie: %s)", strings.Join(loadedSources, ", "))
		s.WriteString(successStyle.Render(hint))
	}

	if m.err != nil {
		s.WriteString(errorStyle.Render(fmt.Sprintf("\n\n❌ %v", m.err)))
	}

	return s.String()
}

func (m modelState) viewLoading() string {
	return fmt.Sprintf("\n %s %s\n", m.spinner.View(), lipgloss.NewStyle().Bold(true).Render("正在处理 '"+m.textInput.Value()+"' ..."))
}

func (m modelState) viewList() string {
	var s strings.Builder
	s.WriteString(m.renderSongTable())
	s.WriteString("\n")
	s.WriteString(subtleStyle.Render(m.statusMsg))
	s.WriteString("\n\n")
	s.WriteString(subtleStyle.Render("↑/↓: 移动 • 空格：选择 • a: 全选 • r: 换源 • p: 播放 • Enter: 下载 • b: 返回 • q: 退出"))
	return s.String()
}

func (m modelState) viewPlaylistResult() string {
	var s strings.Builder
	s.WriteString(m.renderPlaylistTable())
	s.WriteString("\n")
	s.WriteString(subtleStyle.Render(m.statusMsg))
	s.WriteString("\n\n")
	s.WriteString(subtleStyle.Render("↑/↓: 移动 • Enter: 查看详情 • b: 返回 • q: 退出"))
	return s.String()
}

func (m modelState) viewDownloading() string {
	var s strings.Builder
	s.WriteString("\n")
	s.WriteString(m.progress.View() + "\n\n")
	s.WriteString(fmt.Sprintf("%s %s\n", m.spinner.View(), lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("正在处理：%d/%d", m.downloaded, m.totalToDl))))

	if len(m.downloadQueue) > 0 {
		current := m.downloadQueue[0]
		s.WriteString(warningStyle.Render(fmt.Sprintf("→ %s - %s", current.Name, current.Artist)))
	}

	s.WriteString("\n\n" + subtleStyle.Render(m.statusMsg))
	return s.String()
}

func (m modelState) viewSwitching() string {
	var s strings.Builder
	s.WriteString("\n")
	s.WriteString(m.progress.View() + "\n\n")
	s.WriteString(fmt.Sprintf("%s %s\n", m.spinner.View(), lipgloss.NewStyle().Bold(true).Render(m.statusMsg)))
	return s.String()
}

func (m modelState) viewPlaying() string {
	var s strings.Builder

	if m.isPaused {
		s.WriteString(warningStyle.Bold(true).Render("\n⏸ 已暂停\n"))
	} else {
		s.WriteString(fmt.Sprintf("\n%s %s\n", m.spinner.View(), lipgloss.NewStyle().Bold(true).Render(m.statusMsg)))
	}

	if m.playingSong != nil {
		s.WriteString("\n")
		s.WriteString(successStyle.Bold(true).Render("🎵 正在播放:") + "\n")
		s.WriteString(fmt.Sprintf("  %s %s\n", lipgloss.NewStyle().Foreground(primaryColor).Render("歌曲:"), m.playingSong.Name))
		s.WriteString(fmt.Sprintf("  %s %s\n", lipgloss.NewStyle().Foreground(primaryColor).Render("歌手:"), m.playingSong.Artist))

		if m.playingSong.Album != "" {
			s.WriteString(fmt.Sprintf("  %s %s\n", lipgloss.NewStyle().Foreground(primaryColor).Render("专辑:"), m.playingSong.Album))
		}
		if m.playingSong.Duration > 0 {
			s.WriteString(fmt.Sprintf("  %s %s\n", lipgloss.NewStyle().Foreground(primaryColor).Render("时长:"), m.playingSong.FormatDuration()))
		}
	}

	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().Foreground(m.playMode.color()).Bold(true).Render(fmt.Sprintf("🔁 %s", m.playMode.displayName())))
	s.WriteString("\n")
	s.WriteString(subtleStyle.Render("p: 暂停/继续 • m: 切换模式 • b: 返回"))
	return s.String()
}

func (m modelState) viewFavorites() string {
	var s strings.Builder
	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render("📌 我的收藏"))
	s.WriteString("\n\n")

	if len(m.favorites) == 0 {
		s.WriteString(subtleStyle.Render("暂无收藏歌曲"))
	} else {
		s.WriteString(m.renderFavoritesTable())
	}

	s.WriteString("\n\n")
	s.WriteString(subtleStyle.Render(m.statusMsg))
	s.WriteString("\n\n")
	s.WriteString(subtleStyle.Render("↑/↓: 移动 • Enter: 下载 • d: 删除 • b: 返回 • q: 退出"))
	return s.String()
}

func (m modelState) viewLocal() string {
	var s strings.Builder
	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render("📁 本地音乐"))
	s.WriteString("\n\n")

	if len(m.localSongs) == 0 {
		s.WriteString(subtleStyle.Render("暂无本地音乐"))
	} else {
		s.WriteString(m.renderLocalTable())
	}

	s.WriteString("\n\n")
	s.WriteString(subtleStyle.Render(m.statusMsg))
	s.WriteString("\n\n")
	s.WriteString(subtleStyle.Render("↑/↓: 移动 • p: 播放 • b: 返回 • q: 退出"))
	return s.String()
}

// --- Table renderers ---

const (
	colCheck  = 6
	colIdx    = 4
	colTitle  = 25
	colArtist = 15
	colAlbum  = 15
	colDur    = 8
	colSize   = 10
	colBit    = 11
	colSrc    = 10
	colCount  = 10
	colPLName = 40
	colPLAuth = 20
)

func (m modelState) renderSongTable() string {
	var b strings.Builder
	header := lipgloss.JoinHorizontal(lipgloss.Left,
		headerStyle.Width(colCheck).Render("[选]"),
		headerStyle.Width(colIdx).Render("ID"),
		headerStyle.Width(colTitle).Render("歌名"),
		headerStyle.Width(colArtist).Render("歌手"),
		headerStyle.Width(colAlbum).Render("专辑"),
		headerStyle.Width(colDur).Render("时长"),
		headerStyle.Width(colSize).Render("大小"),
		headerStyle.Width(colBit).Render("码率"),
		headerStyle.Width(colSrc).Render("来源"),
	)
	b.WriteString(header + "\n")

	start, end := paginate(len(m.songs), m.cursor)
	for i := start; i < end; i++ {
		b.WriteString(m.renderSongRow(i) + "\n")
	}
	return b.String()
}

func (m modelState) renderSongRow(i int) string {
	song := m.songs[i]
	_, isSelected := m.selected[i]
	checkStr := "[ ]"
	if isSelected {
		checkStr = checkedStyle.Render("[✓]")
	}

	var sizeStr string
	if song.IsInvalid {
		sizeStr = errorStyle.Render("!无效")
	} else {
		sizeStr = song.FormatSize()
	}

	bitrate := "-"
	if song.Bitrate > 0 {
		bitrate = fmt.Sprintf("%dkbps", song.Bitrate)
	}

	style := rowStyle
	if m.cursor == i {
		style = selectedRowStyle
	}

	return lipgloss.JoinHorizontal(lipgloss.Left,
		renderCell(checkStr, colCheck, style),
		renderCell(fmt.Sprintf("%d", i+1), colIdx, style),
		renderCell(truncate(song.Name, colTitle-4), colTitle, style),
		renderCell(truncate(song.Artist, colArtist-2), colArtist, style),
		renderCell(truncate(song.Album, colAlbum-2), colAlbum, style),
		renderCell(song.FormatDuration(), colDur, style),
		renderCell(sizeStr, colSize, style),
		renderCell(bitrate, colBit, style),
		renderCell(song.Source, colSrc, style),
	)
}

func (m modelState) renderPlaylistTable() string {
	var b strings.Builder
	header := lipgloss.JoinHorizontal(lipgloss.Left,
		headerStyle.Width(colIdx).Render("ID"),
		headerStyle.Width(colPLName).Render("歌单名称"),
		headerStyle.Width(colCount).Render("歌曲数"),
		headerStyle.Width(colPLAuth).Render("创建者"),
		headerStyle.Width(colSrc).Render("来源"),
	)
	b.WriteString(header + "\n")

	start, end := paginate(len(m.playlists), m.cursor)
	for i := start; i < end; i++ {
		pl := m.playlists[i]
		style := rowStyle
		if m.cursor == i {
			style = selectedRowStyle
		}
		row := lipgloss.JoinHorizontal(lipgloss.Left,
			renderCell(fmt.Sprintf("%d", i+1), colIdx, style),
			renderCell(truncate(pl.Name, colPLName-2), colPLName, style),
			renderCell(fmt.Sprintf("%d首", pl.TrackCount), colCount, style),
			renderCell(truncate(pl.Creator, colPLAuth-2), colPLAuth, style),
			renderCell(pl.Source, colSrc, style),
		)
		b.WriteString(row + "\n")
	}
	return b.String()
}

func (m modelState) renderFavoritesTable() string {
	var b strings.Builder
	header := lipgloss.JoinHorizontal(lipgloss.Left,
		headerStyle.Width(colIdx).Render("ID"),
		headerStyle.Width(colTitle).Render("歌名"),
		headerStyle.Width(colArtist).Render("歌手"),
		headerStyle.Width(colAlbum).Render("专辑"),
		headerStyle.Width(colDur).Render("时长"),
		headerStyle.Width(colSrc).Render("来源"),
	)
	b.WriteString(header + "\n")

	start, end := paginate(len(m.favorites), m.cursor)
	for i := start; i < end; i++ {
		song := m.favorites[i]
		style := rowStyle
		if m.cursor == i {
			style = selectedRowStyle
		}
		row := lipgloss.JoinHorizontal(lipgloss.Left,
			renderCell(fmt.Sprintf("%d", i+1), colIdx, style),
			renderCell(truncate(song.Name, colTitle-2), colTitle, style),
			renderCell(truncate(song.Artist, colArtist-2), colArtist, style),
			renderCell(truncate(song.Album, colAlbum-2), colAlbum, style),
			renderCell(song.FormatDuration(), colDur, style),
			renderCell(song.Source, colSrc, style),
		)
		b.WriteString(row + "\n")
	}
	return b.String()
}

func (m modelState) renderLocalTable() string {
	var b strings.Builder
	header := lipgloss.JoinHorizontal(lipgloss.Left,
		headerStyle.Width(colIdx).Render("ID"),
		headerStyle.Width(colTitle).Render("歌名"),
		headerStyle.Width(colArtist).Render("歌手"),
		headerStyle.Width(colSize).Render("大小"),
	)
	b.WriteString(header + "\n")

	start, end := paginate(len(m.localSongs), m.cursor)
	for i := start; i < end; i++ {
		song := m.localSongs[i]
		style := rowStyle
		if m.cursor == i {
			style = selectedRowStyle
		}
		row := lipgloss.JoinHorizontal(lipgloss.Left,
			renderCell(fmt.Sprintf("%d", i+1), colIdx, style),
			renderCell(truncate(song.Name, colTitle-2), colTitle, style),
			renderCell(truncate(song.Artist, colArtist-2), colArtist, style),
			renderCell(song.FormatSize(), colSize, style),
		)
		b.WriteString(row + "\n")
	}
	return b.String()
}

// Helper styles
var subtleStyle = lipgloss.NewStyle().Foreground(subtleColor)

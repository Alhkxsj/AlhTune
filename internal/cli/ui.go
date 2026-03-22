package cli

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Alhkxsj/AlhTune/internal/utils"
	"github.com/guohuiyuan/music-lib/model"
)

const (
	cookieFile   = "data/cookies.json"
	favoriteFile = "data/favorites.json"
)

var localMusicFormats = []string{".mp3", ".flac", ".m4a", ".ogg", ".wav", ".wma"}

var (
	// Unified theme colors (matching Web UI)
	primaryColor   = lipgloss.Color("#6366F1") // Indigo - main accent
	secondaryColor = lipgloss.Color("#A855F7") // Purple - secondary accent
	bgColor        = lipgloss.Color("#0F172A") // Slate 900 - background
	textColor      = lipgloss.Color("#F1F5F9") // Slate 100 - primary text
	subtleColor    = lipgloss.Color("#94A3B8") // Slate 400 - secondary text
	errorColor     = lipgloss.Color("#EF4444") // Red 500 - errors
	successColor   = lipgloss.Color("#10B981") // Emerald 500 - success
	warningColor   = lipgloss.Color("#F59E0B") // Amber 500 - warnings

	headerStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Background(secondaryColor).
			Bold(true).
			Padding(0, 2).
			MarginBottom(1).
			Border(lipgloss.RoundedBorder(), false, false, true, false).
			BorderForeground(primaryColor)

	rowStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Border(lipgloss.HiddenBorder(), false, false, false, true)

	selectedRowStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true).
				Padding(0, 1).
				Border(lipgloss.RoundedBorder(), false, false, false, true).
				BorderForeground(primaryColor).
				Background(lipgloss.Color("#1E293B"))

	checkedStyle = lipgloss.NewStyle().Foreground(successColor).Bold(true)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(subtleColor).
			Background(bgColor).
			Padding(0, 2).
			Border(lipgloss.RoundedBorder(), true, false, false, false).
			BorderForeground(primaryColor)

	errorStyle = lipgloss.NewStyle().Foreground(errorColor)
	successStyle = lipgloss.NewStyle().Foreground(successColor)
	warningStyle = lipgloss.NewStyle().Foreground(warningColor)
)

var (
	cookieManager = utils.NewCookieManager()
	fm            = &FavoriteManager{songs: make([]model.Song, 0)}
	lm            = &LocalMusicManager{songs: make([]model.Song, 0)}
)

// --- State types ---

const (
	stateInput sessionState = iota
	stateLoading
	stateList
	statePlaylistResult
	stateDownloading
	stateSwitching
statePlaying
	stateFavorites
	stateLocal
)

type sessionState int

type playMode string

const (
	playModeSequential playMode = "sequential"
	playModeLoop       playMode = "loop"
	playModeShuffle    playMode = "shuffle"
)

var playModeOrder = []playMode{playModeSequential, playModeLoop, playModeShuffle}

var playModeDisplay = map[playMode]struct {
	name  string
	color lipgloss.Color
}{
	playModeSequential: {"顺序播放", primaryColor},
	playModeLoop:       {"单曲循环", warningColor},
	playModeShuffle:    {"随机播放", successColor},
}

func (pm playMode) next() playMode {
	for i, m := range playModeOrder {
		if m == pm {
			return playModeOrder[(i+1)%len(playModeOrder)]
		}
	}
	return playModeSequential
}

func (pm playMode) displayName() string {
	if d, ok := playModeDisplay[pm]; ok {
		return d.name
	}
	return "顺序播放"
}

func (pm playMode) color() lipgloss.Color {
	if d, ok := playModeDisplay[pm]; ok {
		return d.color
	}
	return primaryColor
}

// --- Message types ---

type searchResultMsg []model.Song
type playlistResultMsg []model.Playlist
type searchErrorMsg error

type downloadOneFinishedMsg struct {
	err  error
	song model.Song
}

type switchSourceResultMsg struct {
	err   error
	song  model.Song
	index int
}

type playFinishedMsg struct {
	err      error
	filePath string
}

type playStartedMsg struct {
	process  *os.Process
	filePath string
}

// --- Model ---

type modelState struct {
	err             error
	playingProcess  *os.Process
	playingSong     *model.Song
	selected        map[int]struct{}
	outDir          string
	statusMsg       string
	searchType      string
	playingFilePath string
	sources         []string
	songs           []model.Song
	localSongs      []model.Song
	favorites       []model.Song
	downloadQueue   []model.Song
	switchQueue     []int
	playlists       []model.Playlist
	textInput       textinput.Model
	spinner         spinner.Model
	progress        progress.Model
	switchTotal     int
	totalToDl       int
	state           sessionState
	cursor          int
	playlistCursor  int
	switched        int
	downloaded      int
	windowWidth     int
	isPaused        bool
	withLyrics      bool
	withCover       bool
	playMode        playMode
	lastKeyMsg      tea.Msg
}

func StartUI(initialKeyword string, sources []string, outDir string, withCover bool, withLyrics bool) {
	if err := initializeDependencies(outDir); err != nil {
		fmt.Printf("Warning: failed to initialize dependencies: %v\n", err)
	}
	
	ti := textinput.New()
	ti.Placeholder = "输入歌名、歌手或链接 (Tab 切换)..."
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(primaryColor)

	initialState := stateInput
	if initialKeyword != "" {
		ti.SetValue(initialKeyword)
		initialState = stateLoading
	}

	m := modelState{
		state:      initialState,
		searchType: "song",
		textInput:  ti,
		spinner:    sp,
		progress:   progress.New(progress.WithDefaultGradient()),
		selected:   make(map[int]struct{}),
		sources:    sources,
		outDir:     outDir,
		withCover:  withCover,
		withLyrics: withLyrics,
		favorites:  fm.get(),
		localSongs: lm.get(),
		playMode:   playModeSequential,
	}

	if err := runProgram(m); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Error running program:", err)
	}
}

func initializeDependencies(outDir string) error {
	if err := cookieManager.Load(cookieFile); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Warning: failed to load cookies: %v\n", err)
	}

	fm.load()
	lm.scan(outDir)

	return nil
}

func runProgram(m modelState) error {
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("program run failed: %w", err)
	}
	return nil
}

func (m modelState) Init() tea.Cmd {
	cmds := []tea.Cmd{textinput.Blink}
	if m.state == stateLoading {
		cmds = append(cmds, m.spinner.Tick, searchCmd(m.textInput.Value(), m.searchType, m.sources))
	}
	return tea.Batch(cmds...)
}

func (m modelState) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m, cmd = m.handleGlobalMessages(msg)
	
	if cmd != nil {
		return m, cmd
	}
	
	return m.handleStateUpdate(msg)
}

func (m modelState) handleGlobalMessages(msg tea.Msg) (modelState, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.progress.Width = min(msg.Width-10, 50)
	}
	return m, nil
}

func (m modelState) handleStateUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	handlers := map[sessionState]func(tea.Msg) (tea.Model, tea.Cmd){
		stateInput:           m.updateInput,
		stateLoading:         m.updateLoading,
		stateList:            m.updateList,
		statePlaylistResult:  m.updatePlaylistResult,
		stateDownloading:     m.updateDownloading,
		stateSwitching:       m.updateSwitching,
		statePlaying:         m.updatePlaying,
		stateFavorites:       m.updateFavorites,
		stateLocal:           m.updateLocal,
	}

	if handler, ok := handlers[m.state]; ok {
		return handler(msg)
	}
	return m, nil
}

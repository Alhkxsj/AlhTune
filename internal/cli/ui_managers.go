package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/guohuiyuan/music-lib/model"
)

type FavoriteManager struct {
	songs []model.Song
	mu    sync.RWMutex
}

func (m *FavoriteManager) load() {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(favoriteFile)
	if err != nil {
		return
	}
	if err := json.Unmarshal(data, &m.songs); err != nil {
		m.songs = []model.Song{}
	}
}

func (m *FavoriteManager) save() {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := json.MarshalIndent(m.songs, "", "  ")
	if err != nil {
		fmt.Printf("Failed to save favorites: %v\n", err)
		return
	}
	if err := os.WriteFile(favoriteFile, data, 0644); err != nil {
		fmt.Printf("Failed to write favorites file: %v\n", err)
	}
}

func (m *FavoriteManager) add(song model.Song) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, s := range m.songs {
		if s.ID == song.ID {
			return
		}
	}
	m.songs = append(m.songs, song)
}

func (m *FavoriteManager) remove(index int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if index >= 0 && index < len(m.songs) {
		m.songs = append(m.songs[:index], m.songs[index+1:]...)
	}
}

func (m *FavoriteManager) get() []model.Song {
	m.mu.RLock()
	defer m.mu.RUnlock()
	songs := make([]model.Song, len(m.songs))
	copy(songs, m.songs)
	return songs
}

type LocalMusicManager struct {
	songs []model.Song
	mu    sync.RWMutex
}

func (m *LocalMusicManager) scan(dir string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.songs = make([]model.Song, 0)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		fmt.Printf("Failed to scan directory %s: %v\n", dir, err)
		return
	}

	for i, file := range files {
		if file.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if !slices.Contains(localMusicFormats, ext) {
			continue
		}

		name := strings.TrimSuffix(file.Name(), ext)
		info, err := file.Info()
		if err != nil {
			continue
		}

		title, artist := parseLocalFilename(name)
		m.songs = append(m.songs, model.Song{
			ID:     fmt.Sprintf("local_%d", i),
			Name:   title,
			Artist: artist,
			Source: "local",
			Size:   info.Size(),
			URL:    filepath.Join(dir, file.Name()),
		})
	}
}

func parseLocalFilename(name string) (title, artist string) {
	if parts := strings.SplitN(name, " - ", 2); len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return name, "本地音乐"
}

func (m *LocalMusicManager) get() []model.Song {
	m.mu.RLock()
	defer m.mu.RUnlock()
	songs := make([]model.Song, len(m.songs))
	copy(songs, m.songs)
	return songs
}

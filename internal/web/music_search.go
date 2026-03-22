package web

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/Alhkxsj/AlhTune/core"
	"github.com/guohuiyuan/music-lib/model"
)

var defaultRecommendSources = []string{"netease", "qq", "kugou", "kuwo"}

func handleRecommend(c *gin.Context) {
	sources := c.QueryArray("sources")
	if len(sources) == 0 {
		sources = defaultRecommendSources
	}
	playlists := fetchRecommendPlaylists(sources)
	
	config := IndexRenderConfig{
		Playlists:  playlists,
		Keyword:    "🔥 每日推荐",
		Selected:   sources,
		SearchType: "playlist",
	}
	renderIndex(c, config)
}

func fetchRecommendPlaylists(sources []string) []model.Playlist {
	var (
		result []model.Playlist
		wg     sync.WaitGroup
		mu     sync.Mutex
	)
	for _, src := range sources {
		fn := core.GetRecommendFunc(src)
		if fn == nil {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := fn()
			if err != nil || len(res) == 0 {
				return
			}
			mu.Lock()
			result = append(result, res...)
			mu.Unlock()
		}()
	}
	wg.Wait()
	return result
}

func handleSearch(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("q"))
	searchType := c.DefaultQuery("type", "song")
	sources := c.QueryArray("sources")
	if len(sources) == 0 {
		sources = defaultSearchSources(searchType)
	}

	if strings.HasPrefix(keyword, "http") {
		songs, playlists, resolvedType, errMsg := searchByURL(keyword, searchType)
		config := IndexRenderConfig{
			Songs:      songs,
			Playlists:  playlists,
			Keyword:    keyword,
			Selected:   sources,
			ErrorMsg:   errMsg,
			SearchType: resolvedType,
		}
		renderIndex(c, config)
		return
	}

	songs, playlists := searchByKeyword(keyword, searchType, sources)
	
	config := IndexRenderConfig{
		Songs:      songs,
		Playlists:  playlists,
		Keyword:    keyword,
		Selected:   sources,
		SearchType: searchType,
	}
	renderIndex(c, config)
}

func defaultSearchSources(searchType string) []string {
	if searchType == "playlist" {
		return core.GetPlaylistSourceNames()
	}
	return core.GetDefaultSourceNames()
}

func searchByURL(keyword, searchType string) ([]model.Song, []model.Playlist, string, string) {
	src := core.DetectSource(keyword)
	if src == "" {
		return nil, nil, searchType, "不支持该链接的解析，或无法识别来源"
	}

	if songs, resolvedType, ok := tryParseSongURL(src, keyword); ok {
		return songs, nil, resolvedType, ""
	}

	if songs, playlists, resolvedType, ok := tryParsePlaylistURL(src, keyword, searchType); ok {
		return songs, playlists, resolvedType, ""
	}

	errMsg := fmt.Sprintf("解析失败: 暂不支持 %s 平台的此链接类型或解析出错", src)
	return nil, nil, searchType, errMsg
}

func tryParseSongURL(src, keyword string) ([]model.Song, string, bool) {
	parseFn := core.GetParseFunc(src)
	if parseFn == nil {
		return nil, "", false
	}
	song, err := parseFn(keyword)
	if err != nil {
		return nil, "", false
	}
	return []model.Song{*song}, "song", true
}

func tryParsePlaylistURL(src, keyword, searchType string) ([]model.Song, []model.Playlist, string, bool) {
	parseFn := core.GetParsePlaylistFunc(src)
	if parseFn == nil {
		return nil, nil, "", false
	}
	playlist, songs, err := parseFn(keyword)
	if err != nil {
		return nil, nil, "", false
	}
	if searchType == "playlist" {
		return nil, []model.Playlist{*playlist}, searchType, true
	}
	return songs, nil, "song", true
}

func searchByKeyword(keyword, searchType string, sources []string) ([]model.Song, []model.Playlist) {
	var (
		allSongs     []model.Song
		allPlaylists []model.Playlist
		wg           sync.WaitGroup
		mu           sync.Mutex
	)
	for _, src := range sources {
		wg.Add(1)
		if searchType == "playlist" {
			go func() {
				defer wg.Done()
				searchPlaylistsFromSource(src, keyword, &allPlaylists, &mu)
			}()
		} else {
			go func() {
				defer wg.Done()
				searchSongsFromSource(src, keyword, &allSongs, &mu)
			}()
		}
	}
	wg.Wait()
	return allSongs, allPlaylists
}

func searchSongsFromSource(source, keyword string, result *[]model.Song, mu *sync.Mutex) {
	fn := core.GetSearchFunc(source)
	if fn == nil {
		return
	}
	res, err := fn(keyword)
	if err != nil {
		return
	}
	for i := range res {
		res[i].Source = source
	}
	mu.Lock()
	*result = append(*result, res...)
	mu.Unlock()
}

func searchPlaylistsFromSource(source, keyword string, result *[]model.Playlist, mu *sync.Mutex) {
	fn := core.GetPlaylistSearchFunc(source)
	if fn == nil {
		return
	}
	res, err := fn(keyword)
	if err != nil {
		return
	}
	mu.Lock()
	*result = append(*result, res...)
	mu.Unlock()
}

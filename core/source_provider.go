package core

import (
	"github.com/guohuiyuan/music-lib/bilibili"
	"github.com/guohuiyuan/music-lib/fivesing"
	"github.com/guohuiyuan/music-lib/jamendo"
	"github.com/guohuiyuan/music-lib/joox"
	"github.com/guohuiyuan/music-lib/kugou"
	"github.com/guohuiyuan/music-lib/kuwo"
	"github.com/guohuiyuan/music-lib/migu"
	"github.com/guohuiyuan/music-lib/model"
	"github.com/guohuiyuan/music-lib/netease"
	"github.com/guohuiyuan/music-lib/qianqian"
	"github.com/guohuiyuan/music-lib/qq"
	"github.com/guohuiyuan/music-lib/soda"
)

// Function types for source operations
type (
	SearchFunc           func(keyword string) ([]model.Song, error)
	SearchPlaylistFunc   func(keyword string) ([]model.Playlist, error)
	GetPlaylistSongsFunc func(playlistID string) ([]model.Song, error)
	GetRecommendFuncType func() ([]model.Playlist, error)
	GetDownloadURLFunc   func(s *model.Song) (string, error)
	GetLyricFunc         func(s *model.Song) (string, error)
	ParseSongFunc        func(url string) (*model.Song, error)
	ParsePlaylistFunc    func(url string) (*model.Playlist, []model.Song, error)
)

// SourceProvider provides music source operations
type SourceProvider struct {
	Search           SearchFunc
	SearchPlaylist   SearchPlaylistFunc
	GetPlaylistSongs GetPlaylistSongsFunc
	GetRecommend     GetRecommendFuncType
	GetDownload      GetDownloadURLFunc
	GetLyric         GetLyricFunc
	Parse            ParseSongFunc
	ParsePlaylist    ParsePlaylistFunc
}

// newSourceProvider creates a netease source provider
func newSourceProvider(cookie string) *SourceProvider {
	n := netease.New(cookie)
	return &SourceProvider{
		Search:           n.Search,
		SearchPlaylist:   n.SearchPlaylist,
		GetPlaylistSongs: n.GetPlaylistSongs,
		GetRecommend:     n.GetRecommendedPlaylists,
		GetDownload:      n.GetDownloadURL,
		GetLyric:         n.GetLyrics,
		Parse:            n.Parse,
		ParsePlaylist:    n.ParsePlaylist,
	}
}

// sourceRegistry maps source names to provider factories
var sourceRegistry = map[string]func(cookie string) *SourceProvider{
	"netease": func(cookie string) *SourceProvider {
		return newSourceProvider(cookie)
	},
	"qq": func(cookie string) *SourceProvider {
		c := qq.New(cookie)
		return &SourceProvider{
			Search:           c.Search,
			SearchPlaylist:   c.SearchPlaylist,
			GetPlaylistSongs: c.GetPlaylistSongs,
			GetRecommend:     c.GetRecommendedPlaylists,
			GetDownload:      c.GetDownloadURL,
			GetLyric:         c.GetLyrics,
			Parse:            c.Parse,
			ParsePlaylist:    c.ParsePlaylist,
		}
	},
	"kugou": func(cookie string) *SourceProvider {
		c := kugou.New(cookie)
		return &SourceProvider{
			Search:           c.Search,
			SearchPlaylist:   c.SearchPlaylist,
			GetPlaylistSongs: c.GetPlaylistSongs,
			GetRecommend:     c.GetRecommendedPlaylists,
			GetDownload:      c.GetDownloadURL,
			GetLyric:         c.GetLyrics,
			Parse:            c.Parse,
			ParsePlaylist:    c.ParsePlaylist,
		}
	},
	"kuwo": func(cookie string) *SourceProvider {
		c := kuwo.New(cookie)
		return &SourceProvider{
			Search:           c.Search,
			SearchPlaylist:   c.SearchPlaylist,
			GetPlaylistSongs: c.GetPlaylistSongs,
			GetRecommend:     c.GetRecommendedPlaylists,
			GetDownload:      c.GetDownloadURL,
			GetLyric:         c.GetLyrics,
			Parse:            c.Parse,
			ParsePlaylist:    c.ParsePlaylist,
		}
	},
	"migu": func(cookie string) *SourceProvider {
		c := migu.New(cookie)
		return &SourceProvider{
			Search:      c.Search,
			GetDownload: c.GetDownloadURL,
			GetLyric:    c.GetLyrics,
			Parse:       c.Parse,
			SearchPlaylist: func(string) ([]model.Playlist, error) {
				return nil, nil // migu does not support playlist
			},
			GetPlaylistSongs: func(string) ([]model.Song, error) {
				return nil, nil // migu does not support playlist
			},
		}
	},
	"bilibili": func(cookie string) *SourceProvider {
		c := bilibili.New(cookie)
		return &SourceProvider{
			Search:           c.Search,
			SearchPlaylist:   c.SearchPlaylist,
			GetPlaylistSongs: c.GetPlaylistSongs,
			GetDownload:      c.GetDownloadURL,
			GetLyric:         c.GetLyrics,
			Parse:            c.Parse,
			ParsePlaylist:    c.ParsePlaylist,
		}
	},
	"fivesing": func(cookie string) *SourceProvider {
		c := fivesing.New(cookie)
		return &SourceProvider{
			Search:           c.Search,
			SearchPlaylist:   c.SearchPlaylist,
			GetPlaylistSongs: c.GetPlaylistSongs,
			GetDownload:      c.GetDownloadURL,
			GetLyric:         c.GetLyrics,
			Parse:            c.Parse,
			ParsePlaylist:    c.ParsePlaylist,
		}
	},
	"jamendo": func(cookie string) *SourceProvider {
		c := jamendo.New(cookie)
		return &SourceProvider{
			Search:      c.Search,
			GetDownload: c.GetDownloadURL,
			GetLyric:    c.GetLyrics,
			Parse:       c.Parse,
			SearchPlaylist: func(string) ([]model.Playlist, error) {
				return nil, nil // jamendo does not support playlist
			},
			GetPlaylistSongs: func(string) ([]model.Song, error) {
				return nil, nil // jamendo does not support playlist
			},
		}
	},
	"joox": func(cookie string) *SourceProvider {
		c := joox.New(cookie)
		return &SourceProvider{
			Search:      c.Search,
			GetDownload: c.GetDownloadURL,
			GetLyric:    c.GetLyrics,
			SearchPlaylist: func(string) ([]model.Playlist, error) {
				return nil, nil // joox does not support playlist
			},
			GetPlaylistSongs: func(string) ([]model.Song, error) {
				return nil, nil // joox does not support playlist
			},
		}
	},
	"qianqian": func(cookie string) *SourceProvider {
		c := qianqian.New(cookie)
		return &SourceProvider{
			Search:      c.Search,
			GetDownload: c.GetDownloadURL,
			GetLyric:    c.GetLyrics,
			SearchPlaylist: func(string) ([]model.Playlist, error) {
				return nil, nil // qianqian does not support playlist
			},
			GetPlaylistSongs: func(string) ([]model.Song, error) {
				return nil, nil // qianqian does not support playlist
			},
		}
	},
	"soda": func(cookie string) *SourceProvider {
		c := soda.New(cookie)
		return &SourceProvider{
			Search:           c.Search,
			SearchPlaylist:   c.SearchPlaylist,
			GetPlaylistSongs: c.GetPlaylistSongs,
			GetDownload:      c.GetDownloadURL,
			GetLyric:         c.GetLyrics,
			Parse:            c.Parse,
			ParsePlaylist:    c.ParsePlaylist,
		}
	},
}

// getSourceProvider retrieves a source provider by name
func getSourceProvider(source string) *SourceProvider {
	factory, ok := sourceRegistry[source]
	if !ok {
		return nil
	}
	return factory(CM.Get(source))
}

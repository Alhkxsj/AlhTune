package core

import (
	"testing"
)

func TestDetectSource(t *testing.T) {
	tests := []struct {
		name     string
		link     string
		expected string
	}{
		{"netease", "https://music.163.com/#/song?id=123", "netease"},
		{"qq", "https://y.qq.com/n/ryqq/songDetail/123", "qq"},
		{"kugou", "https://www.kugou.com/song/#hash=123", "kugou"},
		{"kuwo", "http://www.kuwo.cn/play_detail/123", "kuwo"},
		{"migu", "https://music.migu.cn/v3/music/song/123", "migu"},
		{"bilibili", "https://www.bilibili.com/video/BV123", "bilibili"},
		{"b23", "https://b23.tv/abc123", "bilibili"},
		{"soda_douyin", "https://www.douyin.com/video/123", "soda"},
		{"soda_qishui", "https://www.qishui.com/video/123", "soda"},
		{"jamendo", "https://www.jamendo.com/track/123", "jamendo"},
		{"fivesing", "http://5sing.kugou.com/original/123.html", "fivesing"},
		{"unknown", "https://example.com/song/123", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectSource(tt.link)
			if result != tt.expected {
				t.Errorf("DetectSource(%q) = %q, want %q", tt.link, result, tt.expected)
			}
		})
	}
}

func TestGetOriginalLink(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		id       string
		typeStr  string
		expected string
	}{
		{"netease_song", "netease", "123", "song", "https://music.163.com/#/song?id=123"},
		{"netease_playlist", "netease", "456", "playlist", "https://music.163.com/#/playlist?id=456"},
		{"qq_song", "qq", "123", "song", "https://y.qq.com/n/ryqq/songDetail/123"},
		{"qq_playlist", "qq", "456", "playlist", "https://y.qq.com/n/ryqq/playlist/456"},
		{"kugou_song", "kugou", "abc", "song", "https://www.kugou.com/song/#hash=abc"},
		{"kugou_playlist", "kugou", "789", "playlist", "https://www.kugou.com/yy/special/single/789.html"},
		{"kuwo_song", "kuwo", "123", "song", "http://www.kuwo.cn/play_detail/123"},
		{"kuwo_playlist", "kuwo", "456", "playlist", "http://www.kuwo.cn/playlist_detail/456"},
		{"migu_song", "migu", "123", "song", "https://music.migu.cn/v3/music/song/123"},
		{"migu_playlist", "migu", "456", "playlist", ""},
		{"bilibili", "bilibili", "BV123", "song", "https://www.bilibili.com/video/BV123"},
		{"fivesing_with_slash", "fivesing", "original/123", "song", "http://5sing.kugou.com/original/123.html"},
		{"fivesing_without_slash", "fivesing", "123", "song", ""},
		{"unknown", "unknown", "123", "song", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetOriginalLink(tt.source, tt.id, tt.typeStr)
			if result != tt.expected {
				t.Errorf("GetOriginalLink(%q, %q, %q) = %q, want %q",
					tt.source, tt.id, tt.typeStr, result, tt.expected)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name     string
		size     int64
		expected string
	}{
		{"zero", 0, "-"},
		{"negative", -100, "-"},
		{"1MB", 1024 * 1024, "1.0 MB"},
		{"1.5MB", 1024*1024 + 1024*512, "1.5 MB"},
		{"10MB", 10 * 1024 * 1024, "10.0 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSize(tt.size)
			if result != tt.expected {
				t.Errorf("FormatSize(%d) = %q, want %q", tt.size, result, tt.expected)
			}
		})
	}
}

func TestDetectAudioExt(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{"wma", []byte{0x30, 0x26, 0xB2, 0x75, 0x8E, 0x66, 0xCF, 0x11, 0xA6, 0xD9, 0x00, 0xAA, 0x00, 0x62, 0xCE, 0x6C}, "wma"},
		{"flac", []byte{'f', 'L', 'a', 'C'}, "flac"},
		{"mp3_id3", []byte{'I', 'D', '3', 0x03}, "mp3"},
		{"mp3_frame", []byte{0xFF, 0xFB, 0x00, 0x00}, "mp3"},
		{"ogg", []byte{'O', 'g', 'S', 'S'}, "ogg"},
		{"m4a_needs_12_bytes", []byte{0x00, 0x00, 0x00, 0x18, 'f', 't', 'y', 'p', 0x00, 0x00, 0x00, 0x00}, "mp3"},
		{"unknown", []byte{0x00, 0x01, 0x02, 0x03}, "mp3"},
		{"empty", []byte{}, "mp3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectAudioExt(tt.data)
			if result != tt.expected {
				t.Errorf("DetectAudioExt(%v) = %q, want %q", tt.data, result, tt.expected)
			}
		})
	}
}

func TestDetectAudioExtByContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expected    string
	}{
		{"flac", "audio/flac", "flac"},
		{"flac_x_flac", "audio/x-flac", "flac"},
		{"wma", "audio/x-ms-wma", "wma"},
		{"mp3", "audio/mpeg", "mp3"},
		{"ogg", "audio/ogg", "ogg"},
		{"m4a", "audio/mp4", "m4a"},
		{"with_charset", "audio/mpeg; charset=utf-8", "mp3"},
		{"unknown", "audio/unknown", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectAudioExtByContentType(tt.contentType)
			if result != tt.expected {
				t.Errorf("DetectAudioExtByContentType(%q) = %q, want %q",
					tt.contentType, result, tt.expected)
			}
		})
	}
}

func TestAudioMimeByExt(t *testing.T) {
	tests := []struct {
		name     string
		ext      string
		expected string
	}{
		{"wma", "wma", "audio/x-ms-wma"},
		{"flac", "flac", "audio/flac"},
		{"ogg", "ogg", "audio/ogg"},
		{"m4a", "m4a", "audio/mp4"},
		{"mp3", "mp3", "audio/mpeg"},
		{"unknown", "unknown", "audio/mpeg"},
		{"empty", "", "audio/mpeg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AudioMimeByExt(tt.ext)
			if result != tt.expected {
				t.Errorf("AudioMimeByExt(%q) = %q, want %q", tt.ext, result, tt.expected)
			}
		})
	}
}

func TestIsDurationClose(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected bool
	}{
		{"same", 180, 180, true},
		{"close_5s", 180, 185, true},
		{"close_10s", 180, 190, true},
		{"close 15percent", 100, 114, true},
		{"not close", 180, 220, false},
		{"zero a", 0, 180, true},
		{"zero b", 180, 0, true},
		{"both zero", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDurationClose(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("IsDurationClose(%d, %d) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestIntAbs(t *testing.T) {
	tests := []struct {
		name     string
		x        int
		expected int
	}{
		{"positive", 5, 5},
		{"negative", -5, 5},
		{"zero", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IntAbs(tt.x)
			if result != tt.expected {
				t.Errorf("IntAbs(%d) = %d, want %d", tt.x, result, tt.expected)
			}
		})
	}
}

func TestCalcSongSimilarity(t *testing.T) {
	tests := []struct {
		name       string
		name1      string
		artist1    string
		name2      string
		artist2    string
		minScore   float64
		maxScore   float64
	}{
		{"same_song", "hello", "taylor", "hello", "taylor", 0.99, 1.0},
		{"same_name_diff_artist", "hello", "taylor", "hello", "adele", 0.6, 0.8},
		{"diff_song_same_artist", "hello", "taylor", "world", "taylor", 0.3, 0.6},
		{"completely different", "hello", "taylor", "goodbye", "adele", 0, 0.3},
		{"empty name", "", "taylor", "hello", "taylor", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalcSongSimilarity(tt.name1, tt.artist1, tt.name2, tt.artist2)
			if result < tt.minScore || result > tt.maxScore {
				t.Errorf("CalcSongSimilarity(%q, %q, %q, %q) = %f, want [%f, %f]",
					tt.name1, tt.artist1, tt.name2, tt.artist2, result, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestGetAllSourceNames(t *testing.T) {
	names := GetAllSourceNames()
	expected := []string{
		"netease", "qq", "kugou", "kuwo", "migu",
		"fivesing", "jamendo", "joox", "qianqian", "soda", "bilibili",
	}

	if len(names) != len(expected) {
		t.Errorf("GetAllSourceNames() returned %d items, want %d", len(names), len(expected))
		return
	}

	for i, name := range names {
		if name != expected[i] {
			t.Errorf("GetAllSourceNames()[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestGetDefaultSourceNames(t *testing.T) {
	names := GetDefaultSourceNames()
	excluded := map[string]bool{"bilibili": true, "joox": true, "jamendo": true, "fivesing": true}

	for _, name := range names {
		if excluded[name] {
			t.Errorf("GetDefaultSourceNames() should not include %q", name)
		}
	}
}

func TestGetSourceDescription(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{"netease", "netease", "网易云音乐"},
		{"qq", "qq", "QQ 音乐"},
		{"unknown", "unknown", "未知音乐源"},
		{"empty", "", "未知音乐源"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetSourceDescription(tt.source)
			if result != tt.expected {
				t.Errorf("GetSourceDescription(%q) = %q, want %q", tt.source, result, tt.expected)
			}
		})
	}
}

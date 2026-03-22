package core

import (
	"net/http"
	"strings"
)

// sourceLinkRules maps keywords to source names for link detection
var sourceLinkRules = []struct {
	keyword string
	source  string
}{
	{"163.com", "netease"},
	{"qq.com", "qq"},
	{"5sing", "fivesing"},
	{"kugou.com", "kugou"},
	{"kuwo.cn", "kuwo"},
	{"migu.cn", "migu"},
	{"bilibili.com", "bilibili"},
	{"b23.tv", "bilibili"},
	{"douyin.com", "soda"},
	{"qishui", "soda"},
	{"jamendo.com", "jamendo"},
}

// DetectSource detects the music source from a link
func DetectSource(link string) string {
	for _, rule := range sourceLinkRules {
		if strings.Contains(link, rule.keyword) {
			return rule.source
		}
	}
	return ""
}

// sourceLinkTemplates maps source names to link generators
var sourceLinkTemplates = map[string]func(id, typeStr string) string{
	"netease": func(id, typeStr string) string {
		if typeStr == "playlist" {
			return "https://music.163.com/#/playlist?id=" + id
		}
		return "https://music.163.com/#/song?id=" + id
	},
	"qq": func(id, typeStr string) string {
		if typeStr == "playlist" {
			return "https://y.qq.com/n/ryqq/playlist/" + id
		}
		return "https://y.qq.com/n/ryqq/songDetail/" + id
	},
	"kugou": func(id, typeStr string) string {
		if typeStr == "playlist" {
			return "https://www.kugou.com/yy/special/single/" + id + ".html"
		}
		return "https://www.kugou.com/song/#hash=" + id
	},
	"kuwo": func(id, typeStr string) string {
		if typeStr == "playlist" {
			return "http://www.kuwo.cn/playlist_detail/" + id
		}
		return "http://www.kuwo.cn/play_detail/" + id
	},
	"migu": func(id, typeStr string) string {
		if typeStr == "song" {
			return "https://music.migu.cn/v3/music/song/" + id
		}
		return ""
	},
	"bilibili": func(id, _ string) string {
		return "https://www.bilibili.com/video/" + id
	},
	"fivesing": func(id, _ string) string {
		if strings.Contains(id, "/") {
			return "http://5sing.kugou.com/" + id + ".html"
		}
		return ""
	},
}

// GetOriginalLink gets the original link for a song or playlist
func GetOriginalLink(source, id, typeStr string) string {
	if fn, ok := sourceLinkTemplates[source]; ok {
		return fn(id, typeStr)
	}
	return ""
}

// sourceRequestConfig configures HTTP requests for specific sources
var sourceRequestConfig = map[string]func(req *http.Request){
	"bilibili": func(req *http.Request) {
		req.Header.Set("Referer", RefBilibili)
	},
	"migu": func(req *http.Request) {
		req.Header.Set("User-Agent", UAMobile)
		req.Header.Set("Referer", RefMigu)
	},
	"qq": func(req *http.Request) {
		req.Header.Set("Referer", "http://y.qq.com")
	},
}

// BuildSourceRequest builds an HTTP request for a music source
func BuildSourceRequest(method, urlStr, source, rangeHeader string) (*http.Request, error) {
	req, err := http.NewRequest(method, urlStr, nil)
	if err != nil {
		return nil, err
	}

	if rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}

	req.Header.Set("User-Agent", UACommon)

	if configFn, ok := sourceRequestConfig[source]; ok {
		configFn(req)
	}

	if cookie := CM.Get(source); cookie != "" {
		req.Header.Set("Cookie", cookie)
	}

	return req, nil
}

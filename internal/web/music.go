package web

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/Alhkxsj/AlhTune/core"
	"github.com/guohuiyuan/music-lib/model"
	"github.com/guohuiyuan/music-lib/utils"
)

// RegisterMusicRoutes registers all music-related HTTP handlers.
func RegisterMusicRoutes(api *gin.RouterGroup) {
	api.GET("/", handleIndex)
	api.GET("/recommend", handleRecommend)
	api.GET("/search", handleSearch)
	api.GET("/playlist", handlePlaylist)
	api.GET("/inspect", handleInspect)
	api.GET("/switch_source", handleSwitchSource)
	api.GET("/download", handleDownload)
	api.GET("/download_lrc", handleDownloadLyric)
	api.GET("/download_cover", handleDownloadCover)
	api.GET("/lyric", handleLyric)
}

func handleIndex(c *gin.Context) {
	config := IndexRenderConfig{
		SearchType: "song",
	}
	renderIndex(c, config)
}

func handlePlaylist(c *gin.Context) {
	id := c.Query("id")
	src := c.Query("source")
	if id == "" || src == "" {
		config := IndexRenderConfig{
			ErrorMsg:   "缺少参数",
			SearchType: "song",
		}
		renderIndex(c, config)
		return
	}

	fn := core.GetPlaylistDetailFunc(src)
	if fn == nil {
		config := IndexRenderConfig{
			ErrorMsg:   "该源不支持查看歌单详情",
			SearchType: "song",
		}
		renderIndex(c, config)
		return
	}

	songs, err := fn(id)
	errMsg := ""
	if err != nil {
		errMsg = fmt.Sprintf("获取歌单失败: %v", err)
	}
	playlistLink := core.GetOriginalLink(src, id, "playlist")
	
	config := IndexRenderConfig{
		Songs:        songs,
		Selected:     []string{src},
		ErrorMsg:     errMsg,
		SearchType:   "song",
		PlaylistLink: playlistLink,
	}
	renderIndex(c, config)
}

func handleLyric(c *gin.Context) {
	id := c.Query("id")
	src := c.Query("source")

	if fn := core.GetLyricFuncFromSource(src); fn != nil {
		if lrc, _ := fn(&model.Song{ID: id, Source: src}); lrc != "" {
			c.String(200, lrc)
			return
		}
	}
	c.String(200, "[00:00.00] 暂无歌词")
}

func handleDownloadLyric(c *gin.Context) {
	id := c.Query("id")
	src := c.Query("source")
	name := c.Query("name")
	artist := c.Query("artist")

	fn := core.GetLyricFuncFromSource(src)
	if fn == nil {
		c.String(404, "No support")
		return
	}

	lrc, err := fn(&model.Song{ID: id, Source: src})
	if err != nil || lrc == "" {
		c.String(404, "Lyric not found")
		return
	}

	filename := fmt.Sprintf("%s - %s.lrc", name, artist)
	setDownloadHeader(c, filename)
	c.String(200, lrc)
}

func handleDownloadCover(c *gin.Context) {
	u := c.Query("url")
	if u == "" {
		return
	}
	resp, err := utils.Get(u, utils.WithHeader("User-Agent", core.UACommon))
	if err != nil {
		return
	}
	filename := fmt.Sprintf("%s - %s.jpg", c.Query("name"), c.Query("artist"))
	setDownloadHeader(c, filename)
	c.Data(200, "image/jpeg", resp)
}

// parseSongExtraQuery parses the JSON-encoded "extra" query parameter into a string map.
func parseSongExtraQuery(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil
	}

	extra := make(map[string]string, len(decoded))
	for key, value := range decoded {
		extra[key] = stringifyValue(value)
	}
	if len(extra) == 0 {
		return nil
	}
	return extra
}

func stringifyValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', 0, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(b)
	}
}

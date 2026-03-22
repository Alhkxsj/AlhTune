package web

import (
	"encoding/json"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/guohuiyuan/music-lib/model"
)

// RegisterCollectionRoutes registers collection CRUD routes
func RegisterCollectionRoutes(api *gin.RouterGroup) {
	repo := NewCollectionRepository(db)

	registerMyCollectionsRoute(api, repo)
	registerCollectionDetailRoute(api, repo)
	registerCollectionAPIRoutes(api, repo)
}

// registerMyCollectionsRoute handles GET /my_collections
func registerMyCollectionsRoute(api *gin.RouterGroup, repo *CollectionRepository) {
	api.GET("/my_collections", func(c *gin.Context) {
		collections, err := repo.List()
		if err != nil {
			config := IndexRenderConfig{
				Keyword:        "我的自制歌单",
				ErrorMsg:       "获取收藏夹失败",
				SearchType:     "playlist",
				IsLocalColPage: true,
			}
			renderIndex(c, config)
			return
		}

		playlists := buildPlaylistsFromCollections(repo, collections)
		config := IndexRenderConfig{
			Playlists:      playlists,
			Keyword:        "我的自制歌单",
			SearchType:     "playlist",
			IsLocalColPage: true,
		}
		renderIndex(c, config)
	})
}

// buildPlaylistsFromCollections converts collections to playlists
func buildPlaylistsFromCollections(repo *CollectionRepository, collections []Collection) []model.Playlist {
	playlists := make([]model.Playlist, 0, len(collections))
	for _, col := range collections {
		count, _ := repo.CountSongs(col.ID)

		cvr := col.Cover
		if cvr == "" {
			cvr = "https://picsum.photos/seed/col_" + itoa(col.ID) + "/400/400"
		}

		playlists = append(playlists, model.Playlist{
			ID:          itoa(col.ID),
			Name:        col.Name,
			Description: col.Description,
			Cover:       cvr,
			Creator:     "我自己",
			TrackCount:  int(count),
			Source:      "local",
		})
	}
	return playlists
}

// registerCollectionDetailRoute handles GET /collection
func registerCollectionDetailRoute(api *gin.RouterGroup, repo *CollectionRepository) {
	api.GET("/collection", func(c *gin.Context) {
		idStr := c.Query("id")
		if idStr == "" {
			config := IndexRenderConfig{
				ErrorMsg:   "缺少参数",
				SearchType: "song",
			}
			renderIndex(c, config)
			return
		}

		id, err := parseCollectionID(idStr)
		if err != nil {
			config := IndexRenderConfig{
				ErrorMsg:   "无效的歌单 ID",
				SearchType: "song",
			}
			renderIndex(c, config)
			return
		}

		col, err := repo.Get(id)
		if err != nil {
			config := IndexRenderConfig{
				ErrorMsg:   "自制歌单不存在",
				SearchType: "song",
			}
			renderIndex(c, config)
			return
		}

		savedSongs, err := repo.GetSongs(id)
		if err != nil {
			savedSongs = []SavedSong{}
		}

		songs := convertToSongs(savedSongs)
		config := IndexRenderConfig{
			Songs:          songs,
			SearchType:     "song",
			CollectionID:   idStr,
			CollectionName: col.Name,
		}
		renderIndex(c, config)
	})
}

// convertToSongs converts SavedSong slice to Song slice
func convertToSongs(savedSongs []SavedSong) []model.Song {
	songs := make([]model.Song, 0, len(savedSongs))
	for _, ss := range savedSongs {
		songs = append(songs, model.Song{
			ID:       ss.SongID,
			Source:   ss.Source,
			Name:     ss.Name,
			Artist:   ss.Artist,
			Cover:    ss.Cover,
			Duration: ss.Duration,
		})
	}
	return songs
}

// registerCollectionAPIRoutes handles RESTful API routes for collections
func registerCollectionAPIRoutes(api *gin.RouterGroup, repo *CollectionRepository) {
	colApi := api.Group("/collections")

	colApi.GET("", listCollectionsHandler(repo))
	colApi.POST("", createCollectionHandler(repo))
	colApi.PUT("/:id", updateCollectionHandler(repo))
	colApi.DELETE("/:id", deleteCollectionHandler(repo))
	colApi.GET("/:id/songs", listCollectionSongsHandler(repo))
	colApi.POST("/:id/songs", addSongToCollectionHandler(repo))
	colApi.DELETE("/:id/songs", removeSongFromCollectionHandler(repo))
}

// listCollectionsHandler handles GET /collections
func listCollectionsHandler(repo *CollectionRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		collections, err := repo.List()
		if err != nil {
			c.JSON(500, gin.H{"error": "获取歌单列表失败：" + err.Error()})
			return
		}
		c.JSON(200, collections)
	}
}

// createCollectionHandler handles POST /collections
func createCollectionHandler(repo *CollectionRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req Collection
		if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
			c.JSON(400, gin.H{"error": "参数错误，必须提供歌单名"})
			return
		}

		col, err := repo.Create(req.Name, req.Description, req.Cover)
		if err != nil {
			c.JSON(500, gin.H{"error": "创建失败：" + err.Error()})
			return
		}
		c.JSON(200, gin.H{"id": col.ID, "name": col.Name})
	}
}

// updateCollectionHandler handles PUT /collections/:id
func updateCollectionHandler(repo *CollectionRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := parseCollectionID(idStr)
		if err != nil {
			c.JSON(400, gin.H{"error": "无效的歌单 ID"})
			return
		}

		var req Collection
		if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
			c.JSON(400, gin.H{"error": "参数错误"})
			return
		}

		if err := repo.Update(id, req.Name, req.Description, req.Cover); err != nil {
			c.JSON(500, gin.H{"error": "更新失败"})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	}
}

// deleteCollectionHandler handles DELETE /collections/:id
func deleteCollectionHandler(repo *CollectionRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := parseCollectionID(idStr)
		if err != nil {
			c.JSON(400, gin.H{"error": "无效的歌单 ID"})
			return
		}

		if err := repo.Delete(id); err != nil {
			c.JSON(500, gin.H{"error": "删除失败"})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	}
}

// listCollectionSongsHandler handles GET /collections/:id/songs
func listCollectionSongsHandler(repo *CollectionRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := parseCollectionID(idStr)
		if err != nil {
			c.JSON(400, gin.H{"error": "无效的歌单 ID"})
			return
		}

		savedSongs, err := repo.GetSongs(id)
		if err != nil {
			savedSongs = []SavedSong{}
		}

		resp := buildSongsResponse(savedSongs)
		c.JSON(200, resp)
	}
}

// buildSongsResponse builds the JSON response for songs
func buildSongsResponse(savedSongs []SavedSong) []gin.H {
	resp := make([]gin.H, 0, len(savedSongs))
	for _, s := range savedSongs {
		var extraObj interface{}
		if err := unmarshalExtra(s.Extra, &extraObj); err != nil {
			extraObj = s.Extra
		}
		resp = append(resp, gin.H{
			"db_id":         s.ID,
			"collection_id": s.CollectionID,
			"id":            s.SongID,
			"source":        s.Source,
			"extra":         extraObj,
			"name":          s.Name,
			"artist":        s.Artist,
			"cover":         s.Cover,
			"duration":      s.Duration,
			"added_at":      s.AddedAt,
		})
	}
	return resp
}

// unmarshalExtra decodes JSON extra data
func unmarshalExtra(extraStr string, target interface{}) error {
	return json.Unmarshal([]byte(extraStr), target)
}

// addSongToCollectionHandler handles POST /collections/:id/songs
func addSongToCollectionHandler(repo *CollectionRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Extra    interface{} `json:"extra"`
			SongID   string      `json:"id" binding:"required"`
			Source   string      `json:"source" binding:"required"`
			Name     string      `json:"name"`
			Artist   string      `json:"artist"`
			Cover    string      `json:"cover"`
			Duration int         `json:"duration"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "参数错误，缺失 id 或 source"})
			return
		}

		idStr := c.Param("id")
		colID, err := parseCollectionID(idStr)
		if err != nil {
			c.JSON(400, gin.H{"error": "无效的歌单 ID"})
			return
		}

		song := SavedSong{
			CollectionID: colID,
			SongID:       req.SongID,
			Source:       req.Source,
			Name:         req.Name,
			Artist:       req.Artist,
			Cover:        req.Cover,
			Duration:     req.Duration,
			Extra:        encodeExtra(req.Extra),
		}

		if err := repo.AddSong(colID, song); err != nil {
			c.JSON(500, gin.H{"error": "添加失败：" + err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	}
}

// removeSongFromCollectionHandler handles DELETE /collections/:id/songs
func removeSongFromCollectionHandler(repo *CollectionRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		colID, err := parseCollectionID(idStr)
		if err != nil {
			c.JSON(400, gin.H{"error": "无效的歌单 ID"})
			return
		}

		songID := c.Query("id")
		source := c.Query("source")

		if songID == "" || source == "" {
			c.JSON(400, gin.H{"error": "需要通过 query 传递 id 和 source"})
			return
		}

		if err := repo.RemoveSong(colID, songID, source); err != nil {
			c.JSON(500, gin.H{"error": "删除失败"})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	}
}

// itoa converts an integer to string
func itoa(n uint) string {
	return strconv.FormatUint(uint64(n), 10)
}

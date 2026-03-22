package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/Alhkxsj/AlhTune/core"
	"github.com/guohuiyuan/music-lib/model"
)

//go:embed templates/*

var templateFS embed.FS

const RoutePrefix = "/music"

type FeatureFlags struct {
	VgChangeCover bool
	VgChangeAudio bool
	VgChangeLyric bool
	VgExportVideo bool
}

var featureFlags FeatureFlags

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
		c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
		c.Header(
			"Access-Control-Expose-Headers",
			"Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type",
		)
		c.Header("Access-Control-Allow-Credentials", "true")
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		c.Next()
	}
}

func setDownloadHeader(c *gin.Context, filename string) {
	encoded := url.QueryEscape(filename)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"; filename*=utf-8''%s", encoded, encoded))
}

type IndexRenderConfig struct {
	Songs          []model.Song
	Playlists      []model.Playlist
	Keyword        string
	Selected       []string
	ErrorMsg       string
	SearchType     string
	PlaylistLink   string
	CollectionID   string
	CollectionName string
	IsLocalColPage bool
}

func renderIndex(c *gin.Context, config IndexRenderConfig) {
	allSrc := core.GetAllSourceNames()
	desc := make(map[string]string)
	for _, s := range allSrc {
		desc[s] = core.GetSourceDescription(s)
	}

	playlistSupported := make(map[string]bool)
	for _, s := range core.GetPlaylistSourceNames() {
		playlistSupported[s] = true
	}

	c.HTML(200, "index.html", gin.H{
		"Result":             config.Songs,
		"Playlists":          config.Playlists,
		"Keyword":            config.Keyword,
		"AllSources":         allSrc,
		"DefaultSources":     core.GetDefaultSourceNames(),
		"SourceDescriptions": desc,
		"Selected":           config.Selected,
		"Error":              config.ErrorMsg,
		"SearchType":         config.SearchType,
		"PlaylistSupported":  playlistSupported,
		"Root":               RoutePrefix,
		"PlaylistLink":       config.PlaylistLink,
		"ColID":              config.CollectionID,
		"ColName":            config.CollectionName,
		"IsLocalColPage":     config.IsLocalColPage,
		"VgChangeCover":      featureFlags.VgChangeCover,
		"VgChangeAudio":      featureFlags.VgChangeAudio,
		"VgChangeLyric":      featureFlags.VgChangeLyric,
		"VgExportVideo":      featureFlags.VgExportVideo,
	})
}

func Start(port string, shouldOpenBrowser bool, flags FeatureFlags) {
	featureFlags = flags
	if err := core.LoadCookies(); err != nil {
		// Log error but continue - cookies are optional
		fmt.Printf("Warning: failed to load cookies: %v\n", err)
	}
	if err := InitDB(); err != nil {
		fmt.Printf("Warning: failed to initialize database: %v\n", err)
	}
	defer CloseDB()

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(corsMiddleware())

	tmpl := template.Must(template.New("").Funcs(template.FuncMap{
		"tojson": func(v interface{}) string {
			if v == nil {
				return ""
			}
			b, err := json.Marshal(v)
			if err != nil {
				return ""
			}
			return string(b)
		},
	}).ParseFS(templateFS,
		"templates/pages/*.html",
		"templates/partials/*.html",
	))
	r.SetHTMLTemplate(tmpl)

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, RoutePrefix)
	})

	videoDir := "data/video_output"
	_ = os.MkdirAll(videoDir, 0755)

	api := r.Group(RoutePrefix)
	api.Static("/videos", videoDir)

	api.GET("/icon.png", func(c *gin.Context) { c.FileFromFS("templates/static/images/icon.png", http.FS(templateFS)) })
	api.GET("/style.css", func(c *gin.Context) { c.FileFromFS("templates/static/css/style.css", http.FS(templateFS)) })
	api.GET(
		"/videogen.css",
		func(c *gin.Context) { c.FileFromFS("templates/static/css/videogen.css", http.FS(templateFS)) },
	)
	api.GET(
		"/videogen.js",
		func(c *gin.Context) { c.FileFromFS("templates/static/js/videogen.js", http.FS(templateFS)) },
	)
	api.GET("/app.js", func(c *gin.Context) { c.FileFromFS("templates/static/js/app.js", http.FS(templateFS)) })
	api.GET("/render", func(c *gin.Context) {
		c.HTML(200, "render.html", gin.H{
			"Root":          RoutePrefix,
			"VgExportVideo": featureFlags.VgExportVideo,
		})
	})

	api.GET("/cookies", func(c *gin.Context) { c.JSON(200, core.CM.GetAll()) })
	api.POST("/cookies", func(c *gin.Context) {
		var req map[string]string
		if c.ShouldBindJSON(&req) == nil {
			core.CM.SetAll(req)
			if err := core.SaveCookies(); err != nil {
				// Log error but continue
				fmt.Printf("Warning: failed to save cookies: %v\n", err)
			}
			c.JSON(200, gin.H{"status": "ok"})
		}
	})

	RegisterMusicRoutes(api)
	RegisterCollectionRoutes(api)
	RegisterVideogenRoutes(api, videoDir)

	urlStr := "http://localhost:" + port + RoutePrefix
	fmt.Printf("Web started at %s\n", urlStr)
	if shouldOpenBrowser {
		go func() { time.Sleep(500 * time.Millisecond); core.OpenBrowser(urlStr) }()
	}
	_ = r.Run(":" + port)
}

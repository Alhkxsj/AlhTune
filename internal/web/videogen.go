package web

import (
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

// Global session manager instance
var sessionManager = NewSessionManager()

// RegisterVideogenRoutes registers video generation routes
func RegisterVideogenRoutes(api *gin.RouterGroup, videoDir string) {
	startCleanupRoutine(videoDir)

	videoApi := api.Group("/videogen")

	videoApi.POST("/init", func(c *gin.Context) {
		initHandler(c, videoDir)
	})

	videoApi.POST("/frame", frameHandler)

	videoApi.POST("/finish", func(c *gin.Context) {
		finishHandler(c, videoDir)
	})
}

// startCleanupRoutine starts background cleanup of old sessions and files
func startCleanupRoutine(videoDir string) {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			sessionManager.Cleanup(10 * time.Minute)
			CleanupOldFiles(videoDir, 10*time.Minute)
		}
	}()
}

// CleanupOldFiles removes files older than maxAge from the directory
func CleanupOldFiles(dir string, maxAge time.Duration) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	now := time.Now()
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if now.Sub(info.ModTime()) > maxAge {
			os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
}

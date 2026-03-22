package web

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"os/exec"

	"github.com/gin-gonic/gin"
	"github.com/Alhkxsj/AlhTune/core"
	"github.com/guohuiyuan/music-lib/model"
)

// execCommand creates a new command (allows for testing)
var execCommand = exec.Command

// saveBase64File decodes and saves a base64-encoded file
func saveBase64File(dataURI, path string) error {
	if len(dataURI) > 23 {
		dataURI = dataURI[23:]
	}

	data, err := base64.StdEncoding.DecodeString(dataURI)
	if err != nil {
		return fmt.Errorf("decode base64 failed: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// initHandler handles video generation initialization
func initHandler(c *gin.Context, videoDir string) {
	id, source, hasCustomAudio, err := parseInitRequest(c)
	if err != nil {
		c.JSON(400, gin.H{"error": "Args error"})
		return
	}

	if id == "" || source == "" {
		c.JSON(400, gin.H{"error": "Missing id or source"})
		return
	}

	sessionID := buildSessionID(source, id)
	tempDir, err := os.MkdirTemp("", "vg_render_"+sessionID+"_")
	if err != nil {
		c.JSON(500, gin.H{"error": "Create temp dir failed"})
		return
	}

	audioPath := filepath.Join(tempDir, "audio.mp3")
	proxyAudioURL, err := setupAudio(c, audioPath, id, source, hasCustomAudio)
	if err != nil {
		os.RemoveAll(tempDir)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	session := sessionManager.Create(sessionID, tempDir, audioPath)
	c.JSON(200, gin.H{
		"session_id": session.ID,
		"audio_url":  proxyAudioURL,
	})
}

// parseInitRequest parses the init request parameters
func parseInitRequest(c *gin.Context) (id, source string, hasCustomAudio bool, err error) {
	if strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
		id = c.PostForm("id")
		source = c.PostForm("source")
		hasCustomAudio = true
	} else {
		var req struct {
			ID     string `json:"id"`
			Source string `json:"source"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			return "", "", false, err
		}
		id = req.ID
		source = req.Source
	}
	return id, source, hasCustomAudio, nil
}

// buildSessionID creates a unique session ID
func buildSessionID(source, id string) string {
	return fmt.Sprintf("%s_%s_%d", source, id, time.Now().Unix())
}

// setupAudio handles audio download or upload
func setupAudio(c *gin.Context, audioPath, id, source string, hasCustomAudio bool) (string, error) {
	if hasCustomAudio {
		return handleCustomAudio(c, audioPath)
	}
	return handleProxyAudio(c, audioPath, id, source)
}

// handleCustomAudio processes uploaded custom audio
func handleCustomAudio(c *gin.Context, audioPath string) (string, error) {
	file, err := c.FormFile("audio_file")
	if err != nil {
		return "", fmt.Errorf("failed to receive custom audio")
	}

	if err := c.SaveUploadedFile(file, audioPath); err != nil {
		return "", fmt.Errorf("failed to save custom audio")
	}

	return "", nil
}

// handleProxyAudio downloads audio from source
func handleProxyAudio(c *gin.Context, audioPath, id, source string) (string, error) {
	fn := core.GetDownloadFunc(source)
	if fn == nil {
		return "", fmt.Errorf("source not supported")
	}

	audioURL, err := fn(&model.Song{ID: id, Source: source})
	if err != nil {
		return "", fmt.Errorf("audio download failed")
	}

	reqHTTP, err := core.BuildSourceRequest("GET", audioURL, source, "")
	if err != nil {
		return "", fmt.Errorf("build request failed")
	}

	client := &http.Client{}
	resp, err := client.Do(reqHTTP)
	if err != nil {
		return "", fmt.Errorf("download audio failed")
	}
	defer resp.Body.Close()

	out, err := os.Create(audioPath)
	if err != nil {
		return "", fmt.Errorf("save audio failed")
	}

	_, err = io.Copy(out, resp.Body)
	out.Close()

	if err != nil {
		return "", fmt.Errorf("save audio failed")
	}

	proxyAudioURL := fmt.Sprintf("%s/download?id=%s&source=%s", RoutePrefix, url.QueryEscape(id), source)
	return proxyAudioURL, nil
}

// frameHandler handles frame upload requests
func frameHandler(c *gin.Context) {
	var req struct {
		SessionID string   `json:"session_id"`
		Frames    []string `json:"frames"`
		StartIdx  int      `json:"start_idx"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Bad request"})
		return
	}

	session, ok := sessionManager.Get(req.SessionID)
	if !ok {
		c.JSON(404, gin.H{"error": "Session not found"})
		return
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	for i, dataURI := range req.Frames {
		frameNum := req.StartIdx + i
		fileName := filepath.Join(session.Dir, fmt.Sprintf("frame_%05d.jpg", frameNum))
		if err := saveBase64File(dataURI, fileName); err != nil {
			c.JSON(500, gin.H{
				"error": fmt.Sprintf("Save frame %d failed: %v", frameNum, err),
			})
			return
		}
	}

	session.Total += len(req.Frames)
	c.JSON(200, gin.H{"status": "ok", "received": len(req.Frames)})
}

// finishHandler handles video rendering completion
func finishHandler(c *gin.Context, videoDir string) {
	var req struct {
		SessionID string `json:"session_id"`
		Name      string `json:"name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Bad request"})
		return
	}

	session, ok := sessionManager.Get(req.SessionID)
	if !ok {
		c.JSON(404, gin.H{"error": "Session not found"})
		return
	}

	sessionManager.Delete(req.SessionID)
	defer os.RemoveAll(session.Dir)

	outPath, err := renderVideo(session, videoDir)
	if err != nil {
		c.JSON(500, gin.H{"error": "Render failed: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"url": "/videos/" + filepath.Base(outPath)})
}

// renderVideo renders the final video using FFmpeg
func renderVideo(session *RenderSession, videoDir string) (string, error) {
	absVideoDir, _ := filepath.Abs(videoDir)
	outName := fmt.Sprintf("render_%s_%d.mp4", session.ID, time.Now().Unix())
	outPath := filepath.Join(absVideoDir, outName)

	cmd := execCommand(
		"ffmpeg",
		"-y",
		"-framerate", "30",
		"-i", filepath.Join(session.Dir, "frame_%05d.jpg"),
		"-i", session.AudioPath,
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-c:a", "aac",
		"-b:a", "320k",
		"-pix_fmt", "yuv420p",
		"-shortest",
		outPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("FFmpeg Error:", string(output))
		return "", err
	}

	return outPath, nil
}

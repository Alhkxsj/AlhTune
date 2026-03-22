package web

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/Alhkxsj/AlhTune/core"
	"github.com/guohuiyuan/music-lib/model"
	"github.com/guohuiyuan/music-lib/soda"
)

// --- inspect handler ---

func handleInspect(c *gin.Context) {
	id := c.Query("id")
	src := c.Query("source")
	durStr := c.Query("duration")
	extra := parseSongExtraQuery(c.Query("extra"))

	urlStr, err := resolveDownloadURL(id, src, extra)
	if err != nil {
		c.JSON(200, gin.H{"valid": false})
		return
	}

	valid, size := probeURL(urlStr, src)
	bitrate := calcBitrate(valid, size, durStr)

	c.JSON(200, gin.H{
		"valid":   valid,
		"url":     urlStr,
		"size":    core.FormatSize(size),
		"bitrate": bitrate,
	})
}

func resolveDownloadURL(id, source string, extra map[string]string) (string, error) {
	if source == "soda" {
		cookie := core.CM.Get("soda")
		sodaInst := soda.New(cookie)
		info, err := sodaInst.GetDownloadInfo(&model.Song{ID: id, Source: source})
		if err != nil {
			return "", err
		}
		return info.URL, nil
	}

	fn := core.GetDownloadFunc(source)
	if fn == nil {
		return "", fmt.Errorf("unsupported source: %s", source)
	}
	urlStr, err := fn(&model.Song{ID: id, Source: source, Extra: extra})
	if err != nil || urlStr == "" {
		return "", fmt.Errorf("failed to resolve download URL")
	}
	return urlStr, nil
}

func probeURL(urlStr, source string) (bool, int64) {
	req, err := core.BuildSourceRequest("GET", urlStr, source, "bytes=0-1")
	if err != nil {
		return false, 0
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return false, 0
	}
	return true, parseContentSize(resp)
}

func parseContentSize(resp *http.Response) int64 {
	cr := resp.Header.Get("Content-Range")
	if parts := strings.Split(cr, "/"); len(parts) == 2 {
		size, _ := strconv.ParseInt(parts[1], 10, 64)
		return size
	}
	return resp.ContentLength
}

func calcBitrate(valid bool, size int64, durStr string) string {
	if !valid || size <= 0 {
		return "-"
	}
	dur, _ := strconv.Atoi(durStr)
	if dur <= 0 {
		return "-"
	}
	kbps := int((size * 8) / int64(dur) / 1000)
	return fmt.Sprintf("%d kbps", kbps)
}

// --- switch_source handler ---

var skipSources = map[string]bool{"soda": true, "fivesing": true}

type songCandidate struct {
	song    model.Song
	score   float64
	durDiff int
}

func handleSwitchSource(c *gin.Context) {
	params, err := parseSwitchParams(c)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	candidates := searchCandidates(params)
	if len(candidates) == 0 {
		c.JSON(404, gin.H{"error": "no match"})
		return
	}

	sortCandidates(candidates)
	selected, score := selectPlayableCandidate(candidates)
	if selected == nil {
		c.JSON(404, gin.H{"error": "no playable match"})
		return
	}

	c.JSON(200, gin.H{
		"id":       selected.ID,
		"name":     selected.Name,
		"artist":   selected.Artist,
		"album":    selected.Album,
		"duration": selected.Duration,
		"source":   selected.Source,
		"cover":    selected.Cover,
		"score":    score,
		"link":     selected.Link,
	})
}

type switchParams struct {
	name         string
	artist       string
	keyword      string
	current      string
	origDuration int
	sources      []string
}

func parseSwitchParams(c *gin.Context) (*switchParams, error) {
	name := strings.TrimSpace(c.Query("name"))
	if name == "" {
		return nil, fmt.Errorf("missing name")
	}

	artist := strings.TrimSpace(c.Query("artist"))
	current := strings.TrimSpace(c.Query("source"))
	target := strings.TrimSpace(c.Query("target"))
	origDuration, _ := strconv.Atoi(strings.TrimSpace(c.Query("duration")))

	keyword := name
	if artist != "" {
		keyword = name + " " + artist
	}

	var sources []string
	if target != "" {
		sources = []string{target}
	} else {
		sources = core.GetAllSourceNames()
	}

	return &switchParams{
		name:         name,
		artist:       artist,
		keyword:      keyword,
		current:      current,
		origDuration: origDuration,
		sources:      sources,
	}, nil
}

func searchCandidates(params *switchParams) []songCandidate {
	var (
		candidates []songCandidate
		wg         sync.WaitGroup
		mu         sync.Mutex
	)
	for _, src := range params.sources {
		if src == "" || src == params.current || skipSources[src] {
			continue
		}
		fn := core.GetSearchFunc(src)
		if fn == nil {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			results := scoreSourceResults(fn, params, src)
			if len(results) > 0 {
				mu.Lock()
				candidates = append(candidates, results...)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return candidates
}

const maxCandidatesPerSource = 8

func scoreSourceResults(searchFn func(string) ([]model.Song, error), params *switchParams, source string) []songCandidate {
	res, err := searchFn(params.keyword)
	if (err != nil || len(res) == 0) && params.artist != "" {
		res, _ = searchFn(params.name)
	}
	if len(res) == 0 {
		return nil
	}

	limit := min(len(res), maxCandidatesPerSource)
	var candidates []songCandidate

	for i := range limit {
		cand := res[i]
		cand.Source = source
		score := core.CalcSongSimilarity(params.name, params.artist, cand.Name, cand.Artist)
		if score <= 0 {
			continue
		}

		durDiff := 0
		if params.origDuration > 0 && cand.Duration > 0 {
			durDiff = core.IntAbs(params.origDuration - cand.Duration)
			if !core.IsDurationClose(params.origDuration, cand.Duration) {
				continue
			}
		}
		candidates = append(candidates, songCandidate{song: cand, score: score, durDiff: durDiff})
	}
	return candidates
}

func sortCandidates(candidates []songCandidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return candidates[i].durDiff < candidates[j].durDiff
		}
		return candidates[i].score > candidates[j].score
	})
}

func selectPlayableCandidate(candidates []songCandidate) (*model.Song, float64) {
	for _, cand := range candidates {
		if core.ValidatePlayable(&cand.song) {
			song := cand.song
			return &song, cand.score
		}
	}
	return nil, 0
}

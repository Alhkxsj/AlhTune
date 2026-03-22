package core

import (
	"strings"
)

// Audio file extensions
const (
	AudioExtMP3  = "mp3"
	AudioExtFLAC = "flac"
	AudioExtM4A  = "m4a"
	AudioExtOGG  = "ogg"
	AudioExtWAV  = "wav"
	AudioExtWMA  = "wma"
)

// Audio MIME types
const (
	MimeMP3      = "audio/mpeg"
	MimeFLAC     = "audio/flac"
	MimeM4A      = "audio/mp4"
	MimeOGG      = "audio/ogg"
	MimeWAV      = "audio/wav"
	MimeWMA      = "audio/x-ms-wma"
	MimeOctet    = "application/octet-stream"
)

// audioExtMimeMap maps audio extensions to MIME types
var audioExtMimeMap = map[string]string{
	AudioExtMP3:  MimeMP3,
	AudioExtFLAC: MimeFLAC,
	AudioExtM4A:  MimeM4A,
	AudioExtOGG:  MimeOGG,
	AudioExtWAV:  MimeWAV,
	AudioExtWMA:  MimeWMA,
}

// mimeAudioExtMap maps MIME types to audio extensions
var mimeAudioExtMap = map[string]string{
	MimeMP3:      AudioExtMP3,
	"audio/mp3":  AudioExtMP3,
	MimeFLAC:     AudioExtFLAC,
	MimeM4A:      AudioExtM4A,
	"audio/aac":  AudioExtM4A,
	MimeOGG:      AudioExtOGG,
	MimeWAV:      AudioExtWAV,
	MimeWMA:      AudioExtWMA,
	MimeOctet:    AudioExtMP3,
}

// AudioMimeByExt returns the MIME type for an audio extension
func AudioMimeByExt(ext string) string {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	if mime, ok := audioExtMimeMap[ext]; ok {
		return mime
	}
	return MimeOctet
}

// DetectAudioExtByContentType detects audio extension from Content-Type header
func DetectAudioExtByContentType(contentType string) string {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if ext, ok := mimeAudioExtMap[contentType]; ok {
		return ext
	}
	return ""
}

// DetectAudioExt detects audio extension from raw audio data
func DetectAudioExt(audioData []byte) string {
	if len(audioData) < 16 {
		return AudioExtMP3
	}

	// Check for common audio signatures in order of specificity
	detectors := []struct {
		detect func([]byte) bool
		ext    string
	}{
		{isFLAC, AudioExtFLAC},
		{isOGG, AudioExtOGG},
		{isWMA, AudioExtWMA},
		{isM4A, AudioExtM4A},
		{isWAV, AudioExtWAV},
		{isMP3, AudioExtMP3},
	}

	for _, detector := range detectors {
		if detector.detect(audioData) {
			return detector.ext
		}
	}

	return AudioExtMP3
}

func isFLAC(data []byte) bool {
	return len(data) >= 4 &&
		data[0] == 'f' && data[1] == 'L' && data[2] == 'a' && data[3] == 'C'
}

func isOGG(data []byte) bool {
	return len(data) >= 4 &&
		data[0] == 'O' && data[1] == 'g' && data[2] == 'g' && data[3] == 'S'
}

func isWMA(data []byte) bool {
	return len(data) >= 4 &&
		data[0] == 0x30 && data[1] == 0x26 && data[2] == 0xB2 && data[3] == 0x75
}

func isM4A(data []byte) bool {
	return len(data) >= 8 &&
		data[4] == 'f' && data[5] == 't' && data[6] == 'y' && data[7] == 'p'
}

func isWAV(data []byte) bool {
	return len(data) >= 12 &&
		data[0] == 'R' && data[1] == 'I' && data[2] == 'F' && data[3] == 'F' &&
		data[8] == 'W' && data[9] == 'A' && data[10] == 'V' && data[11] == 'E'
}

func isMP3(data []byte) bool {
	if len(data) < 3 {
		return false
	}
	// ID3 tag
	if data[0] == 'I' && data[1] == 'D' && data[2] == '3' {
		return true
	}
	// Frame sync
	if data[0] == 0xFF && (data[1]&0xE0) == 0xE0 {
		return true
	}
	return false
}

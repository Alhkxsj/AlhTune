package core

import (
	"bytes"
	"strings"
)

// audioMagicNumbers defines file signatures for audio formats
var audioMagicNumbers = []struct {
	signature []byte
	ext       string
}{
	{[]byte{0x30, 0x26, 0xB2, 0x75, 0x8E, 0x66, 0xCF, 0x11, 0xA6, 0xD9, 0x00, 0xAA, 0x00, 0x62, 0xCE, 0x6C}, "wma"},
	{[]byte{'f', 'L', 'a', 'C'}, "flac"},
	{[]byte{'I', 'D', '3'}, "mp3"},
	{[]byte{'O', 'g', 'g', 'S'}, "ogg"},
	{[]byte{'f', 't', 'y', 'p'}, "m4a"},
}

// DetectAudioExt detects audio format from file header
func DetectAudioExt(data []byte) string {
	for _, magic := range audioMagicNumbers {
		if len(data) >= len(magic.signature) && bytes.Equal(data[:len(magic.signature)], magic.signature) {
			return magic.ext
		}
	}

	if len(data) >= 2 && data[0] == 0xFF && (data[1]&0xE0) == 0xE0 {
		return "mp3"
	}

	return "mp3"
}

// contentTypeToExt maps MIME types to file extensions
var contentTypeToExt = map[string]string{
	"audio/flac":             "flac",
	"audio/x-flac":           "flac",
	"audio/x-ms-wma":         "wma",
	"audio/wma":              "wma",
	"video/x-ms-asf":         "wma",
	"application/vnd.ms-asf": "wma",
	"audio/mpeg":             "mp3",
	"audio/mp3":              "mp3",
	"audio/x-mp3":            "mp3",
	"audio/ogg":              "ogg",
	"application/ogg":        "ogg",
	"audio/mp4":              "m4a",
	"audio/x-m4a":            "m4a",
	"audio/aac":              "m4a",
	"audio/aacp":             "m4a",
}

// DetectAudioExtByContentType detects audio format from Content-Type header
func DetectAudioExtByContentType(contentType string) string {
	contentType = strings.TrimSpace(strings.ToLower(contentType))
	if idx := strings.Index(contentType, ";"); idx >= 0 {
		contentType = strings.TrimSpace(contentType[:idx])
	}

	if ext, ok := contentTypeToExt[contentType]; ok {
		return ext
	}
	return ""
}

// extToMime maps file extensions to MIME types
var extToMime = map[string]string{
	"wma":  "audio/x-ms-wma",
	"flac": "audio/flac",
	"ogg":  "audio/ogg",
	"m4a":  "audio/mp4",
}

// AudioMimeByExt gets MIME type from file extension
func AudioMimeByExt(ext string) string {
	ext = strings.ToLower(strings.TrimSpace(ext))
	if mime, ok := extToMime[ext]; ok {
		return mime
	}
	return "audio/mpeg"
}

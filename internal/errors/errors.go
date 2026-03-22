package errors

import (
	"fmt"
)

// Error types for music download operations
var (
	// ErrFFmpegNotFound is returned when ffmpeg is not available
	ErrFFmpegNotFound = New("ffmpeg not found: metadata embedding disabled")

	// ErrInvalidSong is returned when song data is invalid
	ErrInvalidSong = New("invalid song: missing required fields")

	// ErrUnsupportedSource is returned when a music source is not supported
	ErrUnsupportedSource = New("unsupported music source")

	// ErrEmptyDownloadURL is returned when download URL is empty
	ErrEmptyDownloadURL = New("download URL is empty")

	// ErrFetchFailed is returned when fetching data fails
	ErrFetchFailed = New("fetch operation failed")

	// ErrDownloadFailed is returned when downloading fails
	ErrDownloadFailed = New("download operation failed")

	// ErrMetadataEmbedFailed is returned when metadata embedding fails
	ErrMetadataEmbedFailed = New("metadata embedding failed")
)

// Error represents a custom error with context
type Error struct {
	message string
	cause   error
}

// New creates a new Error with the given message
func New(message string) *Error {
	return &Error{message: message}
}

// Wrap creates a new Error wrapping another error
func Wrap(cause error, message string) *Error {
	return &Error{
		message: message,
		cause:   cause,
	}
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.message, e.cause)
	}
	return e.message
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.cause
}

// Is checks if an error matches a specific error type
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.message == t.message && e.cause == t.cause
}

// Helper functions for creating contextual errors

// InvalidSong creates an error for invalid song data
func InvalidSong(reason string) error {
	return Wrap(ErrInvalidSong, reason)
}

// UnsupportedSource creates an error for unsupported music source
func UnsupportedSource(source string) error {
	return Wrap(ErrUnsupportedSource, fmt.Sprintf("source: %s", source))
}

// EmptyDownloadURL creates an error for empty download URL
func EmptyDownloadURL(source string) error {
	return Wrap(ErrEmptyDownloadURL, fmt.Sprintf("source: %s", source))
}

// FetchFailed creates an error for fetch operation failure
func FetchFailed(source, reason string) error {
	return Wrap(ErrFetchFailed, fmt.Sprintf("source %s: %s", source, reason))
}

// DownloadFailed creates an error for download operation failure
func DownloadFailed(source, reason string) error {
	return Wrap(ErrDownloadFailed, fmt.Sprintf("source %s: %s", source, reason))
}

// MetadataEmbedFailed creates an error for metadata embedding failure
func MetadataEmbedFailed(reason string) error {
	return Wrap(ErrMetadataEmbedFailed, reason)
}
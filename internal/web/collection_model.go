package web

import (
	"time"
)

// Collection represents a user-created playlist collection
type Collection struct {
	ID          uint        `json:"id"          gorm:"primaryKey"`
	CreatedAt   time.Time   `json:"created_at"`
	Name        string      `json:"name"        gorm:"not null"`
	Description string      `json:"description"`
	Cover       string      `json:"cover"`
	SavedSongs  []SavedSong `json:"-"           gorm:"constraint:OnDelete:CASCADE;"`
}

// SavedSong represents a song saved in a collection
type SavedSong struct {
	ID           uint      `json:"db_id"         gorm:"primaryKey"`
	CollectionID uint      `json:"collection_id" gorm:"uniqueIndex:idx_col_song_src"`
	SongID       string    `json:"song_id"       gorm:"uniqueIndex:idx_col_song_src;not null"`
	Source       string    `json:"source"        gorm:"uniqueIndex:idx_col_song_src;not null"`
	AddedAt      time.Time `json:"added_at"`
	Name         string    `json:"name"`
	Artist       string    `json:"artist"`
	Cover        string    `json:"cover"`
	Duration     int       `json:"duration"`
	Extra        string    `json:"extra"`
}

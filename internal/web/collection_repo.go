package web

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CollectionRepository handles database operations for collections
type CollectionRepository struct {
	db *gorm.DB
}

// NewCollectionRepository creates a new repository instance
func NewCollectionRepository(database *gorm.DB) *CollectionRepository {
	return &CollectionRepository{db: database}
}

// List returns all collections ordered by ID descending
func (r *CollectionRepository) List() ([]Collection, error) {
	var collections []Collection
	err := r.db.Order("id DESC").Find(&collections).Error
	return collections, err
}

// Get retrieves a collection by ID
func (r *CollectionRepository) Get(id uint) (*Collection, error) {
	var col Collection
	err := r.db.First(&col, id).Error
	if err != nil {
		return nil, err
	}
	return &col, nil
}

// Create creates a new collection
func (r *CollectionRepository) Create(name, description, cover string) (*Collection, error) {
	col := &Collection{
		Name:        name,
		Description: description,
		Cover:       cover,
	}
	err := r.db.Create(col).Error
	return col, err
}

// Update updates an existing collection
func (r *CollectionRepository) Update(id uint, name, description, cover string) error {
	return r.db.Model(&Collection{}).Where("id = ?", id).Updates(Collection{
		Name:        name,
		Description: description,
		Cover:       cover,
	}).Error
}

// Delete removes a collection
func (r *CollectionRepository) Delete(id uint) error {
	return r.db.Delete(&Collection{}, id).Error
}

// GetSongs retrieves all songs in a collection
func (r *CollectionRepository) GetSongs(collectionID uint) ([]SavedSong, error) {
	var songs []SavedSong
	err := r.db.Where("collection_id = ?", collectionID).Order("id DESC").Find(&songs).Error
	return songs, err
}

// AddSong adds a song to a collection (ignores duplicates)
func (r *CollectionRepository) AddSong(collectionID uint, song SavedSong) error {
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&song).Error
}

// RemoveSong removes a song from a collection
func (r *CollectionRepository) RemoveSong(collectionID uint, songID, source string) error {
	return r.db.Where("collection_id = ? AND song_id = ? AND source = ?", collectionID, songID, source).
		Delete(&SavedSong{}).Error
}

// CountSongs returns the number of songs in a collection
func (r *CollectionRepository) CountSongs(collectionID uint) (int64, error) {
	var count int64
	err := r.db.Model(&SavedSong{}).Where("collection_id = ?", collectionID).Count(&count).Error
	return count, err
}

// db is the global database instance
var db *gorm.DB

// InitDB initializes the database connection and runs migrations
func InitDB() error {
	if err := os.MkdirAll("data", 0755); err != nil {
		return fmt.Errorf("create data directory failed: %w", err)
	}

	var err error
	db, err = gorm.Open(sqlite.Open("data/favorites.db?_pragma=foreign_keys(1)"), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("connect SQLite failed: %w", err)
	}

	if err := db.AutoMigrate(&Collection{}, &SavedSong{}); err != nil {
		return fmt.Errorf("migrate database failed: %w", err)
	}

	return nil
}

// CloseDB closes the database connection
func CloseDB() {
	if db != nil {
		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.Close()
		}
	}
}

// parseCollectionID parses a collection ID from string
func parseCollectionID(idStr string) (uint, error) {
	var id uint
	_, err := fmt.Sscanf(idStr, "%d", &id)
	return id, err
}

// encodeExtra encodes extra data to JSON string
func encodeExtra(extra interface{}) string {
	if extra == nil {
		return ""
	}
	b, _ := json.Marshal(extra)
	return string(b)
}

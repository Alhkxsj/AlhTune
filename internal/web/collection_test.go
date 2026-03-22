package web

import (
	"os"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *CollectionRepository {
	t.Helper()
	tmpFile := "data/test_favorites.db"
	os.MkdirAll("data", 0755)
	t.Cleanup(func() {
		os.Remove(tmpFile)
	})

	var err error
	db, err = initTestDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to init test DB: %v", err)
	}
	t.Cleanup(CloseDB)

	return NewCollectionRepository(db)
}

func TestCollectionRepository_Create(t *testing.T) {
	repo := setupTestDB(t)

	col, err := repo.Create("Test Playlist", "Test Description", "http://example.com/cover.jpg")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if col.Name != "Test Playlist" {
		t.Errorf("Name = %q, want %q", col.Name, "Test Playlist")
	}
	if col.ID == 0 {
		t.Error("ID should not be 0")
	}
}

func TestCollectionRepository_List(t *testing.T) {
	repo := setupTestDB(t)

	_, err := repo.Create("Test Playlist", "Test Description", "http://example.com/cover.jpg")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	collections, err := repo.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(collections) != 1 {
		t.Errorf("List returned %d items, want 1", len(collections))
	}
}

func TestCollectionRepository_Get(t *testing.T) {
	repo := setupTestDB(t)

	col, err := repo.Create("Get Test", "", "")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := repo.Get(col.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name != "Get Test" {
		t.Errorf("Name = %q, want %q", got.Name, "Get Test")
	}
}

func TestCollectionRepository_Update(t *testing.T) {
	repo := setupTestDB(t)

	col, err := repo.Create("Old Name", "Old Desc", "")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = repo.Update(col.ID, "New Name", "New Desc", "http://new.com")
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, err := repo.Get(col.ID)
	if err != nil {
		t.Fatalf("Get after update failed: %v", err)
	}
	if got.Name != "New Name" {
		t.Errorf("Name = %q, want %q", got.Name, "New Name")
	}
}

func TestCollectionRepository_Delete(t *testing.T) {
	repo := setupTestDB(t)

	col, err := repo.Create("Delete Me", "", "")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = repo.Delete(col.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = repo.Get(col.ID)
	if err == nil {
		t.Error("Get after delete should return error")
	}
}

func TestCollectionRepository_AddSong(t *testing.T) {
	repo := setupTestDB(t)

	col, err := repo.Create("Song Test", "", "")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	song := SavedSong{
		CollectionID: col.ID,
		SongID:       "song123",
		Source:       "netease",
		Name:         "Test Song",
		Artist:       "Test Artist",
		Duration:     180,
	}

	err = repo.AddSong(col.ID, song)
	if err != nil {
		t.Fatalf("AddSong failed: %v", err)
	}
}

func TestCollectionRepository_GetSongs(t *testing.T) {
	repo := setupTestDB(t)

	col, err := repo.Create("GetSongs Test", "", "")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	song := SavedSong{
		CollectionID: col.ID,
		SongID:       "song456",
		Source:       "qq",
		Name:         "Another Song",
		Artist:       "Another Artist",
		Duration:     200,
	}
	repo.AddSong(col.ID, song)

	songs, err := repo.GetSongs(col.ID)
	if err != nil {
		t.Fatalf("GetSongs failed: %v", err)
	}
	if len(songs) != 1 {
		t.Errorf("GetSongs returned %d items, want 1", len(songs))
	}
}

func TestCollectionRepository_RemoveSong(t *testing.T) {
	repo := setupTestDB(t)

	col, err := repo.Create("RemoveSong Test", "", "")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	song := SavedSong{
		CollectionID: col.ID,
		SongID:       "song789",
		Source:       "kugou",
		Name:         "Remove Me",
		Artist:       "Artist",
	}
	repo.AddSong(col.ID, song)

	err = repo.RemoveSong(col.ID, "song789", "kugou")
	if err != nil {
		t.Fatalf("RemoveSong failed: %v", err)
	}

	songs, _ := repo.GetSongs(col.ID)
	if len(songs) != 0 {
		t.Errorf("GetSongs after remove returned %d items, want 0", len(songs))
	}
}

func TestCollectionRepository_CountSongs(t *testing.T) {
	repo := setupTestDB(t)

	col, err := repo.Create("CountSongs Test", "", "")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	for i := 0; i < 5; i++ {
		song := SavedSong{
			CollectionID: col.ID,
			SongID:       string(rune('a' + i)),
			Source:       "netease",
			Name:         "Song",
		}
		repo.AddSong(col.ID, song)
	}

	count, err := repo.CountSongs(col.ID)
	if err != nil {
		t.Fatalf("CountSongs failed: %v", err)
	}
	if count != 5 {
		t.Errorf("CountSongs = %d, want 5", count)
	}
}

func TestSessionManager(t *testing.T) {
	sm := NewSessionManager()

	t.Run("Create and Get", func(t *testing.T) {
		session := sm.Create("test1", "/tmp/test1", "/tmp/test1/audio.mp3")
		if session == nil {
			t.Fatal("Create returned nil")
		}

		got, ok := sm.Get("test1")
		if !ok {
			t.Error("Get returned false")
		}
		if got.ID != "test1" {
			t.Errorf("ID = %q, want %q", got.ID, "test1")
		}
	})

	t.Run("Get non-existent", func(t *testing.T) {
		_, ok := sm.Get("nonexistent")
		if ok {
			t.Error("Get should return false for non-existent session")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		sm.Create("test2", "/tmp/test2", "/tmp/test2/audio.mp3")
		sm.Delete("test2")

		_, ok := sm.Get("test2")
		if ok {
			t.Error("Get after Delete should return false")
		}
	})
}

func TestCleanupOldFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create old file
	oldFile := tmpDir + "/old.txt"
	if err := os.WriteFile(oldFile, []byte("old"), 0644); err != nil {
		t.Fatalf("Create old file failed: %v", err)
	}

	// Modify time to make it old
	oldTime := time.Now().Add(-15 * time.Minute)
	os.Chtimes(oldFile, oldTime, oldTime)

	// Create new file
	newFile := tmpDir + "/new.txt"
	if err := os.WriteFile(newFile, []byte("new"), 0644); err != nil {
		t.Fatalf("Create new file failed: %v", err)
	}

	CleanupOldFiles(tmpDir, 10*time.Minute)

	// Old file should be deleted
	if _, err := os.Stat(oldFile); err == nil {
		t.Error("Old file should be deleted")
	}

	// New file should still exist
	if _, err := os.Stat(newFile); err != nil {
		t.Error("New file should still exist")
	}
}

func initTestDB(dbPath string) (*gorm.DB, error) {
	testDB, err := gorm.Open(sqlite.Open(dbPath+"?_pragma=foreign_keys(1)"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := testDB.AutoMigrate(&Collection{}, &SavedSong{}); err != nil {
		return nil, err
	}

	return testDB, nil
}

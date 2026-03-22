package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewCookieManager(t *testing.T) {
	cm := NewCookieManager()
	if cm == nil {
		t.Fatal("NewCookieManager() returned nil")
	}
	if cm.cookies == nil {
		t.Error("NewCookieManager() initialized with nil cookies map")
	}
}

func TestCookieManager_Get(t *testing.T) {
	cm := NewCookieManager()

	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{"empty", "", ""},
		{"non-existent", "unknown", ""},
		{"case sensitive", "NetEase", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cm.Get(tt.source)
			if result != tt.expected {
				t.Errorf("Get(%q) = %q, want %q", tt.source, result, tt.expected)
			}
		})
	}
}

func TestCookieManager_Set(t *testing.T) {
	cm := NewCookieManager()

	// Test setting a cookie
	cm.Set("netease", "cookie1")
	if got := cm.Get("netease"); got != "cookie1" {
		t.Errorf("After Set(\"netease\", \"cookie1\"), Get() = %q, want %q", got, "cookie1")
	}

	// Test updating a cookie
	cm.Set("netease", "cookie2")
	if got := cm.Get("netease"); got != "cookie2" {
		t.Errorf("After Set(\"netease\", \"cookie2\"), Get() = %q, want %q", got, "cookie2")
	}

	// Test deleting a cookie by setting empty string
	cm.Set("netease", "")
	if got := cm.Get("netease"); got != "" {
		t.Errorf("After Set(\"netease\", \"\"), Get() = %q, want empty", got)
	}
}

func TestCookieManager_SetAll(t *testing.T) {
	cm := NewCookieManager()

	cookies := map[string]string{
		"netease": "cookie1",
		"qq":      "cookie2",
		"kugou":   "cookie3",
	}

	cm.SetAll(cookies)

	for source, expected := range cookies {
		if got := cm.Get(source); got != expected {
			t.Errorf("After SetAll(), Get(%q) = %q, want %q", source, got, expected)
		}
	}

	// Test updating and deleting
	updateCookies := map[string]string{
		"netease": "updated",
		"qq":      "", // Delete
		"kugou":   "cookie3", // Keep
		"kuwo":    "cookie4", // Add
	}

	cm.SetAll(updateCookies)

	if got := cm.Get("netease"); got != "updated" {
		t.Errorf("After SetAll() with update, Get(\"netease\") = %q, want %q", got, "updated")
	}
	if got := cm.Get("qq"); got != "" {
		t.Errorf("After SetAll() with empty value, Get(\"qq\") = %q, want empty", got)
	}
	if got := cm.Get("kugou"); got != "cookie3" {
		t.Errorf("After SetAll() with same value, Get(\"kugou\") = %q, want %q", got, "cookie3")
	}
	if got := cm.Get("kuwo"); got != "cookie4" {
		t.Errorf("After SetAll() with new value, Get(\"kuwo\") = %q, want %q", got, "cookie4")
	}
}

func TestCookieManager_GetAll(t *testing.T) {
	cm := NewCookieManager()

	// Empty manager
	all := cm.GetAll()
	if len(all) != 0 {
		t.Errorf("GetAll() on empty manager returned %d items, want 0", len(all))
	}

	// With cookies
	cm.Set("netease", "cookie1")
	cm.Set("qq", "cookie2")

	all = cm.GetAll()
	if len(all) != 2 {
		t.Errorf("GetAll() returned %d items, want 2", len(all))
	}
	if all["netease"] != "cookie1" {
		t.Errorf("GetAll()[\"netease\"] = %q, want %q", all["netease"], "cookie1")
	}
	if all["qq"] != "cookie2" {
		t.Errorf("GetAll()[\"qq\"] = %q, want %q", all["qq"], "cookie2")
	}

	// Verify it's a copy
	all["netease"] = "modified"
	if got := cm.Get("netease"); got != "cookie1" {
		t.Error("GetAll() returned a reference, not a copy")
	}
}

func TestCookieManager_Has(t *testing.T) {
	cm := NewCookieManager()

	if cm.Has("netease") {
		t.Error("Has(\"netease\") on empty manager returned true")
	}

	cm.Set("netease", "cookie1")

	if !cm.Has("netease") {
		t.Error("Has(\"netease\") after Set() returned false")
	}
	if cm.Has("qq") {
		t.Error("Has(\"qq\") on unset source returned true")
	}
}

func TestCookieManager_Delete(t *testing.T) {
	cm := NewCookieManager()

	cm.Set("netease", "cookie1")
	cm.Set("qq", "cookie2")

	cm.Delete("netease")

	if cm.Has("netease") {
		t.Error("After Delete(), Has(\"netease\") returned true")
	}
	if !cm.Has("qq") {
		t.Error("After Delete(\"netease\"), Has(\"qq\") returned false")
	}

	// Delete non-existent should not panic
	cm.Delete("nonexistent")
}

func TestCookieManager_Clear(t *testing.T) {
	cm := NewCookieManager()

	cm.Set("netease", "cookie1")
	cm.Set("qq", "cookie2")

	cm.Clear()

	if cm.Count() != 0 {
		t.Errorf("After Clear(), Count() = %d, want 0", cm.Count())
	}
	if cm.Has("netease") {
		t.Error("After Clear(), Has(\"netease\") returned true")
	}
}

func TestCookieManager_Count(t *testing.T) {
	cm := NewCookieManager()

	if cm.Count() != 0 {
		t.Errorf("Count() on empty manager = %d, want 0", cm.Count())
	}

	cm.Set("netease", "cookie1")
	if cm.Count() != 1 {
		t.Errorf("Count() after 1 Set() = %d, want 1", cm.Count())
	}

	cm.Set("qq", "cookie2")
	cm.Set("kugou", "cookie3")
	if cm.Count() != 3 {
		t.Errorf("Count() after 3 Set() = %d, want 3", cm.Count())
	}

	cm.Set("netease", "") // Delete by setting empty
	if cm.Count() != 2 {
		t.Errorf("Count() after delete = %d, want 2", cm.Count())
	}
}

func TestCookieManager_GetSources(t *testing.T) {
	cm := NewCookieManager()

	sources := cm.GetSources()
	if len(sources) != 0 {
		t.Errorf("GetSources() on empty manager returned %d items", len(sources))
	}

	cm.Set("netease", "cookie1")
	cm.Set("qq", "cookie2")
	cm.Set("kugou", "cookie3")

	sources = cm.GetSources()
	if len(sources) != 3 {
		t.Errorf("GetSources() returned %d items, want 3", len(sources))
	}

	// Check all sources are present
	sourceMap := make(map[string]bool)
	for _, s := range sources {
		sourceMap[s] = true
	}

	expectedSources := []string{"netease", "qq", "kugou"}
	for _, expected := range expectedSources {
		if !sourceMap[expected] {
			t.Errorf("GetSources() missing expected source %q", expected)
		}
	}
}

func TestCookieManager_SaveLoad(t *testing.T) {
	// Create temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_cookies.json")

	originalCM := NewCookieManager()
	originalCM.Set("netease", "cookie1")
	originalCM.Set("qq", "cookie2")
	originalCM.Set("kugou", "cookie3")

	// Test Save
	err := originalCM.Save(tmpFile)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("Save() did not create file")
	}

	// Test Load
	loadedCM := NewCookieManager()
	err = loadedCM.Load(tmpFile)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify cookies are loaded
	if loadedCM.Count() != originalCM.Count() {
		t.Errorf("After Load(), Count() = %d, want %d", loadedCM.Count(), originalCM.Count())
	}

	if got := loadedCM.Get("netease"); got != "cookie1" {
		t.Errorf("After Load(), Get(\"netease\") = %q, want %q", got, "cookie1")
	}
	if got := loadedCM.Get("qq"); got != "cookie2" {
		t.Errorf("After Load(), Get(\"qq\") = %q, want %q", got, "cookie2")
	}
	if got := loadedCM.Get("kugou"); got != "cookie3" {
		t.Errorf("After Load(), Get(\"kugou\") = %q, want %q", got, "cookie3")
	}
}

func TestCookieManager_Load_NonExistent(t *testing.T) {
	cm := NewCookieManager()

	// Load from non-existent file should not error
	err := cm.Load("/nonexistent/path/cookies.json")
	if err != nil {
		t.Errorf("Load() from non-existent file returned error: %v", err)
	}
}

func TestCookieManager_Concurrency(t *testing.T) {
	cm := NewCookieManager()

	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(n int) {
			cm.Set("source"+string(rune('0'+n%10)), "cookie"+string(rune('0'+n%10)))
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			cm.GetAll()
			cm.Count()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Verify final state
	if cm.Count() < 10 {
		t.Errorf("After concurrent operations, Count() = %d, want at least 10", cm.Count())
	}
}
package core

// GetSearchFunc retrieves the search function for a source
func GetSearchFunc(source string) SearchFunc {
	if p := getSourceProvider(source); p != nil {
		return p.Search
	}
	return nil
}

// GetPlaylistSearchFunc retrieves the playlist search function for a source
func GetPlaylistSearchFunc(source string) SearchPlaylistFunc {
	if p := getSourceProvider(source); p != nil {
		return p.SearchPlaylist
	}
	return nil
}

// GetPlaylistDetailFunc retrieves the playlist detail function for a source
func GetPlaylistDetailFunc(source string) GetPlaylistSongsFunc {
	if p := getSourceProvider(source); p != nil {
		return p.GetPlaylistSongs
	}
	return nil
}

// GetRecommendFunc retrieves the recommendation function for a source
func GetRecommendFunc(source string) GetRecommendFuncType {
	if p := getSourceProvider(source); p != nil {
		return p.GetRecommend
	}
	return nil
}

// GetDownloadFunc retrieves the download URL function for a source
func GetDownloadFunc(source string) GetDownloadURLFunc {
	if p := getSourceProvider(source); p != nil {
		return p.GetDownload
	}
	return nil
}

// GetLyricFuncFromSource retrieves the lyric function for a source
func GetLyricFuncFromSource(source string) GetLyricFunc {
	if p := getSourceProvider(source); p != nil {
		return p.GetLyric
	}
	return nil
}

// GetParseFunc retrieves the parse function for a source
func GetParseFunc(source string) ParseSongFunc {
	if p := getSourceProvider(source); p != nil {
		return p.Parse
	}
	return nil
}

// GetParsePlaylistFunc retrieves the playlist parse function for a source
func GetParsePlaylistFunc(source string) ParsePlaylistFunc {
	if p := getSourceProvider(source); p != nil {
		return p.ParsePlaylist
	}
	return nil
}

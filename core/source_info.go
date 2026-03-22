package core

// GetAllSourceNames returns all available music source names
func GetAllSourceNames() []string {
	return []string{
		"netease", "qq", "kugou", "kuwo", "migu",
		"fivesing", "jamendo", "joox", "qianqian", "soda", "bilibili",
	}
}

// GetPlaylistSourceNames returns sources that support playlists
func GetPlaylistSourceNames() []string {
	return []string{"netease", "qq", "kugou", "kuwo", "bilibili", "soda", "fivesing"}
}

// GetDefaultSourceNames returns default sources (excludes some providers)
func GetDefaultSourceNames() []string {
	excluded := map[string]bool{"bilibili": true, "joox": true, "jamendo": true, "fivesing": true}
	allSources := GetAllSourceNames()

	defaultSources := make([]string, 0, len(allSources)-len(excluded))
	for _, source := range allSources {
		if !excluded[source] {
			defaultSources = append(defaultSources, source)
		}
	}
	return defaultSources
}

// sourceDescriptions maps source names to Chinese descriptions
var sourceDescriptions = map[string]string{
	"netease":  "网易云音乐",
	"qq":       "QQ 音乐",
	"kugou":    "酷狗音乐",
	"kuwo":     "酷我音乐",
	"migu":     "咪咕音乐",
	"fivesing": "5sing",
	"jamendo":  "Jamendo (CC)",
	"joox":     "JOOX",
	"qianqian": "千千音乐",
	"soda":     "Soda 音乐",
	"bilibili": "Bilibili",
}

// GetSourceDescription gets the description for a source
func GetSourceDescription(source string) string {
	if desc, ok := sourceDescriptions[source]; ok {
		return desc
	}
	return "未知音乐源"
}

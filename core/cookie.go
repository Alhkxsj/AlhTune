package core

import (
	"github.com/Alhkxsj/AlhTune/internal/utils"
)

const (
	CookieFile   = "data/cookies.json"
	UACommon     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36"
	UAMobile     = "Mozilla/5.0 (iPhone; CPU iPhone OS 9_1 like Mac OS X) AppleWebKit/601.1.46 (KHTML, like Gecko) Version/9.0 Mobile/13B143 Safari/601.1"
	RefBilibili  = "https://www.bilibili.com/"
	RefMigu      = "http://music.migu.cn/"
	AudioExtMP3  = "mp3"
	AudioExtFLAC = "flac"
	AudioExtM4A  = "m4a"
	AudioExtWMA  = "wma"
)

// CM is the global cookie manager instance
var CM = utils.NewCookieManager()

// LoadCookies loads cookies from the default cookie file
func LoadCookies() error {
	return CM.Load(CookieFile)
}

// SaveCookies saves cookies to the default cookie file
func SaveCookies() error {
	return CM.Save(CookieFile)
}

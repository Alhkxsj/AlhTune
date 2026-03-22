package main

import (
	"fmt"
	"os"

	"github.com/Alhkxsj/AlhTune/internal/cli"
	"github.com/spf13/cobra"
)

var (
	showVersion bool
	keyword     string
	sources     []string
	outDir      string
	withCover   bool
	withLyrics  bool
)

var rootCmd = &cobra.Command{
	Use:   "alhtune",
	Short: "AlhTune - 音乐搜索下载工具",
	Long: `AlhTune 音乐搜索下载工具。

支持的音乐源:
  - netease (网易云音乐)
  - qq (QQ 音乐)
  - kugou (酷狗音乐)
  - kuwo (酷我音乐)
  - migu (咪咕音乐)
  - qianqian (千千音乐)
  - soda (汽水音乐)
  - fivesing (5sing)
  - jamendo
  - joox
  - bilibili

功能:
  - TUI 界面
  - Web 界面 (alhtune web)
  - 下载高品质音频
  - 自动下载封面和歌词`,
	Example: `  # 搜索音乐
  alhtune -k "周杰伦"

  # 指定源搜索
  alhtune -k "林俊杰" -s netease,qq

  # 指定下载目录
  alhtune -k "陈奕迅" -o "MyMusic"

  # 启动 Web 界面
  alhtune web

  # 直接进入 TUI 界面
  alhtune`,
	Run: func(cmd *cobra.Command, args []string) {
		if showVersion {
			fmt.Println("alhtune version v1.3.7")
			fmt.Println("Author: Alhkxsj")
			fmt.Println("Github: https://github.com/Alhkxsj")
			return
		}

		if outDir == "" {
			homeDir := os.Getenv("HOME")
			if homeDir == "" {
				homeDir = "/data/data/com.termux/files/home"
			}
			outDir = homeDir + "/Music/alhtune"
		}

		if _, err := os.Stat(outDir); os.IsNotExist(err) {
			_ = os.MkdirAll(outDir, 0755)
		}

		cli.StartUI(keyword, sources, outDir, withCover, withLyrics)
	},
}

func init() {
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "显示版本信息")
	rootCmd.Flags().StringVarP(&keyword, "keyword", "k", "", "搜索关键字")
	rootCmd.Flags().StringSliceVarP(&sources, "sources", "s", []string{}, "指定搜索源，用逗号分隔 (e.g. netease,qq,kugou)")
	rootCmd.Flags().StringVarP(&outDir, "outdir", "o", "", "指定下载目录")
	rootCmd.Flags().BoolVar(&withCover, "cover", true, "下载封面")
	rootCmd.Flags().BoolVarP(&withLyrics, "lyrics", "l", true, "下载歌词")
}

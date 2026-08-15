//go:build windows

package tray

import (
	_ "embed"

	"github.com/getlantern/systray"
)

//go:embed icon.ico
var icon []byte

func Start(openApp func(), openBrowser func(), shutdown chan<- struct{}) {
	go systray.Run(func() {
		systray.SetIcon(icon)
		systray.SetTitle("VRC++")
		systray.SetTooltip("VRC++ 正在后台运行")
		open := systray.AddMenuItem("打开 VRC++", "显示 VRC++ 应用窗口")
		browser := systray.AddMenuItem("在浏览器中打开", "使用默认浏览器打开备用界面")
		systray.AddSeparator()
		quit := systray.AddMenuItem("退出 VRC++", "停止本地服务并退出")
		go func() {
			for {
				select {
				case <-open.ClickedCh:
					openApp()
				case <-browser.ClickedCh:
					openBrowser()
				case <-quit.ClickedCh:
					select {
					case shutdown <- struct{}{}:
					default:
					}
					return
				}
			}
		}()
	}, func() {})
}

func Stop() { systray.Quit() }

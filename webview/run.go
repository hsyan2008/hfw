package webview

import (
	"runtime"

	"github.com/Nerdmaster/terminal"
	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw"
	"github.com/hsyan2008/hfw/common"
	"github.com/webview/webview"
)

//在本机启动内部服务
//注意config.toml里Server的地址必须是127.0.0.1:0
func Run(uri, title string, width, height int, resize bool) {
	if runtime.GOOS == "windows" || !terminal.IsTerminal(0) {
		logger.SetConsole(false)
	}

	addr, err := hfw.GetHTTPAddr(hfw.Config.Server)
	if err != nil {
		panic(err)
	}

	logger.Info("Listen:", addr)

	w := webview.New(logger.Level() == logger.DEBUG)
	defer w.Destroy()

	w.SetTitle(common.ToOsCode(title))
	w.SetSize(width, height, webview.HintNone)
	w.Navigate("http://" + addr + uri)
	w.Run()
}

package webview

import (
	"runtime"

	"github.com/Nerdmaster/terminal"
	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/serve"
	"github.com/zserge/webview"
)

//在本机启动内部服务
//注意config.toml里Server的地址必须是127.0.0.1:0
func Run(uri, title string, width, height int, resize bool) {
	if runtime.GOOS == "windows" || !terminal.IsTerminal(0) {
		logger.SetConsole(false)
	}

	addr, err := serve.GetAddr()
	if err != nil {
		panic(err)
	}

	logger.Info("Listen:", addr)

	err = webview.Open(common.ToOsCode(title),
		"http://"+addr+uri, width, height, resize)
	if err != nil {
		panic(err)
	}
}

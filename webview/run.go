package webview

import (
	"net"
	"net/http"
	"runtime"

	"github.com/Nerdmaster/terminal"
	"github.com/google/gops/agent"
	logger "github.com/hsyan2008/go-logger"
	hfw "github.com/hsyan2008/hfw2"
	"github.com/hsyan2008/hfw2/common"
	"github.com/zserge/webview"
)

var randPortListener net.Listener

func run() (err error) {
	//防止被挂起，如webview
	// defer os.Exit(0)
	// defer randPortListener.Close()

	logger.Info("Starting ...")
	defer logger.Info("Shutdowned!")

	logger.Infof("Running, VERSION=%s, ENVIRONMENT=%s, APPNAME=%s, APPPATH=%s", hfw.VERSION, hfw.ENVIRONMENT, hfw.APPNAME, hfw.APPPATH)

	if err = agent.Listen(agent.Options{}); err != nil {
		logger.Fatal(err)
		return
	}

	signalContext := hfw.GetSignalContext()

	//等待工作完成
	defer signalContext.Shutdowned()

	if hfw.Config.HotDeploy.Enable {
		go hfw.HotDeploy(hfw.Config.HotDeploy)
	}

	return http.Serve(randPortListener, nil)
}

func runRandPort() string {
	randPortListener, _ = net.Listen("tcp", "127.0.0.1:0")
	go func() {
		err := run()
		if err != nil {
			logger.Warn(err)
		}
	}()

	return randPortListener.Addr().String()
}

func Run(uri, title string, width, height int, resize bool) {

	if runtime.GOOS == "windows" || !terminal.IsTerminal(0) {
		logger.SetConsole(false)
	}

	addr := runRandPort()

	logger.Info("Listen:", addr)

	err := webview.Open(common.ToOsCode(title),
		"http://"+addr+uri, width, height, resize)
	if err != nil {
		panic(err)
	}
}

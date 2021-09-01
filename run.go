package hfw

import (
	"net/http"

	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/signal"
)

//Run start
func Run() (err error) {

	signalContext := signal.GetSignalContext()

	signalContext.Mix("Starting ...")
	defer signalContext.Mix("Shutdowned!")

	signalContext.Mixf("Running, VERSION=%s, ENVIRONMENT=%s, APPNAME=%s, APPPATH=%s",
		common.GetVersion(), common.GetEnv(), common.GetAppName(), common.GetAppPath())

	//监听信号
	go signalContext.Listen()

	//等待工作完成
	defer signalContext.Shutdowned()

	if len(Config.Server.Address) == 0 {
		signalContext.Fatal("server address is nil")
		<-signal.GetSignalContext().Ctx.Done()
		return
	}

	//启动http
	signalContext.IsHTTP = true

	err = StartHTTP(Config.Server)

	//如果未启动服务，就触发退出
	if err != nil && err != http.ErrServerClosed {
		signalContext.Fatal(err)
	}

	return
}

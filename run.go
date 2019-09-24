package hfw

import (
	"net/http"

	"github.com/google/gops/agent"
	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/deploy"
	"github.com/hsyan2008/hfw/serve"
	"github.com/hsyan2008/hfw/signal"
)

//Run start
func Run() (err error) {

	logger.Mix("Starting ...")
	defer logger.Mix("Shutdowned!")

	logger.Mixf("Running, VERSION=%s, ENVIRONMENT=%s, APPNAME=%s, APPPATH=%s",
		common.GetVersion(), common.GetEnv(), common.GetAppName(), common.GetAppPath())

	if err = agent.Listen(agent.Options{}); err != nil {
		logger.Fatal(err)
		return
	}

	signalContext := signal.GetSignalContext()

	//监听信号
	go signalContext.Listen()

	//等待工作完成
	defer signalContext.Shutdowned()

	if Config.HotDeploy.Enable {
		go deploy.HotDeploy(Config.HotDeploy)
	}

	if len(Config.Server.Address) == 0 {
		logger.Fatal("server address is nil")
		return
	}

	//启动http
	signalContext.IsHTTP = true

	err = serve.Start(Config.Server)

	//如果未启动服务，就触发退出
	if err != nil && err != http.ErrServerClosed {
		logger.Fatal(err)
	}

	return
}

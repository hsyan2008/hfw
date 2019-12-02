package hfw

import (
	"time"

	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/signal"
	cron "github.com/robfig/cron/v3"
)

var crontab *cron.Cron

func init() {
	crontab = cron.New(cron.WithSeconds())
	crontab.Start()
}

func AddCron(spec string, cmd func()) (cron.EntryID, error) {
	return crontab.AddFunc(spec, cmd)
}

func StopCron() {
	crontab.Stop()
}

func AddWrapCron(spec string, cmd func(httpCtx *HTTPContext) error) (cron.EntryID, error) {
	return AddCron(spec, func() {
		httpCtx := NewHTTPContext()
		signal.GetSignalContext().WgAdd()
		defer func(now time.Time) {
			if err := recover(); err != nil {
				if err != ErrStopRun {
					httpCtx.Warn(err, string(common.GetStack()))
				}
			}
			signal.GetSignalContext().WgDone()
			httpCtx.Infof("CostTime: %s", time.Since(now))
		}(time.Now())

		err := cmd(httpCtx)
		if err != nil {
			httpCtx.Warn(err)
		}
	})
}

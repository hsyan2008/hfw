package hfw

import (
	"time"

	"github.com/hsyan2008/hfw/common"
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
	return AddCron(spec, WrapCron(cmd))
}

func DoWrapCron(spec string, cmd func(httpCtx *HTTPContext) error) (cron.EntryID, error) {
	f := WrapCron(cmd)
	go f()
	return AddCron(spec, f)
}

func WrapCron(cmd func(httpCtx *HTTPContext) error) func() {
	return func() {
		httpCtx := NewHTTPContext()
		defer httpCtx.Cancel()
		defer func(now time.Time) {
			if err := recover(); err != nil {
				if err != ErrStopRun {
					httpCtx.Warn(err, string(common.GetStack()))
				}
			}
			httpCtx.Infof("CostTime: %s", time.Since(now))
		}(time.Now())

		err := cmd(httpCtx)
		if err != nil {
			httpCtx.Warn(err)
		}
	}
}

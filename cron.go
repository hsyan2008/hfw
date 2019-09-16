package hfw

import (
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

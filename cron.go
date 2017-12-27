package hfw

import "github.com/robfig/cron"

var crontab *cron.Cron

func init() {
	crontab = cron.New()
	crontab.Start()
}

func AddCron(spec string, cmd func()) error {
	return crontab.AddFunc(spec, cmd)
}

func StopCron() {
	crontab.Stop()
}

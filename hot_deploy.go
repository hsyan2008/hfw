package hfw

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hsyan2008/go-logger/logger"
	"github.com/hsyan2008/hfw2/common"
	"github.com/hsyan2008/hfw2/configs"
	"github.com/shirou/gopsutil/process"
)

func HotDeploy(hotDeploy configs.HotDeployConfig) {

	if hotDeploy.Dep <= 0 || hotDeploy.Dep > 10 {
		hotDeploy.Dep = 5
	}

	signalContext.WgAdd()
	defer signalContext.WgDone()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Fatal(err)
		return
	}
	defer watcher.Close()

	err = addWatch(watcher, APPPATH, 0)
	if err != nil {
		logger.Fatal(err)
		return
	}

	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		logger.Warn("os FindProcess failed:", err)
		return
	}

	var isRestart bool
	for {
		select {
		case <-signalContext.Ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			logger.Debug("event:", event)
			if event.Op&fsnotify.Create == fsnotify.Create {
				logger.Info("modified file:", event.Name)
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				logger.Info("modified file:", event.Name)
			}
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				logger.Info("modified file:", event.Name)
			}
			if event.Op&fsnotify.Rename == fsnotify.Rename {
				logger.Info("modified file:", event.Name)
			}
			if event.Op&fsnotify.Chmod == fsnotify.Chmod {
				logger.Info("modified file:", event.Name)
			}

			baseName := filepath.Base(event.Name)
			if baseName[:1] == "." {
				continue
			}
			if len(hotDeploy.Exts) > 0 {
				ext := filepath.Ext(event.Name)
				if len(ext) > 0 {
					ext = filepath.Ext(event.Name)[1:]
				}
				var isFind bool
				for _, v := range hotDeploy.Exts {
					if v == ext || v == baseName {
						isFind = true
						break
					}
				}
				if !isFind {
					continue
				}
			}

			isRestart = true

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logger.Warn("error:", err)
		case <-time.After(time.Second):
			if !isRestart {
				continue
			}
			if common.IsGoRun() {
				err = p.Signal(syscall.SIGINT)
				if err != nil {
					logger.Warn("send signal sigterm failed:", err)
					return
				}
				if len(hotDeploy.Cmd) == 0 {
					pp, err := process.NewProcess(int32(os.Getppid()))
					if err != nil {
						return
					}
					hotDeploy.Cmd, err = pp.CmdlineSlice()
					if err != nil {
						return
					}
				}

				execSpec := &os.ProcAttr{
					Dir:   APPPATH,
					Env:   os.Environ(),
					Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
				}

				hotDeploy.Cmd[0], err = exec.LookPath(hotDeploy.Cmd[0])
				if err != nil {
					return
				}
				_, err = os.StartProcess(hotDeploy.Cmd[0], append(hotDeploy.Cmd, os.Args[1:]...), execSpec)
				if err != nil {
					return
				}
			} else {
				err = p.Signal(syscall.SIGTERM)
				if err != nil {
					logger.Warn("send signal sigterm failed:", err)
					return
				}
			}
		}
	}
}

func addWatch(watcher *fsnotify.Watcher, path string, dep int) (err error) {
	if dep > 5 {
		return
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	fi, err := f.Stat()
	if err != nil {
		return
	}
	if fi.IsDir() {
		err = watcher.Add(path)
		if err != nil {
			logger.Fatal(err)
			return
		}

		fs, err := f.Readdir(0)
		if err != nil {
			return err
		}
		for _, f := range fs {
			if f.Name()[:1] == "." {
				continue
			}
			addWatch(watcher, filepath.Join(path, f.Name()), dep+1)
		}
	}

	return
}

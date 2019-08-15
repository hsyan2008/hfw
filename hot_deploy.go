package hfw

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/configs"
	"github.com/hsyan2008/hfw/signal"
	"github.com/shirou/gopsutil/process"
)

func HotDeploy(hotDeployConfig configs.HotDeployConfig) {

	if hotDeployConfig.Dep <= 0 || hotDeployConfig.Dep > 10 {
		hotDeployConfig.Dep = 5
	}

	signalContext := signal.GetSignalContext()

	signalContext.WgAdd()
	defer signalContext.WgDone()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Fatal(err)
		return
	}
	defer watcher.Close()

	err = addWatch(watcher, hotDeployConfig, APPPATH, 0)
	if err != nil {
		logger.Fatal(err)
		return
	}

	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		logger.Warn("os FindProcess failed:", err)
		return
	}

	var isToRestart bool
	for {
		select {
		case <-signalContext.Ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			baseName := filepath.Base(event.Name)
			if baseName[:1] == "." {
				continue
			}
			if len(hotDeployConfig.Exts) > 0 {
				ext := filepath.Ext(event.Name)
				if len(ext) > 0 {
					ext = filepath.Ext(event.Name)[1:]
				}
				var isTrigger bool
				for _, v := range hotDeployConfig.Exts {
					if v == ext || v == baseName {
						isTrigger = true
						break
					}
				}
				if !isTrigger {
					continue
				}
			}

			isToRestart = true

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logger.Warn("error:", err)
		case <-time.After(time.Second):
			if !isToRestart {
				continue
			}
			if common.IsGoRun() {
				err = p.Signal(syscall.SIGINT)
				if err != nil {
					logger.Warn("send signal sigterm failed:", err)
					return
				}
				var cmd []string
				if len(hotDeployConfig.Cmd) == 0 {
					pp, err := process.NewProcess(int32(os.Getppid()))
					if err != nil {
						return
					}
					cmd, err = pp.CmdlineSlice()
					if err != nil {
						return
					}
				} else {
					cmd = strings.Fields(hotDeployConfig.Cmd)
				}

				execSpec := &os.ProcAttr{
					Dir:   APPPATH,
					Env:   os.Environ(),
					Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
				}

				cmd[0], err = exec.LookPath(cmd[0])
				if err != nil {
					return
				}
				_, err = os.StartProcess(cmd[0], append(cmd, os.Args[1:]...), execSpec)
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

func addWatch(watcher *fsnotify.Watcher, hotDeployConfig configs.HotDeployConfig, path string, dep int) (err error) {
	if dep > hotDeployConfig.Dep {
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
			addWatch(watcher, hotDeployConfig, filepath.Join(path, f.Name()), dep+1)
		}
	}

	return
}

package hfw

//问题较多，见最下方的runCmd
//另外，通过ExecOutput执行ssh命令，结束的时候，第一个按键都无法捕获
import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strings"
	"syscall"

	termbox "github.com/nsf/termbox-go"
)

const (
	HISTMAX = 1024
	PREFIX  = "\033[32;1m>> \033[0m"
)

var (
	termPrefix = PREFIX
	cmd        *exec.Cmd
	cmdText    string
	inputMode  = NormalMode

	history             [][]rune
	historyCurrentIndex int
	input               []rune
	inputLen            int
	termWidth           int
	termHeight          int
)

type InputMode uint8

const (
	//默认模式
	NormalMode InputMode = iota
	//应用自己的模式，在这个模式下，所有输入都由应用处理，如help
	APPMode
)

func SetPrefix(prefix string) {
	termPrefix = prefix
}

func ResetPrefix() {
	termPrefix = PREFIX
}

func SetMode(i InputMode) {
	inputMode = i
}

func GetMode() InputMode {
	return inputMode
}

func initTerm() {

	isInit := termbox.IsInit

	if isInit {
		cmd = nil
		//某些命令会接管tty，重置一下
		if !isTermReset() {
			return
		}
		termbox.Close()
	}

	err := termbox.Init()
	if err != nil {
		panic(err)
	}

	if !isInit {
		termWidth, termHeight = termbox.Size()
		fmt.Printf(" Welcome to use %s. Enter ? or help for help\n", APPNAME)
	}
	//显示光标
	fmt.Print("\033[?25h")
}

func StartTerminal(fn func(string)) {

	initTerm()
	defer termbox.Close()

	go termSignal()

	fmt.Print(termPrefix)

	for {
		if historyCurrentIndex < len(history) {
			inputLen = len(history[historyCurrentIndex])
		} else {
			inputLen = len(input)
		}
		ev := termbox.PollEvent()
		switch ev.Type {
		case termbox.EventKey:
			switch ev.Key {
			case termbox.KeyCtrlD:
				if inputMode != APPMode {
					return
				}
				//退出ssh
				fn("quit")
				fmt.Print("\n")
				fmt.Print(termPrefix)
			case termbox.KeyCtrlC:
				if cmd != nil {
					killCommandIns()
				} else {
					fmt.Print("\n")
					fmt.Print(termPrefix)
				}
				input = make([]rune, 0)
				historyCurrentIndex = len(history)
			case termbox.KeySpace:
				fmt.Print(" ")
				historyToInput()
				input = append(input, rune(' '))
			case termbox.KeyTab:
				//不支持自动完成，屏蔽tab键
				continue
				historyToInput()
				i := 4 - inputLen%4
				for j := 0; j < i; j++ {
					fmt.Print(" ")
					input = append(input, rune(' '))
				}
			case termbox.KeyBackspace2:
				if inputLen > 0 {
					fmt.Print("\b \b")
					historyToInput()
					input = input[0 : inputLen-1]
				}
			case termbox.KeyCtrlU:
				if inputLen > 0 {
					cleanLine(inputLen)
					input = make([]rune, 0)
					historyCurrentIndex = len(history)
				}
			case termbox.KeyCtrlL:
				//清屏
				fmt.Print("\033[2J")
				//光标设置为左上角
				fmt.Print("\033[0;0H")

				fmt.Print(termPrefix)
				if historyCurrentIndex < len(history) {
					fmt.Print(string(history[historyCurrentIndex]))
				} else {
					fmt.Print(string(input))
				}
			case termbox.KeyEnter:
				fmt.Print("\n")
				if cmd != nil {
					continue
				}
				if inputLen > 0 {
					if historyCurrentIndex < len(history) {
						input = history[historyCurrentIndex]
						historyCurrentIndex = len(history)
					}
					if len(history) == 0 || string(input) != string(history[len(history)-1]) {
						history = append(history, input)
						if len(history) > HISTMAX {
							history = history[0:HISTMAX]
						}
					}
					cmdText = strings.TrimSpace(string(input))
					if inputMode == NormalMode {
						if cmdText == "quit" || cmdText == "exit" {
							return
						}
						if cmdText == "history" {
							termHistory()
						} else if cmdText == "?" || cmdText == "help" {
							termHelp()
						}
					}
					fn(cmdText)

					input = make([]rune, 0)
				}
				if cmd == nil {
					fmt.Print(termPrefix)
				}
			case termbox.KeyArrowUp, termbox.KeyPgup, termbox.MouseWheelUp:
				if historyCurrentIndex > 0 {
					historyCurrentIndex--
					if inputLen > 0 {
						cleanLine(inputLen)
					}
					fmt.Print(string(history[historyCurrentIndex]))
				}
			case termbox.KeyArrowDown, termbox.KeyPgdn, termbox.MouseWheelDown:
				if historyCurrentIndex < len(history) {
					historyCurrentIndex++
					if inputLen > 0 {
						cleanLine(inputLen)
					}
					if historyCurrentIndex == len(history) {
						fmt.Print(string(input))
					} else {
						fmt.Print(string(history[historyCurrentIndex]))
					}
				}
			default:
				if cmd != nil {
					continue
				}
				input = append(input, ev.Ch)
				fmt.Print(string(ev.Ch))
			}
		case termbox.EventInterrupt:
			fmt.Println("Interrupt")
		}
	}
}

func historyToInput() {
	if historyCurrentIndex < len(history) {
		input = history[historyCurrentIndex]
		historyCurrentIndex = len(history)
	}
}

func cleanLine(inputLen int) {
	fmt.Print(strings.Repeat("\b", inputLen))
	fmt.Print(strings.Repeat(" ", inputLen))
	fmt.Print(strings.Repeat("\b", inputLen))
}

func termHelp() {
	fmt.Println("  Shortcut key:")
	fmt.Println("    Ctrl+L                           clean terminal")
	fmt.Println("    Ctrl+U                           clean left input char")
	fmt.Println("    ArrowUp/PageUp/MouseUp           show prev history")
	fmt.Println("    ArrowDown/PageDown/MouseDown     show next history")
	fmt.Println("    Ctrl+D                           exit terminal")
	fmt.Println("  Command:")
	fmt.Println("    history                          show input history")
}

func termHistory() {
	l := len(fmt.Sprint(len(history)))
	f := fmt.Sprintf(" %%%dd  %%s\n", l)
	for k, v := range history {
		fmt.Printf(f, k+1, string(v))
	}
}

//防止top类命令执行的时候，ctrl+c导致进程退出
func termSignal() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT)
	for {
		<-c
	}
}

//top、mysql等命令执行完毕退出要重置，否则方向键有问题
func isTermReset() bool {
	fields := strings.Fields(cmdText)
	name := path.Base(fields[0])

	switch name {
	case "top", "mysql":
		return true
	}

	return false
}

//示例，调用方法为StartTerminal(RunCmd)
func RunCmd(text string) {
	//mysql不用输入密码的可以用，ssh不可以
	//ls等直接返回的没有问题
	//tail等没有快捷键的，只要别ctrl+c，也没问题，否则会导致程序退出
	//top，按ctrl+c现象同上，按q可以退出，但影响方向键(PageUp/PageDown不影响)
	//采用重新init的方式，解决top问题，但不完美
	cmd = exec.Command("sh", "-c", text)
	//stdin没有的话，不影响快捷键，但不支持top命令，报tty不存在
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	//如果是Run，top可以退出，但tail无法退出
	_ = cmd.Start()

	go func() {
		_ = cmd.Wait()

		initTerm()
		fmt.Print(termPrefix)
	}()
}

//对于tail等不接管tty的命令，按ctrl+c才有效
func killCommandIns() {
	if cmd != nil {
		_ = cmd.Process.Kill()
	}
}

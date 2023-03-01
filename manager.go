package reload

import (
	"context"
	"github.com/evolidev/evoli/framework/logging"
	"github.com/evolidev/evoli/framework/use"
	"log"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Manager struct {
	*Configuration
	ID         string
	Logger     *logging.Logger
	Restart    chan bool
	Kill       chan bool
	cancelFunc context.CancelFunc
	context    context.Context
	gil        *sync.Once
	cmd        *exec.Cmd
}

func New(c *Configuration) *Manager {
	return NewWithContext(c, context.Background())
}

func NewWithContext(c *Configuration, ctx context.Context) *Manager {
	ctx, cancelFunc := context.WithCancel(ctx)
	m := &Manager{
		Configuration: c,
		ID:            ID(),
		Logger: logging.NewLogger(&logging.Config{
			Name:         "reload",
			EnableColors: true,
			PrefixColor:  148,
		}),
		Restart:    make(chan bool),
		Kill:       make(chan bool),
		cancelFunc: cancelFunc,
		context:    ctx,
		gil:        &sync.Once{},
	}
	return m
}

func (m *Manager) Start() error {
	w := NewWatcher(m)
	w.Start()

	go m.build(fsnotify.Event{Name: ":start:"})

	//go m.build()

	restart := func() {
		m.Restart <- true
	}

	debounced := use.Debounce(100 * time.Millisecond)

	if !m.Debug {
		go func() {
		LoopRebuilder:
			for {
				select {
				case event := <-w.Events():
					if !w.isFileEligibleForChange(event.Name) {
						continue
					}

					if event.Op != fsnotify.Chmod {
						debounced(restart)
					}

					if w.ForcePolling {
						//w.Logger.Print("Removing file from watchlist: %s", event.Name)
						w.Remove(event.Name)
						w.Add(event.Name)
					}

				case <-m.context.Done():
					m.Logger.Print("Shutting down from Start")
					break LoopRebuilder
				}
			}
		}()
	}

	go func() {
		for {
			select {
			case err := <-w.Errors():
				m.Logger.Error("Manager error", err)
			case <-m.context.Done():
				break
			}
		}
	}()

	m.runner()
	return nil
}

func (m *Manager) Stop() {
	m.Logger.Print("Stopping from Manager:Stop")
	//m.Kill <- true
	//m.cancelFunc()

	time.Sleep(3 * time.Second)
}

func (m *Manager) makeCmd() *exec.Cmd {

	//m.Logger.Print("Rebuild on: %s", event.Name)

	command, args := m.getCommandArguments()
	cmd := exec.CommandContext(m.context, command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	cmd.Dir = m.AppRoot

	return cmd
}

func (m *Manager) build(event fsnotify.Event) {

	time.Sleep(100 * time.Millisecond)
	m.Restart <- true

	//r.gil.Do(func() {
	//	defer func() {
	//		r.gil = &sync.Once{}
	//	}()
	//	// time.Sleep(r.BuildDelay * time.Millisecond)
	//
	//	//now := time.Now()
	//	//r.Logger.Print("Rebuild on: %s", event.Name)
	//
	//	//args := []string{"build", "-v"}
	//	//args = append(args, r.BuildFlags...)
	//	//args = append(args, "-o", r.FullBuildPath(), r.BuildTargetPath)
	//	//cmd := exec.CommandContext(r.context, "go", args...)
	//	//cmd := r.makeCmd()
	//	//
	//	//err := r.runAndListen(cmd)
	//	//if err != nil {
	//	//	if strings.Contains(err.Error(), "no buildable Go source files") {
	//	//		r.cancelFunc()
	//	//		log.Fatal(err)
	//	//	}
	//	//	return
	//	//}
	//	//
	//	//tt := time.Since(now)
	//	//r.Logger.Success("Building Completed (PID: %d) (Time: %s)", cmd.Process.Pid, tt)
	//	r.Restart <- true
	//	return
	//})
}

func (m *Manager) run(cmd *exec.Cmd) *exec.Cmd {

	timer := use.TimeRecord()
	err := m.runAndListen(cmd)
	m.cmd = cmd
	if err != nil {
		if strings.Contains(err.Error(), "no buildable Go source files") {
			m.cancelFunc()
			log.Fatal(err)
		}
		return nil
	}

	m.Logger.Success("Buildings Completed (PID: %d) %s",
		cmd.Process.Pid,
		timer.ElapsedColored(),
	)
	return cmd
}

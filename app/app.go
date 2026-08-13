package app

import (
	"github.com/rusl222/scada/logger"

	"github.com/rusl222/scada/debug"

	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	scada "github.com/rusl222/scada/types"
)

type App struct {
	logger *slog.Logger
	sup    *Supervisor
	Broker scada.Api
	Debug  bool
}

type DebugLevel int

func NewApp(br scada.Api, debugMode bool, webport int) *App {

	// Настройка логирования
	logger := logger.Logger(debugMode)

	if debugMode {
		// Оборачиваем в debug-прокси
		br = debug.RunDebugger(br, "./lib/debug/webui", webport)
		logger.Info("debug mode active")
	}

	// Супервайзер управляет выполнением алгоблоков
	sup := NewSupervisor(logger.With("module", "supervisor"), br)

	a := &App{
		logger: logger,
		sup:    sup,
		Broker: br,
		Debug:  debugMode,
	}
	return a
}

func (a *App) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := a.sup.InitAll(); err != nil {
		return err
	}

	a.logger.Info("supervisor started")
	a.sup.Start(ctx)
	a.logger.Info("supervisor stopped")

	return nil
}

func (a *App) RegisterModule(m Module, cfg ModuleConfig) {
	a.sup.Register(m, cfg)
}

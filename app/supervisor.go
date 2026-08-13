package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Module describes the minimal lifecycle a supervised module must expose.
// - OnInit is called once when the supervisor initializes modules.
// - OnTick is called repeatedly by the supervisor's tick loop.
// - OnStop is called when the supervisor is shutting down.
type Module interface {
	OnInit() error
	OnTick() error
	OnStop() error
}

// ModuleConfig contains per-module runtime settings.
type ModuleConfig struct {
	Name         string
	TickInterval time.Duration
}

func (c ModuleConfig) normalized() ModuleConfig {
	if c.TickInterval <= 0 {
		c.TickInterval = defaultTickInterval
	}
	return c
}

// ModuleStats holds timing and run information for a module.
type ModuleStats struct {
	LastDuration time.Duration
	AvgDuration  time.Duration
	Runs         int64
	LastRun      time.Time
	LastError    error

	// Panic / restart tracking
	Restarts  int
	LastPanic string

	mu sync.Mutex
}

// Snapshot returns a copy of the stats suitable for external inspection.
func (s *ModuleStats) Snapshot() ModuleStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return ModuleStats{
		LastDuration: s.LastDuration,
		AvgDuration:  s.AvgDuration,
		Runs:         s.Runs,
		LastRun:      s.LastRun,
		LastError:    s.LastError,
	}
}

type moduleRuntime struct {
	module Module
	cfg    ModuleConfig
}

// Supervisor manages registered modules, runs their OnInit and periodically calls OnTick.
type Supervisor struct {
	mu      sync.RWMutex
	modules map[string]Module
	configs map[string]ModuleConfig
	stats   map[string]*ModuleStats
	runners map[string]context.CancelFunc
	brocker connchecker

	log *slog.Logger
}

const defaultTickInterval = time.Second
const defaultRestartLimit = 0

type connchecker interface {
	Connected() bool
}

// New creates a Supervisor with the provided tick interval. If tickInterval <= 0,
// a default of 1s is used.
func NewSupervisor(slog *slog.Logger, br connchecker) *Supervisor {

	return &Supervisor{
		modules: make(map[string]Module),
		configs: make(map[string]ModuleConfig),
		stats:   make(map[string]*ModuleStats),
		runners: make(map[string]context.CancelFunc),
		log:     slog,
		brocker: br,
	}
}

// Register adds a module to the supervisor. It returns an error when a module
// with the same name is already registered.
func (s *Supervisor) Register(m Module, cfg ModuleConfig) (err error) {
	defer func() {
		if err != nil {
			if s.log != nil {
				s.log.Error("module registration failed", "module", cfg.Name, "error", err)
			}
		}
	}()
	if m == nil {
		return errors.New("module is nil")
	}
	cfg = cfg.normalized()
	if cfg.Name == "" {
		return errors.New("module name is empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.modules[cfg.Name]; ok {
		return errors.New("module already registered: " + cfg.Name)
	}

	s.modules[cfg.Name] = m
	s.configs[cfg.Name] = cfg
	s.stats[cfg.Name] = &ModuleStats{}
	return nil
}

// Unregister removes a module by name.
func (s *Supervisor) Unregister(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.modules, name)
	delete(s.configs, name)
	delete(s.stats, name)
}

// InitAll runs OnInit on all registered modules. It returns the first error encountered.
func (s *Supervisor) InitAll() error {

	for {
		if s.brocker.Connected() {
			break
		}
		time.Sleep(1 * time.Second)
	}

	s.mu.RLock()
	mods := make([]Module, 0, len(s.modules))
	names := make([]string, 0, len(s.modules))
	for name, m := range s.modules {
		mods = append(mods, m)
		names = append(names, name)
	}
	s.mu.RUnlock()

	for i, m := range mods {
		if err := s.callModuleInit(m, names[i]); err != nil {
			if s.log != nil {
				s.log.Error("module init failed", "module", names[i], "error", err)
			}
			return err
		}
	}
	return nil
}

// Start runs the periodic tick loop until the provided context is cancelled.
// Each module is executed independently and its own tick interval is used.
func (s *Supervisor) Start(ctx context.Context) {
	runtimes := s.snapshotRuntimes()
	for _, rt := range runtimes {
		s.startModuleLoop(ctx, rt.module, rt.cfg, true)
	}

	<-ctx.Done()
	s.stopModules()
}

// StartModule starts a single module by name if it is registered.
// It runs OnInit once and then performs an immediate OnTick before the periodic loop starts.
func (s *Supervisor) StartModule(name string) error {
	if name == "" {
		return errors.New("module name is empty")
	}

	s.mu.RLock()
	mod, ok := s.modules[name]
	cfg, cfgOk := s.configs[name]
	_, alreadyRunning := s.runners[name]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("module not found: %s", name)
	}
	if alreadyRunning {
		return fmt.Errorf("module already running: %s", name)
	}
	if !cfgOk {
		cfg = ModuleConfig{Name: name, TickInterval: defaultTickInterval}
	}

	cfg = cfg.normalized()
	if err := s.runModuleInit(mod, name); err != nil {
		return err
	}
	if err := s.runModuleTickOnce(mod, cfg); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.runners[name] = cancel
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.runners, name)
			s.mu.Unlock()
		}()
		s.runModuleLoop(ctx, mod, cfg, false)
	}()
	return nil
}

func (s *Supervisor) snapshotRuntimes() []moduleRuntime {
	s.mu.RLock()
	defer s.mu.RUnlock()

	runtimes := make([]moduleRuntime, 0, len(s.modules))
	for name, mod := range s.modules {
		cfg, ok := s.configs[name]
		if !ok {
			cfg = ModuleConfig{
				Name:         name,
				TickInterval: defaultTickInterval,
			}
		}
		runtimes = append(runtimes, moduleRuntime{module: mod, cfg: cfg})
	}
	return runtimes
}

func (s *Supervisor) startModuleLoop(parent context.Context, mod Module, cfg ModuleConfig, runInitialTick bool) {
	if mod == nil {
		return
	}
	if cfg.Name == "" {
		return
	}

	ctx, cancel := context.WithCancel(parent)
	s.mu.Lock()
	s.runners[cfg.Name] = cancel
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.runners, cfg.Name)
			s.mu.Unlock()
		}()
		s.runModuleLoop(ctx, mod, cfg, runInitialTick)
	}()
}

func (s *Supervisor) runModuleLoop(ctx context.Context, mod Module, cfg ModuleConfig, runInitialTick bool) {
	if runInitialTick {
		s.runModuleTick(mod, cfg)
	}
	for {
		if ctx.Err() != nil {
			return
		}

		timer := time.NewTimer(cfg.TickInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}

		s.runModuleTick(mod, cfg)
	}
}

func (s *Supervisor) runModuleInit(mod Module, name string) error {
	if mod == nil {
		return nil
	}
	return s.callModuleInit(mod, name)
}

func (s *Supervisor) runModuleTickOnce(mod Module, cfg ModuleConfig) error {
	if mod == nil {
		return nil
	}
	return s.runModuleTickWithError(mod, cfg)
}

func (s *Supervisor) runModuleTick(mod Module, cfg ModuleConfig) {
	_ = s.runModuleTickWithError(mod, cfg)
}

func (s *Supervisor) runModuleTickWithError(mod Module, cfg ModuleConfig) error {
	if mod == nil {
		return nil
	}

	name := cfg.Name
	if name == "" {
		return nil
	}

	s.mu.RLock()
	st, ok := s.stats[name]
	s.mu.RUnlock()
	if !ok {
		return nil
	}

	start := time.Now()

	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				st.mu.Lock()
				st.Restarts++
				st.LastPanic = fmt.Sprint(r)
				st.LastError = fmt.Errorf("panic: %v", r)
				st.mu.Unlock()

				if s.log != nil {
					s.log.Error("module panicked", "module", name, "panic", r, "restarts", st.Restarts)
				}
			}
		}()
		if s.brocker.Connected() {
			err = mod.OnTick()
		}
	}()

	dur := time.Since(start)

	st.mu.Lock()
	st.LastDuration = dur
	st.Runs++
	if st.Runs == 1 {
		st.AvgDuration = dur
	} else {
		prev := st.AvgDuration
		n := float64(st.Runs)
		st.AvgDuration = time.Duration((prev.Seconds()*(n-1) + dur.Seconds()) / n * float64(time.Second))
	}
	st.LastRun = time.Now()
	if err != nil {
		st.LastError = err
	}
	st.mu.Unlock()
	return err
}

func (s *Supervisor) callModuleInit(mod Module, name string) error {
	if mod == nil {
		return nil
	}
	if s.log != nil {
		s.log.Info("initializing module", "algoblock", name)
	}

	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				if s.log != nil {
					s.log.Error("panic in OnInit", "module", name, "panic", r)
				}
				err = fmt.Errorf("panic in OnInit: %v", r)
			}
		}()
		err = mod.OnInit()
	}()
	return err
}

func (s *Supervisor) stopModules() {
	s.mu.RLock()
	runners := make([]context.CancelFunc, 0, len(s.runners))
	for _, cancel := range s.runners {
		runners = append(runners, cancel)
	}
	s.mu.RUnlock()
	for _, cancel := range runners {
		if cancel != nil {
			cancel()
		}
	}

	runtimes := s.snapshotRuntimes()
	for _, rt := range runtimes {
		s.stopModule(rt.module, rt.cfg.Name)
	}
}

// StopModule stops a single module by name if it is registered.
// It returns true when a module with that name was found and stopped.
func (s *Supervisor) StopModule(name string) bool {
	if name == "" {
		return false
	}

	s.mu.RLock()
	mod, ok := s.modules[name]
	cancel, hasRunner := s.runners[name]
	s.mu.RUnlock()
	if !ok {
		return false
	}

	if hasRunner && cancel != nil {
		cancel()
	}
	s.stopModule(mod, name)
	return true
}

func (s *Supervisor) stopModule(mod Module, name string) {
	if mod == nil {
		return
	}

	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("panic in OnStop for %s: %v\n", name, r)
			}
		}()
		mod.OnStop()
	}()
}

// StatsSnapshot returns a map of module name → ModuleStats snapshot.
func (s *Supervisor) StatsSnapshot() map[string]ModuleStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]ModuleStats, len(s.stats))
	for name, st := range s.stats {
		out[name] = st.Snapshot()
	}
	return out
}

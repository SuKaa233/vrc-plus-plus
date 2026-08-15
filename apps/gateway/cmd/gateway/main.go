package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/local/vrc-web-companion/gateway/internal/desktop"
	"github.com/local/vrc-web-companion/gateway/internal/diagnostics"
	"github.com/local/vrc-web-companion/gateway/internal/events"
	"github.com/local/vrc-web-companion/gateway/internal/gamelog"
	"github.com/local/vrc-web-companion/gateway/internal/localapi"
	"github.com/local/vrc-web-companion/gateway/internal/media"
	"github.com/local/vrc-web-companion/gateway/internal/pipeline"
	"github.com/local/vrc-web-companion/gateway/internal/security"
	"github.com/local/vrc-web-companion/gateway/internal/singleinstance"
	"github.com/local/vrc-web-companion/gateway/internal/storage"
	"github.com/local/vrc-web-companion/gateway/internal/tray"
	"github.com/local/vrc-web-companion/gateway/internal/updater"
	"github.com/local/vrc-web-companion/gateway/internal/vrchat"
	"github.com/local/vrc-web-companion/gateway/web"
)

const (
	appName = "VRC++"
)

var version = "0.9.0-beta.1"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "gateway:", err)
		os.Exit(1)
	}
}

func run() error {
	listenAddress := flag.String("listen", "127.0.0.1:47831", "local gateway listen address")
	dataDirectory := flag.String("data-dir", defaultDataDirectory(), "local data directory")
	devOrigin := flag.String("dev-origin", "", "allowed Vite development origin")
	shouldUseDesktop := flag.Bool("desktop", true, "open the web UI in an embedded WebView2 window")
	shouldOpenBrowser := flag.Bool("open-browser", false, "use the default browser instead of the desktop window")
	shouldShowTray := flag.Bool("tray", true, "show a Windows notification area icon")
	updateHelper := flag.Bool("update-helper", false, "run the detached update replacement helper")
	replaceFrom := flag.String("replace-from", "", "staged executable path")
	replaceTarget := flag.String("replace-target", "", "installed executable path")
	waitPID := flag.Int("wait-pid", 0, "process id to wait for")
	flag.Parse()
	if *updateHelper {
		return updater.RunHelper(*replaceFrom, *replaceTarget, *waitPID)
	}
	instance, primary, err := singleinstance.Acquire("VRCPlusPlus")
	if err != nil {
		return err
	}
	if !primary {
		return nil
	}
	defer instance.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx := context.Background()
	store, err := storage.Open(ctx, filepath.Join(*dataDirectory, "harbor.db"))
	if err != nil {
		return err
	}
	defer store.Close()

	protector := security.NewProtector()
	userAgent := envOrDefault("VRC_HARBOR_USER_AGENT", "VRCPlusPlus/0.9.0-beta.1 contact@example.invalid")
	baseURL := envOrDefault("VRC_HARBOR_VRCHAT_BASE_URL", "https://api.vrchat.cloud/api/1")
	vrcClient, err := vrchat.NewClient(baseURL, userAgent, store, protector)
	if err != nil {
		return err
	}
	networkConfig, err := vrchat.LoadNetworkConfig(ctx, store)
	if err != nil {
		return err
	}
	if _, err := vrcClient.ApplyNetworkConfig(ctx, networkConfig); err != nil {
		return err
	}
	diagnosticService := diagnostics.New(store, vrcClient)
	eventBus := events.NewBus()
	gameLogWatcher := gamelog.New(gamelog.DefaultDirectory(), eventBus, logger)
	gameLogWatcher.Start()
	defer gameLogWatcher.Stop()
	historyEvents, unsubscribeHistory := eventBus.Subscribe()
	historyCtx, stopHistory := context.WithCancel(context.Background())
	defer func() {
		stopHistory()
		unsubscribeHistory()
	}()
	go func() {
		for {
			select {
			case <-historyCtx.Done():
				return
			case event := <-historyEvents:
				if err := store.RecordDomainEvent(historyCtx, event); err != nil && !errors.Is(err, context.Canceled) {
					logger.Warn("could not persist local activity event", "type", event.Type)
				}
			}
		}
	}()
	pipelineManager := pipeline.New(vrcClient, eventBus, logger, envOrDefault("VRC_HARBOR_PIPELINE_URL", "wss://pipeline.vrchat.cloud"))
	defer pipelineManager.Stop()
	mediaService, err := media.New(filepath.Join(*dataDirectory, "media-cache"), vrcClient, logger)
	if err != nil {
		return err
	}
	updateService := updater.New(version, *dataDirectory, nil, logger)
	shutdownRequest := make(chan struct{}, 1)
	apiServer, err := localapi.New(localapi.Config{
		AppName:      appName,
		Version:      version,
		DevOrigin:    *devOrigin,
		StaticFS:     web.Dist(),
		SecurityName: protector.Name(),
		Store:        store,
		GameLog:      gameLogWatcher,
		Updater:      updateService,
		Shutdown:     shutdownRequest,
	}, vrcClient, diagnosticService, eventBus, pipelineManager, mediaService, logger)
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", *listenAddress)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", *listenAddress, err)
	}
	server := &http.Server{
		Handler:           apiServer.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	localURL := "http://" + listener.Addr().String()
	logger.Info("VRC++ gateway started", "url", localURL, "data", *dataDirectory)
	appContext, cancelApp := context.WithCancel(context.Background())
	defer cancelApp()
	windowManager := desktop.New()
	var desktopMode atomic.Bool
	desktopMode.Store(*shouldUseDesktop && !*shouldOpenBrowser)
	openInBrowser := func() {
		go func() {
			if err := openBrowser(localURL); err != nil {
				logger.Warn("could not open default browser", "error", err)
			}
		}()
	}
	openApp := func() {
		if desktopMode.Load() {
			windowManager.Show()
			return
		}
		openInBrowser()
	}
	if *shouldShowTray {
		tray.Start(openApp, openInBrowser, shutdownRequest)
		defer tray.Stop()
	}
	go func() {
		for range instance.Activations() {
			openApp()
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- server.Serve(listener)
		cancelApp()
	}()
	go func() {
		select {
		case <-stop:
		case <-shutdownRequest:
		case <-appContext.Done():
		}
		cancelApp()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if desktopMode.Load() {
		err = windowManager.Run(appContext, localURL, *dataDirectory)
		if errors.Is(err, desktop.ErrUnavailable) {
			desktopMode.Store(false)
			logger.Warn("WebView2 is unavailable; using the default browser")
			openInBrowser()
			<-appContext.Done()
		} else if err != nil {
			return err
		}
	} else {
		openInBrowser()
		<-appContext.Done()
	}
	err = <-serveDone
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func defaultDataDirectory() string {
	if configured := os.Getenv("VRC_HARBOR_DATA_DIR"); configured != "" {
		return configured
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(".", ".data")
	}
	current := filepath.Join(base, "VRC++")
	legacy := filepath.Join(base, "VRC Harbor")
	if _, err := os.Stat(current); err == nil {
		return current
	}
	if _, err := os.Stat(legacy); err == nil {
		return legacy
	}
	return current
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

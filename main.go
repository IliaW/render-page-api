package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/IliaW/render-page-api/config"
	"github.com/gin-gonic/gin"
)

var (
	cfg         *config.Config
	browserPool *BrowserPool
)

var (
	waitForBrowserTimeoutError = errors.New("too many requests. Increase 'browsers_count' or 'browser_wait' parameters")
	tooLongUploadDeadlineError = errors.New("page load timeout. Deadline exceeded")
	somethingWentWrongError    = errors.New("something went wrong =(")
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg = config.MustLoad()
	setupLogger()
	browserPool = NewBrowserPool(ctx, cfg)

	port := fmt.Sprintf(":%v", cfg.Port)
	srv := &http.Server{
		Addr:         port,
		Handler:      httpServer().Handler(),
		ReadTimeout:  cfg.HttpServerSettings.ReadTimeout,
		WriteTimeout: cfg.HttpServerSettings.WriteTimeout,
		IdleTimeout:  cfg.HttpServerSettings.IdleTimeout,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				slog.Info("server closed gracefully")
				return
			}
			slog.Error("listen:", slog.Any("err", err))
			os.Exit(1)
		}
	}()
	slog.Info(fmt.Sprintf("started %s on port %s. Env %s", cfg.ServiceName, cfg.Port, cfg.Env))

	<-ctx.Done()
	slog.Info("stopping server...")
	ctxT, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := srv.Shutdown(ctxT)
	if errors.Is(err, context.DeadlineExceeded) {
		slog.Error("shutdown timeout exceeded")
		os.Exit(1)
	}

	go browserPool.Close()
	<-ctxT.Done()
	slog.Info("application stopped")
}

func httpServer() *gin.Engine {
	setupGinMod()
	r := gin.New()
	r.UseH2C = true
	r.Use(gin.Recovery())
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{SkipPaths: []string{"/ping"}}))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	r.GET(cfg.RenderPagePath, renderPage)

	r.NoRoute(func(c *gin.Context) {
		c.AbortWithStatusJSON(http.StatusNotFound,
			gin.H{"message": fmt.Sprintf("no route found for %s %s", c.Request.Method, c.Request.URL)})
	})

	return r
}

func setupGinMod() {
	env := strings.ToLower(cfg.Env)
	if env == "dev" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
}

func setupLogger() *slog.Logger {
	envLogLevel := strings.ToLower(cfg.LogLevel)
	var slogLevel slog.Level
	err := slogLevel.UnmarshalText([]byte(envLogLevel))
	if err != nil {
		log.Printf("encountenred log level: '%s'. The package does not support custom log levels", envLogLevel)
		slogLevel = slog.LevelDebug
	}
	log.Printf("slog level overwritten to '%v'", slogLevel)
	slog.SetLogLoggerLevel(slogLevel)

	replaceAttrs := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.SourceKey {
			source := a.Value.Any().(*slog.Source)
			source.File = filepath.Base(source.File)
		}
		return a
	}

	var logger *slog.Logger
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource:   true,
		Level:       slogLevel,
		ReplaceAttr: replaceAttrs}))

	slog.SetDefault(logger)
	logger.Debug("debug messages are enabled.")

	return logger
}

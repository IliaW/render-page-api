package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/IliaW/render-page-api/config"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

type BrowserPool struct {
	ctx  context.Context
	cfg  *config.Config
	pool chan *rod.Browser
}

func NewBrowserPool(ctx context.Context, cfg *config.Config) *BrowserPool {
	slog.Info("initializing browser pool...")

	if cfg.BrowsersCount == 0 {
		slog.Error("browsers_count can't be 0. Update configuration file.")
		os.Exit(1)
	}

	bp := &BrowserPool{
		ctx:  ctx,
		cfg:  cfg,
		pool: make(chan *rod.Browser, cfg.BrowsersCount),
	}

	for i := 0; i < cfg.BrowsersCount; i++ {
		browser, err := bp.newBrowser()
		if err != nil {
			bp.Close()
			slog.Error("error when creating browsers pool.", err.Error())
			os.Exit(1)
		}
		bp.pool <- browser
	}
	slog.Info(fmt.Sprintf("launched %d browser(s)", cfg.BrowsersCount))

	return bp
}

func (bp *BrowserPool) Get() (*rod.Browser, error) {
	slog.Debug("waiting for browser from pool...")
	timeout, cancel := context.WithTimeout(bp.ctx, bp.cfg.BrowserWait)
	defer cancel()

	for {
		select {
		case <-timeout.Done():
			slog.Info("context canceled. Browser pool timeout exceeded.")
			return nil, waitForBrowserTimeoutError
		case browser := <-bp.pool:
			slog.Debug("get browser from pool.")
			return browser, nil
		}
	}
}

func (bp *BrowserPool) Put(browser *rod.Browser) {
	if browser != nil {
		bp.pool <- browser
		slog.Debug("return browser to pool.")
	}
}

func (bp *BrowserPool) Close() {
	slog.Info("closing browser pool...")
	close(bp.pool)

	for browser := range bp.pool {
		if err := browser.Close(); err != nil {
			slog.Error(fmt.Sprintf("error closing browser: %v", err))
		}
	}

	slog.Info("browser pool closed.")
}

func (bp *BrowserPool) newBrowser() (*rod.Browser, error) {
	slog.Debug("launching browser...")
	l := launcher.New().
		Leakless(cfg.EnableLeakless).
		Headless(cfg.HeadlessBrowser).
		Set("disable-background-timer-throttling").
		Set("disable-backgrounding-occluded-windows").
		Set("mute-audio")

	u := l.MustLaunch()
	browser := rod.New().
		ControlURL(u).
		MustConnect()

	slog.Debug("browser opened")
	return browser, nil
}

func takeScreenshot(page *rod.Page) {
	slog.Debug("taking screenshot...")
	if !browserPool.cfg.TakeScreenshot {
		return
	}
	screenshotBytes, _ := page.Screenshot(true, &proto.PageCaptureScreenshot{
		Format: proto.PageCaptureScreenshotFormatPng,
	})

	fullPath := filepath.Join("screenshots", fmt.Sprintf("page_%v.png", time.Now().UnixMilli()))
	tempFile, err := os.Create(fullPath)
	if err != nil {
		slog.Error("can't create temp file", slog.String("error", err.Error()))
		return
	}
	defer tempFile.Close()

	_, err = tempFile.Write(screenshotBytes)
	if err != nil {
		slog.Error("can't write to temp file", slog.String("error", err.Error()))
		return
	}

	slog.Debug("screenshot taken.", slog.String("path", fullPath))
}

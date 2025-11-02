package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
)

type RenderStatus string

const (
	RenderStatusSuccess RenderStatus = "success"
	RenderStatusFailed  RenderStatus = "failed"
	RenderStatusSkipped RenderStatus = "skipped"
)

type Response struct {
	URL        string       `json:"url"`
	Rendering  RenderStatus `json:"rendering"`
	StatusCode int          `json:"status_code,omitempty"`
	Error      string       `json:"error,omitempty"`
}

func renderPage(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		c.JSON(http.StatusBadRequest, newResponse("", RenderStatusSkipped, 0, "'url' is required"))
		return
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
		slog.Debug(fmt.Sprintf("protocol not specified. New URL: %s", url))
	}

	browser, err := browserPool.Get()
	if err != nil {
		c.JSON(http.StatusTooManyRequests, newResponse(url, RenderStatusSkipped, 0, waitForBrowserTimeoutError.Error()))
		return
	}
	defer browserPool.Put(browser)

	var statusCode int
	var page *rod.Page
	err = rod.Try(func() {
		page = stealth.MustPage(browser).Timeout(cfg.RenderTimeout)
		defer page.MustClose()
		e := proto.NetworkResponseReceived{}
		wait := page.WaitEvent(&e)
		page.MustNavigate(url)
		wait()
		statusCode = e.Response.Status
		page.MustWaitLoad()
		page.MustWaitIdle()
		takeScreenshot(page)
	})

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			slog.Error(fmt.Sprintf("dedline exceeded: %s", url), "error", err.Error())
			c.JSON(http.StatusBadRequest, newResponse(url, RenderStatusFailed, 0, tooLongUploadDeadlineError.Error()))
			return
		}
		slog.Error(fmt.Sprintf("failed to render url: %s", url), "error", err.Error())
		c.JSON(http.StatusBadRequest, newResponse(url, RenderStatusFailed, 0, somethingWentWrongError.Error()))
		return
	}

	c.JSON(http.StatusOK, newResponse(url, RenderStatusSuccess, statusCode, ""))
}

func newResponse(url string, rendered RenderStatus, statusCode int, err string) *Response {
	return &Response{
		URL:        url,
		Rendering:  rendered,
		StatusCode: statusCode,
		Error:      err,
	}
}

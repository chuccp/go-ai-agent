//go:build web

package main

import (
	wf "github.com/chuccp/go-web-frame"
	"github.com/chuccp/go-web-frame/core"
)

// DesktopInitService is a no-op in web mode.
type DesktopInitService struct{}

func (s *DesktopInitService) Init(ctx *core.Context) error { return nil }

func runDesktop(_ *wf.WebFrame) {}

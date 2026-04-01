package domain

import "errors"

var (
	ErrPluginNotFound     = errors.New("plugin not found")
	ErrPluginTimeout      = errors.New("plugin timeout")
	ErrPluginCrashed      = errors.New("plugin crashed")
	ErrMethodNotFound     = errors.New("method not found")
	ErrBrowserUnavailable = errors.New("browser unavailable")
	ErrBrowserTimeout     = errors.New("browser launch timeout")
	ErrQueueFull          = errors.New("queue full")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInvalidParams      = errors.New("invalid parameters")
	ErrSystemOverloaded   = errors.New("system overloaded")
	ErrNotFound           = errors.New("not found")
)

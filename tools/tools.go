//go:build tools

// Package tools ensures tool dependencies are kept in sync. This is the recommended way of doing this
// according to https://go.dev/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module.
// To install the following tools at the version used by this repo run:
// $ make bootstrap
//
// Versions are pinned to match the Makefile (SWAG_VERSION / GOMODIFYTAGS_VERSION /
// MOCKERY_VERSION), which is the single source of truth; keep these in sync.
package tools

//go:generate go install github.com/swaggo/swag/cmd/swag@v1.16.4
//go:generate go install github.com/fatih/gomodifytags@v1.17.0
//go:generate go install github.com/vektra/mockery/v2@v2.53.6

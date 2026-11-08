package app

import (
	"fmt"
	"runtime"
	"strings"
	"unicode"
)

// GuideOptions apply only to generated instructions, never to saved records.
type GuideOptions struct {
	LocalOS, RemoteOS, WindowsAccount, IdentityFile string
}

func LocalPlatform() string {
	if runtime.GOOS == "darwin" {
		return "macos"
	}
	return runtime.GOOS
}

func (options GuideOptions) Validate() error {
	for _, platform := range []string{options.LocalOS, options.RemoteOS} {
		if platform != "" && platform != "linux" && platform != "macos" && platform != "windows" {
			return fmt.Errorf("%w: platform must be linux, macos or windows", ErrInvalidInput)
		}
	}
	if options.WindowsAccount != "" && options.WindowsAccount != "standard" && options.WindowsAccount != "administrator" {
		return fmt.Errorf("%w: Windows account must be standard or administrator", ErrInvalidInput)
	}
	if options.RemoteOS != "" && options.RemoteOS != "windows" && options.WindowsAccount != "" {
		return fmt.Errorf("%w: Windows account applies only to a Windows destination", ErrInvalidInput)
	}
	if options.IdentityFile != "" {
		platform := options.LocalOS
		if platform == "" {
			platform = LocalPlatform()
		}
		path := options.IdentityFile
		for _, character := range path {
			if unicode.IsControl(character) {
				return fmt.Errorf("%w: identity-file contains control characters", ErrInvalidInput)
			}
		}
		absolute := strings.HasPrefix(path, "/")
		if platform == "windows" {
			path = strings.ReplaceAll(path, "\\", "/")
			drive := len(path) >= 3 && ((path[0] >= 'a' && path[0] <= 'z') || (path[0] >= 'A' && path[0] <= 'Z')) && path[1:3] == ":/"
			absolute = drive || strings.HasPrefix(path, "//")
		}
		if !absolute && !strings.HasPrefix(path, "~/") {
			return fmt.Errorf("%w: identity-file must be an absolute %s path or ~/path", ErrInvalidInput, platform)
		}
	}
	return nil
}

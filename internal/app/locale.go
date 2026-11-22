package app

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func systemLanguage() string {
	if runtime.GOOS == "darwin" {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if out, err := exec.CommandContext(ctx, "defaults", "read", "-g", "AppleLanguages").Output(); err == nil {
			if lang := firstAppleLanguage(string(out)); lang != "" {
				return lang
			}
		}
	}
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(key); v != "" && v != "C" && v != "POSIX" {
			return languageCode(v)
		}
	}
	return "en"
}

func firstAppleLanguage(plist string) string {
	for field := range strings.FieldsFuncSeq(plist, func(r rune) bool {
		return strings.ContainsRune("(),\"\n\t ", r)
	}) {
		return languageCode(field)
	}
	return ""
}

func languageCode(tag string) string {
	tag, _, _ = strings.Cut(tag, ".")
	code, _, _ := strings.Cut(strings.ReplaceAll(tag, "_", "-"), "-")
	return strings.ToLower(code)
}

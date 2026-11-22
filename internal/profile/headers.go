package profile

import (
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

type Meta struct {
	Info                Info
	Title               string
	UpdateIntervalHours int
}

func ParseHeaders(h http.Header) Meta {
	m := Meta{Info: parseUserInfo(h.Get("Subscription-Userinfo"))}
	m.Info.SupportURL = strings.TrimSpace(h.Get("Support-Url"))
	m.Info.WebPageURL = strings.TrimSpace(h.Get("Profile-Web-Page-Url"))
	m.Info.Announce = decodeMaybeBase64(h.Get("Announce"))
	m.Title = decodeMaybeBase64(h.Get("Profile-Title"))
	if m.Title == "" {
		m.Title = filenameTitle(h.Get("Content-Disposition"))
	}
	if hours, err := strconv.Atoi(strings.TrimSpace(h.Get("Profile-Update-Interval"))); err == nil && hours > 0 {
		m.UpdateIntervalHours = hours
	}
	return m
}

func parseUserInfo(value string) Info {
	var info Info
	for _, part := range strings.Split(value, ";") {
		key, raw, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		n, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "upload":
			info.Upload = int64(n)
		case "download":
			info.Download = int64(n)
		case "total":
			info.Total = int64(n)
		case "expire":
			info.Expire = int64(n)
		}
	}
	return info
}

func decodeMaybeBase64(value string) string {
	value = strings.TrimSpace(value)
	encoded, ok := strings.CutPrefix(value, "base64:")
	if !ok {
		return value
	}
	decoded, _ := decodeBase64(encoded)
	return strings.TrimSpace(string(decoded))
}

func filenameTitle(disposition string) string {
	_, params, err := mime.ParseMediaType(disposition)
	if err != nil {
		return ""
	}
	name := params["filename"]
	return strings.TrimSuffix(name, filepath.Ext(name))
}

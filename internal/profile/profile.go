package profile

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type Info struct {
	Upload   int64 `json:"upload"`
	Download int64 `json:"download"`

	Total int64 `json:"total"`

	Expire     int64  `json:"expire"`
	SupportURL string `json:"supportUrl,omitempty"`
	WebPageURL string `json:"webPageUrl,omitempty"`
	Announce   string `json:"announce,omitempty"`
}

type Profile struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	UserAgent string `json:"userAgent"`

	UpdateIntervalHours int               `json:"updateIntervalHours"`
	UpdatedAt           time.Time         `json:"updatedAt"`
	Info                Info              `json:"info"`
	Selected            map[string]string `json:"selected,omitempty"`
}

func IsLink(text string) bool {
	u, err := url.Parse(text)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" &&
		!strings.ContainsAny(text, " \t\r\n")
}

func (p Profile) Host() string {
	u, err := url.Parse(p.URL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func (p Profile) Due(now time.Time) bool {
	return p.UpdateIntervalHours > 0 &&
		now.Sub(p.UpdatedAt) >= time.Duration(p.UpdateIntervalHours)*time.Hour
}

func NewID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("profile: generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

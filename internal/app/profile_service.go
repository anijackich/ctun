package app

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"ctun/internal/profile"
)

const autoUpdateTick = 10 * time.Minute

type ProfileView struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Host                string `json:"host"`
	UserAgent           string `json:"userAgent"`
	UpdateIntervalHours int    `json:"updateIntervalHours"`
	UpdatedAt           int64  `json:"updatedAt"`
	Active              bool   `json:"active"`

	Local bool         `json:"local"`
	Info  profile.Info `json:"info"`
}

type AddProfileRequest struct {
	URL                 string `json:"url"`
	Name                string `json:"name"`
	UserAgent           string `json:"userAgent"`
	UpdateIntervalHours int    `json:"updateIntervalHours"`
}

type EditProfileRequest struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	URL                 string `json:"url"`
	UserAgent           string `json:"userAgent"`
	UpdateIntervalHours int    `json:"updateIntervalHours"`
}

type ProfileService struct {
	store   *profile.Store
	core    *CoreService
	fetcher *profile.Fetcher

	refreshMu sync.Mutex
	stopLoop  context.CancelFunc

	delaysMu sync.Mutex
	delays   map[string]map[string]int
}

func NewProfileService(store *profile.Store, core *CoreService) *ProfileService {
	s := &ProfileService{
		store:   store,
		core:    core,
		fetcher: &profile.Fetcher{ProxyURL: core.ProxyURL},
		delays:  map[string]map[string]int{},
	}
	core.onConnected = s.testActive
	return s
}

func (s *ProfileService) ServiceStartup(context.Context, application.ServiceOptions) error {
	ctx, cancel := context.WithCancel(context.Background())
	s.stopLoop = cancel
	go s.autoUpdate(ctx)
	return nil
}

func (s *ProfileService) ServiceShutdown() error {
	if s.stopLoop != nil {
		s.stopLoop()
	}
	return nil
}

func (s *ProfileService) List() []ProfileView {
	active := s.store.ActiveID()
	profiles := s.store.List()
	views := make([]ProfileView, len(profiles))
	for i, p := range profiles {
		views[i] = ProfileView{
			ID:                  p.ID,
			Name:                p.Name,
			Host:                p.Host(),
			UserAgent:           p.UserAgent,
			UpdateIntervalHours: p.UpdateIntervalHours,
			UpdatedAt:           p.UpdatedAt.UnixMilli(),
			Active:              p.ID == active,
			Local:               p.URL == "",
			Info:                p.Info,
		}
	}
	return views
}

func (s *ProfileService) Add(req AddProfileRequest) (ProfileView, error) {
	link := strings.TrimSpace(req.URL)
	if !profile.IsLink(link) {
		return ProfileView{}, errInvalidLink
	}
	s.refreshMu.Lock()
	p, body, err := s.fetcher.Download(profile.Profile{
		URL:                 link,
		UserAgent:           cmp.Or(strings.TrimSpace(req.UserAgent), profile.DefaultUserAgent),
		UpdateIntervalHours: req.UpdateIntervalHours,
	})
	s.refreshMu.Unlock()
	if err != nil {
		return ProfileView{}, err
	}
	p.Name = cmp.Or(strings.TrimSpace(req.Name), p.Name)
	if err := s.store.Put(p, body); err != nil {
		return ProfileView{}, err
	}
	slog.Info("subscription added", "name", p.Name)
	s.changed()
	return s.view(p.ID)
}

func (s *ProfileService) Import(text string) (ProfileView, error) {
	text = strings.TrimSpace(text)
	if profile.IsLink(text) {
		return s.Add(AddProfileRequest{URL: text})
	}
	return s.addConfig([]byte(text), "", "")
}

func (s *ProfileService) ImportFile() (*ProfileView, error) {
	app := application.Get()
	path, err := app.Dialog.OpenFile().
		AttachToWindow(app.Window.Current()).
		CanChooseFiles(true).
		PromptForSingleSelection()
	if err != nil {
		return nil, fmt.Errorf("open file dialog: %w", err)
	}
	if path == "" {
		return nil, nil
	}
	content, err := profile.ReadConfig(path)
	if err != nil {
		return nil, err
	}
	view, err := s.addConfig(content, "", strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	if err != nil {
		return nil, err
	}
	return &view, nil
}

func (s *ProfileService) addConfig(content []byte, name, fallback string) (ProfileView, error) {
	content = []byte(strings.TrimSpace(string(content)))
	layer, err := profile.Parse(content)
	if err != nil {
		return ProfileView{}, err
	}
	id, err := profile.NewID()
	if err != nil {
		return ProfileView{}, err
	}
	if name == "" && len(layer.Proxies) == 1 {
		name, _ = layer.Proxies[0]["name"].(string)
	}
	p := profile.Profile{ID: id, Name: cmp.Or(name, fallback, "Config"), UpdateIntervalHours: -1, UpdatedAt: time.Now()}
	if err := s.store.Put(p, content); err != nil {
		return ProfileView{}, err
	}
	slog.Info("config imported", "name", p.Name, "servers", len(layer.Proxies))
	s.changed()
	return s.view(p.ID)
}

func (s *ProfileService) Refresh(id string) error {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()

	current, err := s.store.Get(id)
	if err != nil {
		return err
	}
	if current.URL == "" {
		return errLocalConfig
	}
	p, body, err := s.fetcher.Download(current)
	if err != nil {
		return err
	}
	p.Name = current.Name
	if err := s.store.Put(p, body); err != nil {
		return err
	}
	slog.Info("subscription updated", "name", p.Name)
	s.changed()
	if id == s.store.ActiveID() {
		return s.core.reload()
	}
	return nil
}

func (s *ProfileService) Edit(req EditProfileRequest) (ProfileView, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return ProfileView{}, errNameNeeded
	}
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()

	current, err := s.store.Get(req.ID)
	if err != nil {
		return ProfileView{}, err
	}
	next := current
	next.Name = name
	if current.URL != "" {
		next.UserAgent = cmp.Or(strings.TrimSpace(req.UserAgent), profile.DefaultUserAgent)
		next.UpdateIntervalHours = req.UpdateIntervalHours
		if link := strings.TrimSpace(req.URL); link != "" {
			if !profile.IsLink(link) {
				return ProfileView{}, errInvalidLink
			}
			next.URL = link
		}
	}

	refetch := current.URL != "" &&
		(next.URL != current.URL || next.UserAgent != cmp.Or(current.UserAgent, profile.DefaultUserAgent))
	if refetch {
		downloaded, body, err := s.fetcher.Download(next)
		if err != nil {
			return ProfileView{}, err
		}
		downloaded.Name = name
		if err := s.store.Put(downloaded, body); err != nil {
			return ProfileView{}, err
		}
	} else if _, err := s.store.Update(req.ID, func(p *profile.Profile) { *p = next }); err != nil {
		return ProfileView{}, err
	}
	slog.Info("subscription edited", "name", name)
	s.changed()
	if refetch && req.ID == s.store.ActiveID() {
		if err := s.core.reload(); err != nil {
			return ProfileView{}, err
		}
	}
	return s.view(req.ID)
}

func (s *ProfileService) Remove(id string) error {
	wasActive := id == s.store.ActiveID()
	if err := s.store.Remove(id); err != nil {
		return err
	}
	s.changed()
	if !wasActive {
		return nil
	}
	if s.store.ActiveID() == "" {
		return s.core.Stop()
	}
	return s.core.reload()
}

func (s *ProfileService) SubscriptionURL(id string) (string, error) {
	p, err := s.store.Get(id)
	if err == nil && p.URL == "" {
		err = errLocalConfig
	}
	return p.URL, err
}

func (s *ProfileService) Activate(id string) error {
	if id == s.store.ActiveID() {
		return nil
	}
	if err := s.store.SetActive(id); err != nil {
		return err
	}
	if p, err := s.store.Get(id); err == nil {
		slog.Info("subscription activated", "name", p.Name)
	}
	s.changed()
	if err := s.core.reload(); err != nil {
		return err
	}
	go s.testActive()
	return nil
}

func (s *ProfileService) autoUpdate(ctx context.Context) {
	ticker := time.NewTicker(autoUpdateTick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		now := time.Now()
		for _, p := range s.store.List() {
			if !p.Due(now) {
				continue
			}
			if err := s.Refresh(p.ID); err != nil {
				slog.Warn("auto-update subscription", "name", p.Name, "err", err)
			}
		}
	}
}

func (s *ProfileService) view(id string) (ProfileView, error) {
	for _, v := range s.List() {
		if v.ID == id {
			return v, nil
		}
	}
	return ProfileView{}, profile.ErrNotFound
}

func (s *ProfileService) changed() {
	emit(EventProfiles, s.List())
}

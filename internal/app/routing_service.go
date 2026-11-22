package app

import (
	"log/slog"

	"ctun/internal/profile"
	"ctun/internal/routing"
)

type RoutingProfileView struct {
	Profile routing.Profile      `json:"profile"`
	Active  bool                 `json:"active"`
	Errors  []routing.EntryError `json:"errors"`
}

type RoutingService struct {
	core     *CoreService
	profiles *profile.Store
	routes   *routing.Store
}

func NewRoutingService(core *CoreService, profiles *profile.Store, routes *routing.Store) *RoutingService {
	return &RoutingService{core: core, profiles: profiles, routes: routes}
}

func (s *RoutingService) Profiles() []RoutingProfileView {
	active := s.routes.ActiveID()
	list := s.routes.List()
	views := make([]RoutingProfileView, len(list))
	for i, p := range list {
		views[i] = RoutingProfileView{Profile: p, Active: p.ID == active, Errors: p.Errors()}
	}
	return views
}

func (s *RoutingService) SaveProfile(p routing.Profile) (RoutingProfileView, error) {
	saved, err := s.routes.Save(p)
	if err != nil {
		return RoutingProfileView{}, err
	}
	active := saved.ID == s.routes.ActiveID()
	s.changed(active)
	return RoutingProfileView{Profile: saved, Active: active, Errors: saved.Errors()}, nil
}

func (s *RoutingService) CreateProfile(name, from string) (RoutingProfileView, error) {
	p, err := s.routes.Create(name, from)
	if err != nil {
		return RoutingProfileView{}, err
	}
	slog.Info("routing profile created", "name", p.Name)
	s.changed(false)
	return RoutingProfileView{Profile: p, Errors: p.Errors()}, nil
}

func (s *RoutingService) DeleteProfile(id string) error {
	wasActive := id == s.routes.ActiveID()
	if err := s.routes.Remove(id); err != nil {
		return err
	}
	s.changed(wasActive)
	return nil
}

func (s *RoutingService) ActivateProfile(id string) error {
	if id == s.routes.ActiveID() {
		return nil
	}
	if err := s.routes.SetActive(id); err != nil {
		return err
	}
	slog.Info("routing profile activated", "name", s.routes.Active().Name)
	s.changed(true)
	return nil
}

func (s *RoutingService) changed(reload bool) {
	emit(EventRouting, s.Profiles())
	if !reload {
		return
	}
	s.core.syncSystemSettings()
	if err := s.core.reload(); err != nil {
		slog.Warn("apply routing profile", "err", err)
	}
}

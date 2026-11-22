package app

import (
	"context"
	"log/slog"
	"time"

	"ctun/internal/core"
	"ctun/internal/privilege"
)

const privilegeTimeout = 2 * time.Minute

type HelperStatus struct {
	State privilege.State `json:"state"`
}

type PrivilegeService struct {
	core *CoreService
}

func NewPrivilegeService(core *CoreService) *PrivilegeService {
	return &PrivilegeService{core: core}
}

func (s *PrivilegeService) Status() HelperStatus {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return HelperStatus{State: privilege.Check(ctx)}
}

func (s *PrivilegeService) Install() error {
	helperBinary, err := privilege.FindHelper()
	if err != nil {
		return err
	}
	coreBinary, err := core.FindBinary()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), privilegeTimeout)
	defer cancel()
	if err := privilege.Install(ctx, helperBinary, coreBinary); err != nil {
		return err
	}
	slog.Info("TUN helper installed")
	return nil
}

func (s *PrivilegeService) Uninstall() error {
	if s.core.tunRunning() {
		if err := s.core.Stop(); err != nil {
			return err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), privilegeTimeout)
	defer cancel()
	return privilege.Uninstall(ctx)
}

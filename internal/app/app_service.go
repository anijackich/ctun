package app

type AppService struct {
	version string
}

func NewAppService(version string) *AppService {
	return &AppService{version: version}
}

func (s *AppService) Version() string {
	return s.version
}

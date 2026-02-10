package healthcheck

// Service handles health check business logic
type Service struct{}

// NewService creates a new health check service
func NewService() *Service {
	return &Service{}
}

// CheckHealth performs health check and returns status
func (s *Service) CheckHealth() string {
	return "ok"
}

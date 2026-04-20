package healthcheck

// Service defines the business logic for health checks
type Service interface {
	CheckHealth() string
}

type service struct{}

// NewService creates a new health check service
func NewService() Service {
	return &service{}
}

// CheckHealth performs health check and returns status
func (s *service) CheckHealth() string {
	return "ok"
}

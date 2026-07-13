package health

// Status is the response body for GET /health.
type Status struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type Checker struct {
	version string
}

// NewChecker constructs a Checker that reports the server health.
func NewChecker(version string) *Checker {
	return &Checker{version: version}
}

func (s *Checker) Check() Status {
	return Status{
		Status:  "ok",
		Version: s.version,
	}
}

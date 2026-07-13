package health

// Status is the response body for GET /health.
type Status struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

type Checker struct {
	status Status
}

// NewChecker constructs a Checker that reports the server health.
func NewChecker(version, commit, date string) *Checker {
	return &Checker{status: Status{
		Status:  "ok",
		Version: version,
		Commit:  commit,
		Date:    date,
	}}
}

func (s *Checker) Check() Status {
	return s.status
}

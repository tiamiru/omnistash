package namespace

import (
	"fmt"
	"regexp"
)

// namePattern mirrors the OCI distribution spec's repository name grammar.
var namePattern = regexp.MustCompile(`^[a-z0-9]+((\.|_|__|-+)[a-z0-9]+)*(\/[a-z0-9]+((\.|_|__|-+)[a-z0-9]+)*)*$`)

// ValidateName reports whether name matches the OCI repository name grammar.
func ValidateName(name string) error {
	if name == "" || !namePattern.MatchString(name) {
		return fmt.Errorf("%w: %s", ErrNameInvalid, name)
	}

	return nil
}

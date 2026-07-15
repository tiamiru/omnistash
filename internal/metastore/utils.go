package metastore

import "time"

func UnixToTime(unix int64) time.Time {
	return time.Unix(unix, 0).UTC()
}

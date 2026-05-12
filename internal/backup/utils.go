package backup

import (
	"fmt"
	"time"
)

func humanSize(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.2f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.2f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.2f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func GenerateFilename(db, schema string, timescale bool) string {
	ts := time.Now().Format("20060102_150405")
	prefix := db
	if schema != "" {
		prefix = fmt.Sprintf("%s_%s", db, schema)
	}
	if timescale {
		prefix = prefix + "_timescale"
	}
	return fmt.Sprintf("%s_%s.dump", prefix, ts)
}

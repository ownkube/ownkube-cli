package sourcearchive

import "fmt"

// CheckSize refuses an archive over the upload slot's limit (0 = no limit).
func CheckSize(size, limit int64) error {
	if limit > 0 && size > limit {
		return fmt.Errorf("source is %s; the limit is %s. Add build output, dependencies, and other large files to .gitignore, then retry",
			HumanBytes(size), HumanBytes(limit))
	}
	return nil
}

// HumanBytes formats a byte count for upload progress lines.
func HumanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

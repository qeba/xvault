package metrics

import "syscall"

// StatFs represents filesystem statistics
type StatFs struct {
	Bsize  int64
	Blocks uint64
	Bfree  uint64
	Bavail uint64
}

// Statfs wraps the syscall.Statfs function
func Statfs(path string, stat *StatFs) error {
	var sysstat syscall.Statfs_t
	if err := syscall.Statfs(path, &sysstat); err != nil {
		return err
	}

	stat.Bsize = sysstat.Bsize
	stat.Blocks = sysstat.Blocks
	stat.Bfree = sysstat.Bfree
	stat.Bavail = sysstat.Bavail

	return nil
}

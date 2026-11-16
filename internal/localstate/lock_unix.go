//go:build unix

package localstate

import (
	"golang.org/x/sys/unix"
	"os"
)

func (s *Store) diskLock() (*os.File, error) {
	f, err := os.OpenFile(s.file+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = unix.Flock(int(f.Fd()), unix.LOCK_EX); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}
func unlock(f *os.File) { _ = unix.Flock(int(f.Fd()), unix.LOCK_UN); _ = f.Close() }

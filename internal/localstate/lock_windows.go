package localstate

import (
	"golang.org/x/sys/windows"
	"os"
)

func (s *Store) diskLock() (*os.File, error) {
	f, err := os.OpenFile(s.file+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	var overlap windows.Overlapped
	if err = windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, &overlap); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}
func unlock(f *os.File) {
	var overlap windows.Overlapped
	_ = windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, &overlap)
	_ = f.Close()
}

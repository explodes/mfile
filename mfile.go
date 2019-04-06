package mfile

import (
	"io"
	"os"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"

	"github.com/hashicorp/go-multierror"
)

const (
	nilPtr = uintptr(0)
)

// File is a file that is mapped to memory.
type File struct {
	filename string
	fd       int
	buf      []byte
	size     int
	data     uintptr
}

// Open maps a random-access file to memory with the given size.
func Open(filename string, size int) (*File, error) {
	m := &File{
		filename: filename,
	}
	if err := m.ResizeMinimum(size); err != nil {
		err = labelErr(err, "could not resize file")
		return nil, closeWithErr(err, m)
	}
	return m, nil
}

func (m *File) reopenFile() error {
	if err := m.Close(); err != nil {
		return labelErr(err, "could not close open file")
	}
	fd, err := syscall.Open(m.filename, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return labelErr(err, "could not open file")
	}
	m.fd = fd
	return nil
}

func (m *File) mmap(size int) error {
	buf, err := syscall.Mmap(m.fd, 0, size, syscall.PROT_WRITE|syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return labelErr(err, "could not mmap file descriptor")
	}
	if err := syscall.Madvise(buf, syscall.MADV_RANDOM); err != nil {
		err = labelErr(err, "could not perform SYS_MADVISE")
		return closeWithErr(err, m)
	}
	m.buf = buf
	m.size = len(buf)
	m.data = uintptr(unsafe.Pointer(&buf[0]))
	return nil
}

func (m *File) truncate(size int) error {
	err := syscall.Ftruncate(m.fd, int64(size))
	return labelErr(err, "could not perform SYS_FTRUNCATE")
}

// Resize truncates the file to given size in bytes.
func (m *File) Resize(size int) error {
	if err := m.reopenFile(); err != nil {
		return labelErr(err, "could not reopen file")
	}
	if err := m.truncate(size); err != nil {
		return closeWithErr(labelErr(err, "could not truncate file"), m)
	}
	if err := m.mmap(size); err != nil {
		return closeWithErr(labelErr(err, "could not mmap file"), m)
	}
	return nil
}

// ResizeMinimum truncates the file to a minimum size in bytes.
func (m *File) ResizeMinimum(size int) error {
	if err := m.reopenFile(); err != nil {
		return labelErr(err, "could not reopen file")
	}
	var stat syscall.Stat_t
	err := syscall.Stat(m.filename, &stat)
	if err != nil {
		return closeWithErr(labelErr(err, "could not stat file"), m)
	}
	if int(stat.Size) >= size {
		// No need to truncate, skip it.
		goto performMmap
	}
	if err := m.truncate(size); err != nil {
		return closeWithErr(labelErr(err, "could not truncate file"), m)
	}
performMmap:
	if err := m.mmap(size); err != nil {
		return closeWithErr(labelErr(err, "could not mmap file"), m)
	}
	return nil
}

// Len returns the length of the underlying data.
func (m *File) Len() int {
	return m.size
}

// DataPtr returns the pointer to the beginning of the mapped data.
func (m *File) DataPtr() unsafe.Pointer {
	return unsafe.Pointer(m.data)
}

// DataPtr returns the pointer to the data at an offset.
func (m *File) DataAt(offset uintptr) unsafe.Pointer {
	return unsafe.Pointer(m.data + offset)
}

// BytesAt returns the byte slice specified.
func (m *File) BytesAt(offset uintptr, size int) []byte {
	return m.buf[offset : offset+uintptr(size)]
}

// Sync syncs the file's contents to disk.
func (m *File) Sync() error {
	err := syscall.Fsync(m.fd)
	return labelErr(err, "error syncing file to disk")
}

// SyncRange syncs a range of the file's contents to disk.
func (m *File) SyncRange(offset, length int64) error {
	err := syscall.SyncFileRange(m.fd, offset, length, 0)
	return labelErr(err, "error syncing range to disk")
}

// Close closes the file.
func (m *File) Close() error {
	var err error
	if m.fd != 0 {
		err = mergeErr(err, labelErr(syscall.Close(m.fd), "could not close file"))
		m.fd = 0
	}
	if m.buf != nil {
		err = mergeErr(err, labelErr(syscall.Munmap(m.buf), "could not Munmap file"))
		m.buf = nil
	}
	m.size = 0
	m.data = nilPtr
	return err
}

func closeWithErr(err error, closer io.Closer) error {
	return mergeErr(err, labelErr(closer.Close(), "could not close"))
}

func mergeErr(e1, e2 error) error {
	switch {
	case e1 != nil && e2 == nil:
		// Common case, e1 not nil, e2 nil.
		return e1
	case e1 == nil && e2 == nil:
		return nil
	case e1 != nil && e2 != nil:
		// Both errors are non-nil.
		return multierror.Append(e1, e2)
	default:
		// Least likely case, e1 non-nil, e2 nil.
		return e2

	}
}

func labelErr(err error, label string) error {
	return errors.Wrap(err, label)
}

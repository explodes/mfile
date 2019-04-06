package mfile

import (
	"bytes"
	"encoding/hex"
	"math/rand"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mfileFixture struct {
	t        *testing.T
	filePath string
	mf       *File
}

func newFixture(t *testing.T, size int) *mfileFixture {
	nameBuf := make([]byte, 32)
	if _, err := rand.Read(nameBuf); err != nil {
		t.Fatal(err)
	}
	fname := "mfile_test_" + hex.EncodeToString(nameBuf)
	filePath := path.Join(os.TempDir(), fname)
	mf, err := Open(filePath, size)
	if err != nil {
		t.Fatal(err)
	}
	return &mfileFixture{
		t:        t,
		filePath: filePath,
		mf:       mf,
	}
}

func (f *mfileFixture) Stat() os.FileInfo {
	fi, err := os.Stat(f.filePath)
	if err != nil {
		f.t.Fatal(err)
	}
	return fi
}

func (f *mfileFixture) CloseOrError() {
	if err := f.mf.Close(); err != nil {
		f.t.Error(err)
	}
	if err := os.Remove(f.filePath); err != nil {
		f.t.Error(err)
	}
}

func TestOpen(t *testing.T) {
	fix := newFixture(t, 100)
	defer fix.CloseOrError()

	stat := fix.Stat()
	assert.Equal(t, int64(100), stat.Size())
}

func TestFile_Resize(t *testing.T) {
	fix := newFixture(t, 200)
	defer fix.CloseOrError()

	stat := fix.Stat()
	assert.Equal(t, int64(200), stat.Size())

	if err := fix.mf.Resize(300); err != nil {
		t.Fatal(err)
	}

	stat = fix.Stat()
	assert.Equal(t, int64(300), stat.Size())
}

func TestFile_Resize_SameSize(t *testing.T) {
	fix := newFixture(t, 200)
	defer fix.CloseOrError()

	stat := fix.Stat()
	assert.Equal(t, int64(200), stat.Size())

	if err := fix.mf.Resize(200); err != nil {
		t.Fatal(err)
	}

	stat = fix.Stat()
	assert.Equal(t, int64(200), stat.Size())
}

func TestFile_ResizeMinimum(t *testing.T) {
	fix := newFixture(t, 200)
	defer fix.CloseOrError()

	stat := fix.Stat()
	assert.Equal(t, int64(200), stat.Size())

	if err := fix.mf.ResizeMinimum(100); err != nil {
		t.Fatal(err)
	}

	stat = fix.Stat()
	assert.Equal(t, int64(200), stat.Size())

	if err := fix.mf.ResizeMinimum(300); err != nil {
		t.Fatal(err)
	}

	stat = fix.Stat()
	assert.Equal(t, int64(300), stat.Size())
}

func TestFile_Len(t *testing.T) {
	fix := newFixture(t, 100)
	defer fix.CloseOrError()

	assert.Equal(t, 100, fix.mf.Len())
}

func TestFile_DataPtr(t *testing.T) {
	fix := newFixture(t, 100)
	defer fix.CloseOrError()

	assert.True(t, uintptr(fix.mf.DataPtr()) > uintptr(0))
}

func TestFile_DataAt(t *testing.T) {
	fix := newFixture(t, 100)
	defer fix.CloseOrError()

	assert.True(t, uintptr(fix.mf.DataAt(10)) > uintptr(10))
}

func TestFile_Sync(t *testing.T) {
	fix := newFixture(t, 100)
	defer fix.CloseOrError()

	assert.NoError(t, fix.mf.Sync())
}

func TestFile_SyncRange(t *testing.T) {
	fix := newFixture(t, 100)
	defer fix.CloseOrError()

	assert.NoError(t, fix.mf.SyncRange(0, 10))
}

func TestFile_BytesAt(t *testing.T) {
	const alpha = "abcdefghijklmnopqrtuvwyz"
	fix := newFixture(t, 100)
	defer fix.CloseOrError()

	copy(fix.mf.buf, []byte(alpha))
	assert.NoError(t, fix.mf.Sync())

	assert.True(t, bytes.Equal([]byte(alpha), fix.mf.BytesAt(0, len(alpha))))
}

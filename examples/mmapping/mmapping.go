package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"unsafe"

	"github.com/explodes/scratch/mmapping/mfile"
)

const actualOutput = `
2018/12/03 20:03:34 DELETE FILE
2018/12/03 20:03:34 could not delete file: remove test.dat: no such file or directory
2018/12/03 20:03:34 WRITE FILE
2018/12/03 20:03:34 buf: 0x7f899c9b0000
2018/12/03 20:03:34 m.buf: 0x7f899c9b0000
2018/12/03 20:03:34 The m.buf: 0x7f899c9b0000
2018/12/03 20:03:34 Size: 8
2018/12/03 20:03:34 Initial Contents:
2018/12/03 20:03:34 Mutating struct at: 0xc42001e1d8
2018/12/03 20:03:34 File buf at: 0x7f899c9b0000
2018/12/03 20:03:34 New Contents:
2018/12/03 20:03:34 READ FILE
2018/12/03 20:03:34 Contents:
2018/12/03 20:03:34 DELETE FILE
2018/12/03 20:03:34 File deleted.
`

var (
	// filenameFlag is the name of the file to test mmapp on.
	filenameFlag = flag.String("filename", "test.dat", "filename to open for writing")
)

func main() {
	flag.Parse()

	deleteFile()
	writeFile()
	readFile()
	//deleteFile()
}

// deleteFile deletes the file from disk.
func deleteFile() {
	log.Println("DELETE FILE")
	err := os.Remove(*filenameFlag)
	if !maybeLogErr(err, "could not delete file") {
		log.Println("File deleted.")
	}
}

// writeFile opens the file as a memory mapped file, modifies the contents, and exits.
func writeFile() {
	log.Println("WRITE FILE")

	type obj struct {
		a [12]byte
		b int64
		d float64
		c complex128
		e rune
		f rune
		g struct {
			x byte
			y byte
		}
	}

	m, err := mfile.Open(*filenameFlag, 1e6)
	if err != nil {
		exit(1, "could not open file for write: %v", err)
	}
	if err := m.Resize(int(unsafe.Sizeof(obj{}))); err != nil {
		exit(2, "could not resize mem file: %v", err)
	}
	defer func() {
		maybeLogErr(m.Close(), "could not close writable file")
	}()

	thing := (*obj)(m.DataPtr())
	copy(thing.a[:], []byte("hello, world!"))
	thing.b = 0x6c6f6c21 // lol!
	thing.c = 0x6c6f6c21 + 7i
	thing.d = 3.14159265
	thing.e = 'a'
	thing.f = '力'
	thing.g.x = 31
	thing.g.y = 32

	maybeLogErr(m.Sync(), "could not sync contents")
}

// readFile reads the contents of the file from disk and logs the contents.
func readFile() {
	log.Println("READ FILE")

	file, err := os.Open(*filenameFlag)
	if err != nil {
		exit(1, "could not open file for reading: %v", err)
	}

	b, err := ioutil.ReadAll(file)
	if err != nil {
		exit(2, "could not read file: %v", err)
	}

	log.Printf("Contents: %s", string(b))
}

// maybeLogErr logs an error if it is not nil and returns if there was an error to log.
func maybeLogErr(err error, cause string) bool {
	if err == nil {
		return false
	}
	log.Printf("%s: %v", cause, err)
	return true
}

// exit logs a message and exits the program.
func exit(code int, msg string, args ...interface{}) {
	log.Printf(msg, args...)
	os.Exit(code)
}

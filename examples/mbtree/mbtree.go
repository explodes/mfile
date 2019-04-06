package main

import (
	"flag"
	"log"
	"unsafe"

	"github.com/explodes/scratch/mmapping/mfile"
)

const (
	magic   = 0xbbcde160
	version = 1
)

var (
	// filenameFlag is the name of the file to test mmapp on.
	filenameFlag = flag.String("filename", "test.dat", "filename to open for writing")
)

var (
	nodeSize   = unsafe.Sizeof(Node{})
	headerSize = unsafe.Sizeof(Header{})
)

type Header struct {
	magic   uint32
	version uint16
	length  uintptr
	root    uintptr
}

type Node struct{}

func main() {

	flag.Parse()

	m, err := mfile.Open(*filenameFlag, int(headerSize))
	exitIfErr(err)

	header := (*Header)(m.DataPtr())
	if header.magic == 0 {
		header.magic = magic
		header.version = version
		header.length = headerSize
	} else if header.magic != magic {
		panic("magic number invalid")
	} else if header.version != version {
		panic("unsupported version")
	}

}

// maybeLogErr logs an error if it is not nil and returns if there was an error to log.
func maybeLogErr(err error, cause string) bool {
	if err == nil {
		return false
	}
	log.Printf("%s: %v", cause, err)
	return true
}

func exitIfErr(err error) {
	if err != nil {
		panic(err)
	}
}

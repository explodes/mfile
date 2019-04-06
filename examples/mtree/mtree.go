package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"unsafe"

	"github.com/explodes/cli"

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

type Node struct {
	value       rune
	left, right uintptr
}

type App struct {
	filename string
	header   *Header
	m        *mfile.File
}

func (a *App) Close() error {
	if a.m != nil {
		err := a.m.Close()
		a.m = nil
		a.header = nil
		return err
	}
	return nil
}

func (a *App) open() {
	log.Println("OPEN FILE")

	if err := a.Close(); err != nil {
		log.Printf("error when closing: %v", err)
	}

	m, err := mfile.Open(a.filename, int(headerSize))
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

	a.m = m
	a.header = header
}

func (a *App) loop() {
loop:
	for {
		selection := cli.PresentMenu(
			"?) ",
			"Delete file",
			"Init",
			"Add node",
			"Update node",
			"Print",
			"Print Orphans",
			"Exit")
		switch selection {
		case 0:
			a.deleteFile()
		case 1:
			a.open()
		case 2:
			a.addNode()
		case 3:
			a.updateNode()
		case 4:
			a.printTree()
		case 5:
			a.detectOrphans()
		case 6:
			break loop
		default:
			log.Println("unknown option")
		}
	}
}

func (a *App) promptNodeValue(ps1 string) rune {
	for {
		nameStr := cli.PromptString(ps1)
		if len(nameStr) != 1 {
			fmt.Println("must be a single character")
			continue
		}
		return rune(nameStr[0]) // TODO: not a true rune
	}
}

func (a *App) addNode() {
	log.Println("ADD NODE")

	newName := a.promptNodeValue("New node name) ")

	nodes := a.collectNodeValues()
	menu := append([]string{"root"}, nodes...)
	parent := cli.PresentMenu("Select a parent) ", menu...)

	offset, err := a.putNode(Node{value: newName}, a.header.length)
	if err != nil {
		panic(err)
	}
	a.header.length += nodeSize

	if parent == 0 {
		a.header.root = offset
	} else {
		parentVal := rune(nodes[parent-1][0])
		isLeft := cli.PresentMenu("Left or right? ", "left", "right") == 0

		a.walk(func(node *Node) {
			if node.value != parentVal {
				return
			}
			if isLeft {
				node.left = offset
			} else {
				node.right = offset
			}
		})

	}

	if err := a.m.Sync(); err != nil {
		log.Printf("error with sync: %v", err)
	}
}

func (a *App) updateNode() {
	log.Println("UPDATE NODE")

	nodes := a.collectNodeValues()
	node := cli.PresentMenu("Select a node) ", nodes...)
	nodeVal := rune(nodes[node][0])

	newName := a.promptNodeValue("New node name) ")

	a.walk(func(node *Node) {
		if node.value != nodeVal {
			return
		}
		node.value = newName
	})

	if err := a.m.Sync(); err != nil {
		log.Printf("error with sync: %v", err)
	}
}

func (a *App) deleteFile() {
	log.Println("DELETE FILE")

	err := os.Remove(a.filename)
	if !maybeLogErr(err, "could not delete file") {
		log.Println("File deleted.")
	}
}

func (a *App) putNode(node Node, offset uintptr) (uintptr, error) {
	if err := a.m.Resize(int(offset + nodeSize)); err != nil {
		return 0, err
	}

	newNode := (*Node)(a.m.DataAt(offset))
	*newNode = node

	if err := a.m.Sync(); err != nil {
		return 0, err
	}

	return offset, nil
}

func (a *App) getNode(offset uintptr) *Node {
	return (*Node)(a.m.DataAt(offset))
}

func (a *App) detectOrphans() {
	base := uintptr(unsafe.Pointer(a.header))
	nodes := map[uintptr]struct{}{}
	a.walk(func(node *Node) {
		ptr := uintptr(unsafe.Pointer(node)) - base
		log.Printf("collected node %c at %d", node.value, ptr)
		nodes[ptr] = struct{}{}
	})
	index := -1
	for ptr := headerSize; ptr < uintptr(a.header.length); ptr += nodeSize {
		index++
		_, ok := nodes[ptr]
		if ok {
			log.Printf("node %d is reachable.", index)
			continue
		}
		node := a.getNode(ptr)
		log.Printf("node %d at %d with value %c is not reachable.", index, ptr, node.value)
	}
}

func (a *App) printTree() {
	a.walk(func(node *Node) {
		var left, right Node
		if node.left != 0 {
			left = *a.getNode(node.left)
		}
		if node.right != 0 {
			right = *a.getNode(node.right)
		}
		log.Printf("%c <- %c -> %c", left.value, node.value, right.value)
	})
}

func (a *App) walk(f func(*Node)) {
	if a.header == nil || a.header.root == 0 {
		return
	}
	root := a.getNode(a.header.root)
	a.walkFrom(root, f)
}

func (a *App) walkFrom(n *Node, f func(*Node)) {
	f(n)
	if n.left != 0 {
		left := a.getNode(n.left)
		a.walkFrom(left, f)
	}
	if n.right != 0 {
		right := a.getNode(n.right)
		a.walkFrom(right, f)
	}
}

func (a *App) collectNodeValues() []string {
	log.Println("COLLECT NODES")
	var out []string
	a.walk(func(n *Node) {
		out = append(out, string(n.value))
	})
	return out
}

func main() {
	flag.Parse()

	app := &App{filename: *filenameFlag}
	app.open()
	defer func() {
		fmt.Println("closing")
		err := app.Close()
		fmt.Println(err)
	}()

	app.loop()
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

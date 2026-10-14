package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	check(os.Mkdir("/tmp/namespace", 0700))
	f, err := os.Create("/tmp/namespace/file")
	check(err)
	_, err = f.Write([]byte("abc"))
	check(err)
	_, err = f.WriteAt([]byte("Z"), 1)
	check(err)
	data := make([]byte, 3)
	_, err = f.ReadAt(data, 0)
	check(err)
	if string(data) != "aZc" {
		panic("pread contents")
	}
	_, err = f.Seek(0, io.SeekStart)
	check(err)
	_, err = io.ReadFull(f, data)
	check(err)
	if string(data) != "aZc" {
		panic("read contents")
	}
	check(f.Truncate(5))
	check(f.Sync())
	check(f.Close())
	f, err = os.OpenFile("/tmp/namespace/file", os.O_WRONLY|os.O_APPEND, 0600)
	check(err)
	_, err = f.Write([]byte("X"))
	check(err)
	check(f.Close())
	check(os.Rename("/tmp/namespace/file", "/tmp/namespace/renamed"))
	stat, err := os.Stat("/tmp/namespace/renamed")
	check(err)
	if stat.Size() != 6 {
		panic("append size")
	}
	data, err = os.ReadFile("/tmp/namespace/renamed")
	check(err)
	if !bytes.Equal(data, []byte{'a', 'Z', 'c', 0, 0, 'X'}) {
		panic("resize contents")
	}
	entries, err := os.ReadDir("/tmp/namespace")
	check(err)
	if len(entries) != 1 || entries[0].Name() != "renamed" {
		panic("directory names")
	}
	check(os.Remove("/tmp/namespace/renamed"))
	check(os.Remove("/tmp/namespace"))
	_, err = os.Stdout.Write(bytes.Repeat([]byte{'x'}, 65537))
	check(err)
	_, err = fmt.Fprintln(os.Stderr, "namespace stderr")
	check(err)
	_, err = fmt.Println("namespace owner parity")
	check(err)
}

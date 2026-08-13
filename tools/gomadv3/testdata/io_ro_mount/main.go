package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
)

func main() {
	contents, err := os.ReadFile("mounted/schema.sql")
	if err != nil || string(contents) != "select 1;\n" {
		panic(fmt.Sprintf("ReadFile = %q, %v", contents, err))
	}
	file, err := os.Open("mounted/schema.sql")
	if err != nil {
		panic(err)
	}
	buffer := make([]byte, 6)
	if read, err := file.Read(buffer); err != nil || read != 6 || string(buffer) != "select" {
		panic(fmt.Sprintf("Read = %d, %q, %v", read, buffer, err))
	}
	if offset, err := file.Seek(7, io.SeekStart); err != nil || offset != 7 {
		panic(fmt.Sprintf("Seek = %d, %v", offset, err))
	}
	if read, err := file.ReadAt(buffer[:2], 7); err != nil || read != 2 || string(buffer[:2]) != "1;" {
		panic(fmt.Sprintf("ReadAt = %d, %q, %v", read, buffer[:2], err))
	}
	info, err := file.Stat()
	if err != nil || info.Name() != "schema.sql" || info.Size() != int64(len(contents)) || info.Mode().Perm() != 0o640 {
		panic(fmt.Sprintf("file Stat = %#v, %v", info, err))
	}
	if err := file.Close(); err != nil {
		panic(err)
	}
	entries, err := os.ReadDir("mounted")
	if err != nil || len(entries) != 2 || entries[0].Name() != "empty" || entries[1].Name() != "schema.sql" {
		panic(fmt.Sprintf("ReadDir = %#v, %v", entries, err))
	}
	if _, err := os.OpenFile("mounted/schema.sql", os.O_WRONLY, 0); err == nil || !strings.Contains(err.Error(), "read-only file system") {
		panic(fmt.Sprintf("write open error = %v", err))
	}
	if _, err := os.ReadFile("undeclared"); !errors.Is(err, fs.ErrNotExist) {
		panic(fmt.Sprintf("undeclared read error = %v", err))
	}
	fmt.Println("ok")
}

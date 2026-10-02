package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
)

type operation struct {
	name string
	call func() (string, error)
}
type result struct {
	Operation string `json:"operation"`
	Result    string `json:"result"`
	Error     string `json:"error"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 3 {
		return fmt.Errorf("usage: model_fs seed prefix-length")
	}
	seed, err := strconv.ParseInt(os.Args[1], 10, 64)
	if err != nil {
		return err
	}
	length, err := strconv.Atoi(os.Args[2])
	if err != nil {
		return err
	}
	if length < 1 || length > 64 {
		return fmt.Errorf("prefix length must be 1..64")
	}
	rng := rand.New(rand.NewSource(seed))
	var file *os.File
	var closed bool
	steps := []operation{
		{"mkdir workspace", func() (string, error) { return "", os.Mkdir("workspace", 0700) }},
		{"mkdir workspace/dir", func() (string, error) { return "", os.Mkdir("workspace/dir", 0700) }},
		{"stat-dir workspace/dir", func() (string, error) { info, err := os.Stat("workspace/dir"); return statResult(info, err) }},
		{"open workspace/handle rw-create-exclusive", func() (string, error) {
			var err error
			file, err = os.OpenFile("workspace/handle", os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
			return "", err
		}},
		{"write handle 616263646566", func() (string, error) { n, err := file.Write([]byte("abcdef")); return strconv.Itoa(n), err }},
		{"seek handle 0 start", func() (string, error) { n, err := file.Seek(0, io.SeekStart); return strconv.FormatInt(n, 10), err }},
		{"read handle 9", func() (string, error) {
			data := make([]byte, 9)
			n, err := file.Read(data)
			return fmt.Sprintf("%d:%x", n, data[:n]), err
		}},
		{"read handle 1", func() (string, error) {
			data := make([]byte, 1)
			n, err := file.Read(data)
			return fmt.Sprintf("%d:%x", n, data[:n]), err
		}},
		{"write-at handle 2 5859", func() (string, error) { n, err := file.WriteAt([]byte("XY"), 2); return strconv.Itoa(n), err }},
		{"read-at handle 1 8", func() (string, error) {
			data := make([]byte, 8)
			n, err := file.ReadAt(data, 1)
			return fmt.Sprintf("%d:%x", n, data[:n]), err
		}},
		{"truncate handle 3", func() (string, error) { return "", file.Truncate(3) }},
		{"stat handle", func() (string, error) { info, err := file.Stat(); return statResult(info, err) }},
		{"sync handle", func() (string, error) { return "", file.Sync() }},
		{"close handle", func() (string, error) { closed = true; return "", file.Close() }},
		{"close handle again", func() (string, error) { return "", file.Close() }},
		{"read closed handle 1", func() (string, error) {
			data := make([]byte, 1)
			n, err := file.Read(data)
			return fmt.Sprintf("%d:%x", n, data[:n]), err
		}},
	}
	for len(steps) < 64 {
		path := fmt.Sprintf("workspace/f%d", rng.Intn(4))
		other := fmt.Sprintf("workspace/f%d", rng.Intn(4))
		data := []byte{byte(rng.Intn(256)), byte(rng.Intn(256)), byte(rng.Intn(256))}
		size := int64(rng.Intn(10))
		switch rng.Intn(12) {
		case 0:
			steps = append(steps, operation{fmt.Sprintf("write-file %s %x", path, data), func() (string, error) { return "", os.WriteFile(path, data, 0600) }})
		case 1:
			steps = append(steps, operation{"read-file " + path, func() (string, error) { data, err := os.ReadFile(path); return fmt.Sprintf("%x", data), err }})
		case 2:
			steps = append(steps, operation{"stat " + path, func() (string, error) { info, err := os.Stat(path); return statResult(info, err) }})
		case 3:
			steps = append(steps, operation{"remove " + path, func() (string, error) { return "", os.Remove(path) }})
		case 4:
			steps = append(steps, operation{"rename " + path + " " + other, func() (string, error) { return "", os.Rename(path, other) }})
		case 5:
			steps = append(steps, operation{fmt.Sprintf("truncate %s %d", path, size), func() (string, error) { return "", os.Truncate(path, size) }})
		case 6:
			steps = append(steps, operation{"read-dir workspace", func() (string, error) {
				entries, err := os.ReadDir("workspace")
				if err != nil {
					return "", err
				}
				names := make([]string, 0, len(entries))
				for _, entry := range entries {
					names = append(names, fmt.Sprintf("%s:%t", entry.Name(), entry.IsDir()))
				}
				return marshal(names), nil
			}})
		case 7:
			steps = append(steps, operation{"chmod " + path + " 0600", func() (string, error) { return "", os.Chmod(path, 0600) }})
		case 8:
			steps = append(steps, operation{"mkdir workspace/dir", func() (string, error) { return "", os.Mkdir("workspace/dir", 0700) }})
		case 9:
			steps = append(steps, operation{"mkdir-all workspace/dir/nested", func() (string, error) { return "", os.MkdirAll("workspace/dir/nested", 0700) }})
		case 10:
			steps = append(steps, operation{"remove workspace/dir", func() (string, error) { return "", os.Remove("workspace/dir") }})
		case 11:
			steps = append(steps, operation{"remove-all workspace/dir/nested", func() (string, error) { return "", os.RemoveAll("workspace/dir/nested") }})
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	for _, step := range steps[:length] {
		value, err := step.call()
		if err := encoder.Encode(result{step.name, value, errorClass(err)}); err != nil {
			return err
		}
		if file == nil && step.name == "open workspace/handle rw-create-exclusive" {
			return fmt.Errorf("handle setup failed: %v", err)
		}
	}
	if file != nil && !closed {
		if err := file.Close(); err != nil {
			return err
		}
	}
	return nil
}

func statResult(info os.FileInfo, err error) (string, error) {
	if err != nil {
		return "", err
	}
	return marshal(struct {
		Name      string `json:"name"`
		Directory bool   `json:"directory"`
		Size      int64  `json:"size"`
		Mode      uint32 `json:"mode"`
	}{info.Name(), info.IsDir(), info.Size(), uint32(info.Mode().Perm())}), nil
}

func marshal(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func errorClass(err error) string {
	if err == nil {
		return "ok"
	}
	for _, entry := range []struct {
		err   error
		class string
	}{{io.EOF, "eof"}, {os.ErrClosed, "closed"}, {os.ErrNotExist, "not-exist"}, {os.ErrExist, "exist"}, {os.ErrPermission, "permission"}, {os.ErrInvalid, "invalid"}} {
		if errors.Is(err, entry.err) {
			return entry.class
		}
	}
	for errors.Unwrap(err) != nil {
		err = errors.Unwrap(err)
	}
	return fmt.Sprintf("unclassified:%T:%v", err, err)
}

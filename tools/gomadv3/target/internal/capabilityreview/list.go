package capabilityreview

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type Request struct {
	GoCommand    string
	Directory    string
	Package      string
	Tags         []string
	Overlay      string
	ModFile      string
	Environment  []string
	Test         bool
	OutputLimit  uint64
	PackageLimit int
}

type CommandError struct {
	Err          error
	Stderr       []byte
	InvalidInput bool
}

func (err *CommandError) Error() string {
	return fmt.Sprintf("inspect target capability closure: %v: %s", err.Err, err.Stderr)
}

func (err *CommandError) Unwrap() error {
	return err.Err
}

type Module struct {
	Path    string  `json:"Path"`
	Version string  `json:"Version"`
	Sum     string  `json:"Sum"`
	Main    bool    `json:"Main"`
	Dir     string  `json:"Dir"`
	Replace *Module `json:"Replace"`
}

type Package struct {
	ImportPath   string   `json:"ImportPath"`
	ForTest      string   `json:"ForTest"`
	DepOnly      bool     `json:"DepOnly"`
	Name         string   `json:"Name"`
	Standard     bool     `json:"Standard"`
	Dir          string   `json:"Dir"`
	GoFiles      []string `json:"GoFiles"`
	TestGoFiles  []string `json:"TestGoFiles"`
	XTestGoFiles []string `json:"XTestGoFiles"`
	CgoFiles     []string `json:"CgoFiles"`
	CFiles       []string `json:"CFiles"`
	CXXFiles     []string `json:"CXXFiles"`
	MFiles       []string `json:"MFiles"`
	HFiles       []string `json:"HFiles"`
	FFiles       []string `json:"FFiles"`
	SFiles       []string `json:"SFiles"`
	SwigFiles    []string `json:"SwigFiles"`
	SwigCXXFiles []string `json:"SwigCXXFiles"`
	SysoFiles    []string `json:"SysoFiles"`
	Imports      []string `json:"Imports"`
	TestImports  []string `json:"TestImports"`
	XTestImports []string `json:"XTestImports"`
	Module       *Module  `json:"Module"`
}

func List(ctx context.Context, request Request) ([]Package, error) {
	if request.GoCommand == "" || request.Directory == "" || request.Package == "" || request.OutputLimit == 0 || request.PackageLimit <= 0 {
		return nil, errors.New("capability review request is incomplete")
	}
	arguments := []string{"list", "-deps", "-json", "-mod=readonly"}
	if request.Test {
		arguments = append(arguments, "-test")
	}
	if request.Overlay != "" {
		arguments = append(arguments, "-overlay", request.Overlay)
	}
	if request.ModFile != "" {
		arguments = append(arguments, "-modfile", request.ModFile)
	}
	if len(request.Tags) > 0 {
		arguments = append(arguments, "-tags", strings.Join(request.Tags, ","))
	}
	arguments = append(arguments, request.Package)
	command := exec.CommandContext(ctx, request.GoCommand, arguments...)
	command.Dir = request.Directory
	command.Env = append([]string(nil), request.Environment...)
	stdout, stderr, err := runBounded(command, request.OutputLimit)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return nil, &CommandError{Err: err, Stderr: stderr, InvalidInput: invalidDiagnostic(stderr)}
		}
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(stdout))
	packages := make([]Package, 0)
	for {
		var pkg Package
		if err := decoder.Decode(&pkg); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("decode target capability closure: %w", err)
		}
		packages = append(packages, pkg)
		if len(packages) > request.PackageLimit {
			return nil, fmt.Errorf("target capability closure package count exceeds %d", request.PackageLimit)
		}
	}
	return packages, nil
}

func invalidDiagnostic(stderr []byte) bool {
	message := string(stderr)
	for _, fragment := range []string{
		"build constraints exclude all Go files", "cannot find main module", "directory prefix ", "is not in std",
		"import lookup disabled by -mod=readonly", "malformed import path", "missing go.sum entry", "no Go files in",
		"no required module provides package", "outside main module or its selected dependencies", "package without type was imported",
		"updates to go.mod needed",
	} {
		if strings.Contains(message, fragment) {
			return true
		}
	}
	return false
}

func runBounded(command *exec.Cmd, limit uint64) ([]byte, []byte, error) {
	stdout, err := newBoundedBuffer(limit)
	if err != nil {
		return nil, nil, err
	}
	stderr, err := newBoundedBuffer(limit)
	if err != nil {
		return nil, nil, err
	}
	command.Stdout = stdout
	command.Stderr = stderr
	runErr := command.Run()
	if stdout.overflow || stderr.overflow {
		return nil, nil, fmt.Errorf("target capability closure output exceeds %d bytes", limit)
	}
	if runErr != nil {
		return nil, stderr.bytes, runErr
	}
	return stdout.bytes, stderr.bytes, nil
}

type boundedBuffer struct {
	bytes    []byte
	limit    uint64
	overflow bool
}

func newBoundedBuffer(limit uint64) (*boundedBuffer, error) {
	if limit == 0 || limit > uint64(^uint(0)>>1) {
		return nil, fmt.Errorf("invalid command output limit %d", limit)
	}
	return &boundedBuffer{bytes: make([]byte, 0, int(limit)), limit: limit}, nil
}

func (buffer *boundedBuffer) Write(data []byte) (int, error) {
	remaining := buffer.limit - uint64(len(buffer.bytes))
	if uint64(len(data)) > remaining {
		buffer.bytes = append(buffer.bytes, data[:int(remaining)]...)
		buffer.overflow = true
		return len(data), nil
	}
	buffer.bytes = append(buffer.bytes, data...)
	return len(data), nil
}

package protocolnative

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNetworkTimeoutPortableSourceAdapters(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "..")
	overlay := filepath.Join(root, "internal/backends/native/toolchain/runtime/overlay/src")
	directory := t.TempDir()
	write := func(name string, source []byte) {
		t.Helper()
		path := filepath.Join(directory, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, source, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", []byte("module timeout.test\n\ngo 1.27.1\n"))
	wire, err := os.ReadFile(filepath.Join(overlay, "internal/gomadmodelwire/wire_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	write("modelwire/wire.go", wire)
	wireTests, err := os.ReadFile(filepath.Join(overlay, "internal/gomadmodelwire/wire_generated_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	write("modelwire/wire_test.go", bytes.ReplaceAll(wireTests, []byte(`"internal/gomadmodelwire"`), []byte(`"timeout.test/modelwire"`)))
	commands, err := os.ReadFile(filepath.Join(overlay, "internal/gomadio/process_commands.go"))
	if err != nil {
		t.Fatal(err)
	}
	write("adapters/commands.go", bytes.ReplaceAll(commands, []byte(`"internal/gomadmodelwire"`), []byte(`"timeout.test/modelwire"`)))
	var selected strings.Builder
	selected.WriteString(networkTimeoutAdapterPrelude)
	for _, selection := range []struct {
		path  string
		names string
	}{
		{"internal/gomadio/network.go", "ErrAddressInUse maximumChunkBytes Address connImplementation Conn Conn.Read Conn.Write Conn.LocalAddress Conn.RemoteAddress"},
		{"internal/gomadio/process_network.go", "processConn processConn.Read processConn.Write"},
		{"internal/gomadio/descriptor_backend.go", "descriptorStatus"},
		{"syscall/gomad_vfd_unix.go", "gomadVirtualErr"},
	} {
		selected.WriteString(networkTimeoutDeclarations(t, filepath.Join(overlay, selection.path), selection.names))
	}
	selected.WriteString(strings.ReplaceAll(networkTimeoutDeclarations(t, filepath.Join(overlay, "net/gomad.go"), "gomadInterceptConnRead gomadInterceptConnWrite gomadTCPAddr"), "gomadio.", ""))
	write("adapters/source.go", []byte(selected.String()))
	status := "package gomadvfd\n" + networkTimeoutDeclarations(t, filepath.Join(overlay, "internal/gomadvfd/descriptor.go"), "Status OK")
	write("vfd/status.go", []byte(status))
	write("adapters/source_test.go", []byte(networkTimeoutAdapterTests))
	providedTests := "package gomadio\nimport(\"errors\";\"os\";\"syscall\";\"testing\";\"time\")\n" + networkTimeoutDeclarations(t,
		filepath.Join(overlay, "internal/gomadio/network_handles_test.go"),
		"suppliedErrorConn suppliedErrorConn.Read suppliedErrorConn.Write suppliedErrorConn.Close suppliedErrorConn.CloseRead suppliedErrorConn.CloseWrite suppliedErrorConn.LocalAddress suppliedErrorConn.RemoteAddress suppliedErrorConn.SetDeadline suppliedErrorConn.SetReadDeadline suppliedErrorConn.SetWriteDeadline TestNetworkHandleSuppliedTimeoutAndDeadline")
	write("adapters/network_timeout_test.go", []byte(providedTests))
	translation := "package gomadio\nimport(\"context\";\"errors\";\"io\";\"os\";\"syscall\";\"testing\";gomadmodelwire \"timeout.test/modelwire\")\n" +
		networkTimeoutDeclarations(t, filepath.Join(overlay, "internal/gomadio/process_commands_test.go"), "TestProcessNetworkErrorTranslation")
	write("adapters/translation_test.go", []byte(translation))
	write("adapters/exports_test.go", []byte("package gomadio\nvar EncodeProcessNetworkErrorForTest=encodeProcessNetworkError\nvar DecodeProcessNetworkErrorForTest=decodeProcessNetworkError\n"))
	goExecutable := os.Getenv("GOMAD_STOCK_GO")
	if goExecutable == "" {
		goExecutable, err = exec.LookPath("go")
		if err != nil {
			t.Fatal(err)
		}
	}
	command := exec.CommandContext(t.Context(), goExecutable, "test", "-tags", "test_dep", "-count=1", "./...")
	command.Dir = directory
	command.Env = append(os.Environ(), "GOROOT=", "GOWORK=off", "GOENV=off", "GOFLAGS=", "GOTOOLCHAIN=local", "GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("portable source adapters: %v\n%s", err, output)
	}
}

func networkTimeoutDeclarations(t *testing.T, path, names string) string {
	t.Helper()
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, source, 0)
	if err != nil {
		t.Fatal(err)
	}
	wanted := make(map[string]bool)
	for _, name := range strings.Fields(names) {
		wanted[name] = true
	}
	var selected strings.Builder
	for _, declaration := range file.Decls {
		name := ""
		switch declaration := declaration.(type) {
		case *ast.GenDecl:
			for _, spec := range declaration.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					if wanted[spec.Name.Name] {
						name = spec.Name.Name
					}
				case *ast.ValueSpec:
					for _, identifier := range spec.Names {
						if wanted[identifier.Name] {
							name = identifier.Name
						}
					}
				}
			}
		case *ast.FuncDecl:
			name = declaration.Name.Name
			if declaration.Recv != nil {
				receiver := declaration.Recv.List[0].Type
				if pointer, ok := receiver.(*ast.StarExpr); ok {
					receiver = pointer.X
				}
				name = receiver.(*ast.Ident).Name + "." + name
			}
		}
		if wanted[name] {
			selected.Write(source[fileSet.Position(declaration.Pos()).Offset:fileSet.Position(declaration.End()).Offset])
			selected.WriteString("\n")
			delete(wanted, name)
		}
	}
	if len(wanted) != 0 {
		t.Fatalf("missing declarations in %s: %v", path, wanted)
	}
	return selected.String()
}

const networkTimeoutAdapterPrelude = `package gomadio
import (
 "errors"
 "io"
 "net"
 "sync"
 "syscall"
 "time"
 gomadvfd "timeout.test/vfd"
)

var poll = struct{ ErrNetClosing error }{errors.New("use of closed network connection")}
type netFD struct{}
type conn struct{ fd *netFD }
func (connection *conn) ok() bool { return connection != nil && connection.fd != nil }
var providedConnection *Conn
func gomadConnection(*netFD) *Conn { return providedConnection }
func gomadIOEnabled() bool { return true }
type TCPAddr = net.TCPAddr
type OpError = net.OpError
var ParseIP = net.ParseIP
var IPv4 = net.IPv4
type descriptorIdentity struct{}
const (
 EAGAIN = syscall.EAGAIN
 EBADF = syscall.EBADF
 ECONNREFUSED = syscall.ECONNREFUSED
 EADDRINUSE = syscall.EADDRINUSE
 ENOBUFS = syscall.ENOBUFS
 ENOTCONN = syscall.ENOTCONN
 EPIPE = syscall.EPIPE
 ECONNRESET = syscall.ECONNRESET
 EINVAL = syscall.EINVAL
 EOPNOTSUPP = syscall.EOPNOTSUPP
 ETIMEDOUT = syscall.ETIMEDOUT
)
var suppliedResponse processNetworkResult
func exchangeProcessNetwork(processNetworkCommand) (processNetworkResult, error) {
 return suppliedResponse, suppliedResponse.Err
}
`

const networkTimeoutAdapterTests = `package gomadio
import (
 "errors"
 "os"
 "syscall"
 "testing"
 gomadvfd "timeout.test/vfd"
)

func TestNetInterceptorsPreservePartialTimeoutAndDeadline(t *testing.T) {
 for _, source := range []error{syscall.ETIMEDOUT, os.ErrDeadlineExceeded} {
  providedConnection = &Conn{implementation: &suppliedErrorConn{err: source}}
  wrapped := &conn{fd: &netFD{}}
  for _, op := range []func() (int, error, bool){
   func() (int, error, bool) { return gomadInterceptConnRead(wrapped, make([]byte, 8)) },
   func() (int, error, bool) { return gomadInterceptConnWrite(wrapped, []byte("abcdef")) },
  } {
   n, err, handled := op()
   if n != 3 || !handled || !errors.Is(err, source) {
    t.Fatalf("wrapped partial=%d,%v,%v", n, err, handled)
   }
   timeout, ok := err.(interface{ Timeout() bool })
   if !ok || !timeout.Timeout() {
    t.Fatalf("wrapper classification=%v", err)
   }
  }
 }
 providedConnection = nil
 if n, err, handled := gomadInterceptConnRead(&conn{fd: &netFD{}}, make([]byte, 1)); n != 0 || !handled || err != ErrUnsupported {
  t.Fatalf("missing model=%d,%v,%v", n, err, handled)
 }
 if n, err, handled := gomadInterceptConnWrite(nil, []byte("x")); n != 0 || !handled || err != syscall.EINVAL {
  t.Fatalf("invalid connection=%d,%v,%v", n, err, handled)
 }
}

func TestDescriptorTimeoutAndExistingStatusMapping(t *testing.T) {
 if status := descriptorStatus(syscall.ETIMEDOUT); status != gomadvfd.Status(12) {
  t.Fatalf("timeout status=%v", status)
 }
 err := gomadVirtualErr(gomadvfd.Status(12))
 if !errors.Is(err, syscall.ETIMEDOUT) || errors.Is(err, os.ErrDeadlineExceeded) {
  t.Fatalf("descriptor timeout=%v", err)
 }
 timeout, ok := err.(interface{ Timeout() bool })
 if !ok || !timeout.Timeout() {
  t.Fatalf("classification=%v", err)
 }
 for _, test := range []struct {
  status gomadvfd.Status
  err error
 }{
  {gomadvfd.OK, nil},
  {gomadvfd.Closed, syscall.EBADF},
  {gomadvfd.Reset, syscall.ECONNRESET},
  {gomadvfd.Capacity, syscall.ENOBUFS},
  {gomadvfd.Status(255), syscall.EOPNOTSUPP},
 } {
  if got := gomadVirtualErr(test.status); !errors.Is(got, test.err) {
   t.Fatalf("status %d=%v want %v", test.status, got, test.err)
  }
 }
}

func TestProcessConnPartialTimeoutAndBounds(t *testing.T) {
 for _, err := range []error{syscall.ETIMEDOUT, os.ErrDeadlineExceeded} {
  suppliedResponse = processNetworkResult{BytesTransferred: 3, Data: []byte("abc"), Err: err}
  conn := &processConn{}
  if n, got := conn.Read(make([]byte, 8)); n != 3 || !errors.Is(got, err) {
   t.Fatalf("read=%d,%v", n, got)
  }
  if n, got := conn.Write([]byte("abcdef")); n != 3 || !errors.Is(got, err) {
   t.Fatalf("write=%d,%v", n, got)
  }
 }
 suppliedResponse = processNetworkResult{BytesTransferred: 4, Data: []byte("abc"), Err: syscall.ETIMEDOUT}
 if n, err := (&processConn{}).Read(make([]byte, 8)); n != 0 || err != syscall.EIO {
  t.Fatalf("bad read count=%d,%v", n, err)
 }
 suppliedResponse = processNetworkResult{BytesTransferred: 7, Err: syscall.ETIMEDOUT}
 if n, err := (&processConn{}).Write([]byte("abcdef")); n != 0 || err != syscall.EIO {
  t.Fatalf("bad write count=%d,%v", n, err)
 }
}
`

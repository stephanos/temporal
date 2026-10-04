package gomadio_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"reflect"
	"syscall"
	"testing"

	. "internal/gomadio"
	"internal/gomadmodelwire"
)

func TestProcessNetworkCommandVectors(t *testing.T) {
	cases := []struct {
		command                 ProcessNetworkCommandForTest
		result                  ProcessNetworkResultForTest
		requestHex, responseHex string
	}{
		{ProcessNetworkCommandForTest{Operation: ProcessNetworkOperationForTest(1), Network: "tcp4", Host: "127.0.0.1", Port: 32123}, ProcessNetworkResultForTest{Handle: 17, Local: Address{IP: "127.0.0.1", Port: 32123}}, "474f4d41444d4f01010100010000000000000000000000000000000000007d7b000000000000000000000000000000000000000000000000000000000000000000000000000000047463703400000000000000093132372e302e302e310000000000000000", "474f4d41444d4f01020000000000000000000000000000110000000000007d7b000000000000000000000000000000000000000000000000000000000000000000000000000000093132372e302e302e3100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{ProcessNetworkCommandForTest{Operation: ProcessNetworkOperationForTest(2), Network: "tcp4", Host: "127.0.0.1", Port: 32123, DeadlineNanos: 123456789}, ProcessNetworkResultForTest{Handle: 17, Local: Address{IP: "127.0.0.1", Port: 32123}, Remote: Address{IP: "127.0.0.2", Port: 32124}}, "474f4d41444d4f01010100020000000000000000000000000000000000007d7b00000000075bcd1500000000000000000000000000000000000000000000000000000000000000047463703400000000000000093132372e302e302e310000000000000000", "474f4d41444d4f01020000000000000000000000000000110000000000007d7b0000000000007d7c00000000000000000000000000000000000000000000000000000000000000093132372e302e302e3100000000000000093132372e302e302e320000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{ProcessNetworkCommandForTest{Operation: ProcessNetworkOperationForTest(3), Handle: 7}, ProcessNetworkResultForTest{Handle: 17, Local: Address{IP: "127.0.0.1", Port: 32123}, Remote: Address{IP: "127.0.0.2", Port: 32124}}, "474f4d41444d4f010101000300000000000000000000000700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", "474f4d41444d4f01020000000000000000000000000000110000000000007d7b0000000000007d7c00000000000000000000000000000000000000000000000000000000000000093132372e302e302e3100000000000000093132372e302e302e320000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{ProcessNetworkCommandForTest{Operation: ProcessNetworkOperationForTest(4), Handle: 7}, ProcessNetworkResultForTest{Err: staleVectorError{}}, "474f4d41444d4f010101000400000000000000000000000700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", "474f4d41444d4f0102000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001300000000000000157374616c65204e46532066696c652068616e646c65000000000000000000000000000000000000000000000000"},
		{ProcessNetworkCommandForTest{Operation: ProcessNetworkOperationForTest(5), Handle: 7, DeadlineNanos: 123456789}, ProcessNetworkResultForTest{Err: staleVectorError{}}, "474f4d41444d4f010101000500000000000000000000000700000000075bcd150000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", "474f4d41444d4f0102000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001300000000000000157374616c65204e46532066696c652068616e646c65000000000000000000000000000000000000000000000000"},
		{ProcessNetworkCommandForTest{Operation: ProcessNetworkOperationForTest(6), Handle: 7, ReadLength: 9}, ProcessNetworkResultForTest{BytesTransferred: 2, Data: []byte("ab"), Err: io.EOF}, "474f4d41444d4f010101000600000000000000000000000700000000000000000000000000000000000000000000000900000000000000000000000000000000000000000000000000000000000000000000000000000000", "474f4d41444d4f0102000000000000000000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000000000000000000026162000000000000000000000000000000020000000000000003454f46000000000000000000000000000000000000000000000000"},
		{ProcessNetworkCommandForTest{Operation: ProcessNetworkOperationForTest(7), Handle: 7, Data: []byte("abcd")}, ProcessNetworkResultForTest{BytesTransferred: 2, Err: io.EOF}, "474f4d41444d4f01010100070000000000000000000000070000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000461626364", "474f4d41444d4f010200000000000000000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000020000000000000003454f46000000000000000000000000000000000000000000000000"},
		{ProcessNetworkCommandForTest{Operation: ProcessNetworkOperationForTest(8), Handle: 7}, ProcessNetworkResultForTest{Err: staleVectorError{}}, "474f4d41444d4f010101000800000000000000000000000700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", "474f4d41444d4f0102000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001300000000000000157374616c65204e46532066696c652068616e646c65000000000000000000000000000000000000000000000000"},
		{ProcessNetworkCommandForTest{Operation: ProcessNetworkOperationForTest(9), Handle: 7}, ProcessNetworkResultForTest{Err: staleVectorError{}}, "474f4d41444d4f010101000900000000000000000000000700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", "474f4d41444d4f0102000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001300000000000000157374616c65204e46532066696c652068616e646c65000000000000000000000000000000000000000000000000"},
		{ProcessNetworkCommandForTest{Operation: ProcessNetworkOperationForTest(10), Handle: 7}, ProcessNetworkResultForTest{Err: staleVectorError{}}, "474f4d41444d4f010101000a00000000000000000000000700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", "474f4d41444d4f0102000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001300000000000000157374616c65204e46532066696c652068616e646c65000000000000000000000000000000000000000000000000"},
		{ProcessNetworkCommandForTest{Operation: ProcessNetworkOperationForTest(11), Handle: 7, DeadlineNanos: 123456789}, ProcessNetworkResultForTest{Err: staleVectorError{}}, "474f4d41444d4f010101000b00000000000000000000000700000000075bcd150000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", "474f4d41444d4f0102000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001300000000000000157374616c65204e46532066696c652068616e646c65000000000000000000000000000000000000000000000000"},
		{ProcessNetworkCommandForTest{Operation: ProcessNetworkOperationForTest(12), Handle: 7, DeadlineNanos: 123456789}, ProcessNetworkResultForTest{Err: staleVectorError{}}, "474f4d41444d4f010101000c00000000000000000000000700000000075bcd150000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", "474f4d41444d4f0102000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001300000000000000157374616c65204e46532066696c652068616e646c65000000000000000000000000000000000000000000000000"},
		{ProcessNetworkCommandForTest{Operation: ProcessNetworkOperationForTest(13), Handle: 7, DeadlineNanos: 123456789}, ProcessNetworkResultForTest{Err: staleVectorError{}}, "474f4d41444d4f010101000d00000000000000000000000700000000075bcd150000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", "474f4d41444d4f0102000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001300000000000000157374616c65204e46532066696c652068616e646c65000000000000000000000000000000000000000000000000"},
	}
	for _, test := range cases {
		encoded, err := EncodeProcessNetworkCommandForTest(test.command)
		if err != nil || hex.EncodeToString(encoded) != test.requestHex {
			t.Fatalf("operation %d request=%x err=%v", test.command.Operation, encoded, err)
		}
		decoded, err := DecodeProcessNetworkCommandForTest(encoded)
		if err != nil || !reflect.DeepEqual(decoded, test.command) {
			t.Fatalf("operation %d command=%#v err=%v", test.command.Operation, decoded, err)
		}
		expected, _ := hex.DecodeString(test.responseHex)
		encoded, ok := EncodeProcessNetworkResultForTest(test.result)
		if !ok || !bytes.Equal(encoded, expected) {
			t.Fatalf("operation %d response=%x ok=%v", test.command.Operation, encoded, ok)
		}
		result, err := DecodeProcessNetworkResultForTest(expected)
		if err != nil || !errors.Is(result.Err, vectorDomainError(test.result.Err)) {
			t.Fatalf("operation %d result error=%v codec error=%v", test.command.Operation, result.Err, err)
		}
		result.Err = nil
		want := test.result
		want.Err = nil

		if !reflect.DeepEqual(result, want) {
			t.Fatalf("operation %d result=%#v want=%#v", test.command.Operation, result, want)
		}
	}
}

type staleVectorError struct{}

func (staleVectorError) Error() string { return "stale NFS file handle" }

func (staleVectorError) Unwrap() error { return syscall.ESTALE }

func vectorDomainError(err error) error {
	if value := errors.Unwrap(err); value != nil {
		return value
	}
	return err
}

func TestProcessNetworkPermissiveCommands(t *testing.T) {
	for op := 1; op <= 13; op++ {
		request := gomadmodelwire.Request{Model: gomadmodelwire.ModelNetwork, Operation: gomadmodelwire.Operation(op), String1: "tcp", String2: "host", Int1: -9, Int2: -11, Uint1: ^uint64(0), Uint2: 19, Flags: ^uint64(0), Data: []byte("unused")}
		encoded, err := gomadmodelwire.EncodeRequest(request)
		if err != nil {
			t.Fatal(err)
		}
		command, err := DecodeProcessNetworkCommandForTest(encoded)
		if err != nil {
			t.Fatalf("operation %d: %v", op, err)
		}
		if command.Handle != 0 {
			t.Fatalf("zero handle changed: %+v", command)
		}
		switch op {
		case 1, 2:
			if command.Network != "tcp" || command.Host != "host" || command.Port != -9 {
				t.Fatalf("address=%+v", command)
			}
		case 5, 11, 12, 13:
			if command.DeadlineNanos != -9 {
				t.Fatalf("deadline=%+v", command)
			}
		case 6:
			if command.ReadLength != ^uint64(0) {
				t.Fatalf("read=%+v", command)
			}
		case 7:
			if string(command.Data) != "unused" {
				t.Fatalf("write=%+v", command)
			}
		}
	}
	if _, err := DecodeProcessNetworkCommandForTest([]byte("bad")); err == nil {
		t.Fatal("malformed command accepted")
	}
	if _, err := DecodeProcessNetworkResultForTest([]byte("bad")); err == nil {
		t.Fatal("malformed result accepted")
	}
}

func TestProcessNetworkErrorTranslation(t *testing.T) {
	cases := []struct {
		err     error
		code    gomadmodelwire.ErrorCode
		decoded error
	}{
		{nil, 0, nil}, {io.EOF, 2, io.EOF}, {context.DeadlineExceeded, 3, os.ErrDeadlineExceeded}, {os.ErrDeadlineExceeded, 3, os.ErrDeadlineExceeded},
		{context.Canceled, 4, context.Canceled}, {ErrAddressInUse, 5, ErrAddressInUse}, {ErrClosed, 6, ErrClosed}, {ErrConnectionRefused, 7, ErrConnectionRefused},
		{ErrResourceExhausted, 8, ErrResourceExhausted}, {ErrUnsupported, 9, ErrUnsupported}, {syscall.ESTALE, 19, syscall.ESTALE},
		{errors.Join(context.Canceled, io.EOF), 2, io.EOF}, {errors.Join(ErrClosed, context.DeadlineExceeded), 3, os.ErrDeadlineExceeded},
		{errors.Join(ErrUnsupported, ErrResourceExhausted), 8, ErrResourceExhausted},
	}
	for _, test := range cases {
		wire := EncodeProcessNetworkErrorForTest(test.err)
		if wire.Code != test.code {
			t.Fatalf("%v code=%d want=%d", test.err, wire.Code, test.code)
		}
		// Decode independently supplied error codes, without relying on the encoder.
		result := DecodeProcessNetworkErrorForTest(gomadmodelwire.WireError{Code: test.code, Message: "ignored"})
		if !errors.Is(result, test.decoded) {
			t.Fatalf("code %d error=%v want=%v", test.code, result, test.decoded)
		}
	}
	result := DecodeProcessNetworkErrorForTest(gomadmodelwire.WireError{Code: 1, Message: "custom"})
	if result == nil || result.Error() != "custom" {
		t.Fatalf("generic=%v", result)
	}
}

func TestProcessNetworkResourceKinds(t *testing.T) {
	listener := RegisterProcessNetworkResourceForTest(31, true)
	connection := RegisterProcessNetworkResourceForTest(31, false)
	t.Cleanup(func() { RemoveProcessNetworkResourceForTest(listener); RemoveProcessNetworkResourceForTest(connection) })
	for _, test := range []struct {
		op             ProcessNetworkOperationForTest
		handle, domain uint64
	}{
		{3, 0, 31}, {6, connection, 32}, {6, listener, 31}, {3, connection, 31}, {13, ^uint64(0), 31},
	} {
		result := ApplyProcessNetworkOperationForTest(test.domain, ProcessNetworkCommandForTest{Operation: test.op, Handle: test.handle})
		if !errors.Is(result.Err, syscall.ESTALE) {
			t.Fatalf("%+v => %v", test, result.Err)
		}
	}
	if result := ApplyProcessNetworkOperationForTest(31, ProcessNetworkCommandForTest{Operation: 99}); !errors.Is(result.Err, ErrUnsupported) {
		t.Fatalf("unsupported=%v", result.Err)
	}
	if n, err := ZeroLengthProcessNetworkReadForTest(); n != 0 || err != nil {
		t.Fatalf("zero read=%d,%v", n, err)
	}
	if n, err := ZeroLengthProcessNetworkWriteForTest(); n != 0 || err != nil {
		t.Fatalf("zero write=%d,%v", n, err)
	}
}

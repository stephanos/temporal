package livecap

import (
	"debug/macho"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

func Read(path string, expected Expectation) (_ Record, retErr error) {
	file, err := os.Open(path)
	if err != nil {
		return Record{}, fmt.Errorf("open live capability target: %w", err)
	}
	defer func() {
		retErr = errors.Join(retErr, file.Close())
	}()
	return ReadFile(file, expected)
}

func ReadFile(file *os.File, expected Expectation) (Record, error) {
	info, err := file.Stat()
	if err != nil {
		return Record{}, fmt.Errorf("stat live capability target: %w", err)
	}
	if !info.Mode().IsRegular() {
		return Record{}, errors.New("live capability target is not a regular file")
	}
	parsed, err := macho.NewFile(file)
	if err != nil {
		return Record{}, fmt.Errorf("parse live capability Mach-O target: %w", err)
	}
	if parsed.Symtab == nil {
		return Record{}, errors.New("live capability Mach-O target has no symbol table")
	}
	record, err := extractMachORecord(parsed.Symtab.Syms, parsed.Sections)
	if err != nil {
		return Record{}, err
	}
	return Decode(record, expected)
}

func extractMachORecord(symbols []macho.Symbol, sections []*macho.Section) ([]byte, error) {
	var matches []macho.Symbol
	for _, symbol := range symbols {
		if symbol.Name == ReservedSymbol {
			matches = append(matches, symbol)
		}
	}
	if len(matches) != 1 {
		return nil, fmt.Errorf("live capability Mach-O target must contain exactly one %s symbol, found %d", ReservedSymbol, len(matches))
	}
	address := matches[0].Value
	var containing []*macho.Section
	for _, section := range sections {
		if address >= section.Addr && address-section.Addr < section.Size {
			containing = append(containing, section)
		}
	}
	if len(containing) != 1 {
		return nil, fmt.Errorf("live capability symbol does not resolve to exactly one section")
	}
	section := containing[0]
	if section.Seg != "__TEXT" {
		return nil, fmt.Errorf("live capability symbol is not in a read-only Mach-O section")
	}
	offset := address - section.Addr
	if section.Size-offset < HeaderBytes {
		return nil, fmt.Errorf("live capability header exceeds its Mach-O section bounds")
	}
	header := make([]byte, HeaderBytes)
	if read, err := section.ReadAt(header, int64(offset)); err != nil {
		return nil, fmt.Errorf("read live capability header: %w", err)
	} else if read != len(header) {
		return nil, fmt.Errorf("read live capability header: %w", io.ErrUnexpectedEOF)
	}
	payloadBytes := binary.LittleEndian.Uint64(header[24:32])
	if payloadBytes > MaximumPayloadBytes {
		return nil, &CapacityError{Resource: "payload bytes", Required: payloadBytes, Maximum: MaximumPayloadBytes}
	}
	if payloadBytes > section.Size-offset-HeaderBytes {
		return nil, fmt.Errorf("live capability payload exceeds its Mach-O section bounds")
	}
	record := make([]byte, HeaderBytes+payloadBytes)
	if read, err := section.ReadAt(record, int64(offset)); err != nil {
		return nil, fmt.Errorf("read live capability record: %w", err)
	} else if read != len(record) {
		return nil, fmt.Errorf("read live capability record: %w", io.ErrUnexpectedEOF)
	}
	return record, nil
}

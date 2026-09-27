package livecap

import (
	"debug/elf"
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
	var magic [4]byte
	if _, err := file.ReadAt(magic[:], 0); err != nil {
		return Record{}, fmt.Errorf("read live capability target header: %w", err)
	}
	var record []byte
	if string(magic[:]) == elf.ELFMAG {
		record, err = readELFRecord(file)
	} else {
		record, err = readMachORecord(file)
	}
	if err != nil {
		return Record{}, err
	}
	return Decode(record, expected)
}

func readMachORecord(file *os.File) ([]byte, error) {
	parsed, err := macho.NewFile(file)
	if err != nil {
		return nil, fmt.Errorf("parse live capability Mach-O target: %w", err)
	}
	if parsed.Symtab == nil {
		return nil, errors.New("live capability Mach-O target has no symbol table")
	}
	return extractMachORecord(parsed.Symtab.Syms, parsed.Sections)
}

func readELFRecord(file *os.File) ([]byte, error) {
	parsed, err := elf.NewFile(file)
	if err != nil {
		return nil, fmt.Errorf("parse live capability ELF target: %w", err)
	}
	symbols, err := parsed.Symbols()
	if err != nil {
		return nil, fmt.Errorf("live capability ELF target has no symbol table: %w", err)
	}
	return extractELFRecord(symbols, parsed.Sections)
}

// objectSection is the object-format-neutral view of a loaded section that the
// record extraction needs: where it is mapped, how large it is, whether the
// loader maps it read-only, and how to read its bytes.
type objectSection struct {
	io.ReaderAt
	addr, size uint64
	readOnly   bool
}

type objectSymbol struct {
	name  string
	value uint64
}

func extractMachORecord(symbols []macho.Symbol, sections []*macho.Section) ([]byte, error) {
	objectSymbols := make([]objectSymbol, 0, len(symbols))
	for _, symbol := range symbols {
		objectSymbols = append(objectSymbols, objectSymbol{name: symbol.Name, value: symbol.Value})
	}
	objectSections := make([]objectSection, 0, len(sections))
	for _, section := range sections {
		objectSections = append(objectSections, objectSection{ReaderAt: section, addr: section.Addr, size: section.Size, readOnly: section.Seg == "__TEXT"})
	}
	return extractRecord(objectSymbols, objectSections)
}

func extractELFRecord(symbols []elf.Symbol, sections []*elf.Section) ([]byte, error) {
	objectSymbols := make([]objectSymbol, 0, len(symbols))
	for _, symbol := range symbols {
		objectSymbols = append(objectSymbols, objectSymbol{name: symbol.Name, value: symbol.Value})
	}
	objectSections := make([]objectSection, 0, len(sections))
	for _, section := range sections {
		if section.Flags&elf.SHF_ALLOC == 0 || section.Type == elf.SHT_NOBITS {
			continue
		}
		objectSections = append(objectSections, objectSection{ReaderAt: section, addr: section.Addr, size: section.Size, readOnly: section.Flags&elf.SHF_WRITE == 0})
	}
	return extractRecord(objectSymbols, objectSections)
}

func extractRecord(symbols []objectSymbol, sections []objectSection) ([]byte, error) {
	var matches []objectSymbol
	for _, symbol := range symbols {
		if symbol.name == ReservedSymbol {
			matches = append(matches, symbol)
		}
	}
	if len(matches) != 1 {
		return nil, fmt.Errorf("live capability target must contain exactly one %s symbol, found %d", ReservedSymbol, len(matches))
	}
	address := matches[0].value
	var containing []objectSection
	for _, section := range sections {
		if address >= section.addr && address-section.addr < section.size {
			containing = append(containing, section)
		}
	}
	if len(containing) != 1 {
		return nil, fmt.Errorf("live capability symbol does not resolve to exactly one section")
	}
	section := containing[0]
	if !section.readOnly {
		return nil, fmt.Errorf("live capability symbol is not in a read-only section")
	}
	offset := address - section.addr
	if section.size-offset < HeaderBytes {
		return nil, fmt.Errorf("live capability header exceeds its section bounds")
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
	if payloadBytes > section.size-offset-HeaderBytes {
		return nil, fmt.Errorf("live capability payload exceeds its section bounds")
	}
	record := make([]byte, HeaderBytes+payloadBytes)
	if read, err := section.ReadAt(record, int64(offset)); err != nil {
		return nil, fmt.Errorf("read live capability record: %w", err)
	} else if read != len(record) {
		return nil, fmt.Errorf("read live capability record: %w", io.ErrUnexpectedEOF)
	}
	return record, nil
}

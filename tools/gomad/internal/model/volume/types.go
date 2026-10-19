package volume

type Kind uint8

type MountStatus uint16

type Child struct {
	Name string
	Kind Kind
	Mode uint32
}

type LoadEntry struct {
	Kind     Kind
	Mode     uint32
	Data     []byte
	Children []Child
}

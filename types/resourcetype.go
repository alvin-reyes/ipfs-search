package types

// ResourceType is an enum of possible resource types.
type ResourceType uint8

const (
	// UndefinedType is for resources for which the type is as of yet undefined.
	UndefinedType ResourceType = iota
	// UnsupportedType represents an as-of-yet unsupported type.
	UnsupportedType
	// FileType is a regular file.
	FileType
	// DirectoryType is a directory.
	DirectoryType
	// PartialType represents *unreferenced* partial items.
	PartialType
)

func (t ResourceType) String() string {
	switch t {
	case UndefinedType:
		return "undefined"
	case UnsupportedType:
		return "unsupported"
	case FileType:
		return "file"
	case DirectoryType:
		return "directory"
	case PartialType:
		return "partial"
	default:
		panic("Invalid value for ResourceType.")
	}
}

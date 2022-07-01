package common

import "encoding/binary"

var _ binary.ByteOrder = FieldElementEndianness{}

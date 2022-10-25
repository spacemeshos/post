package gpu

import "C"
import (
	"unsafe"
)

func cBytesCloneToGoBytes(data *C.uchar, len int) []byte {
	cBytes := CBytes{
		data: unsafe.Pointer(data),
		len:  len,
	}

	return cBytes.GoBytesClone()
}

// GoBytes is an alias type for []byte slice, used to define local methods.
type GoBytes []byte

// CBytesClone is using the built-in `CBytes` cgo function to
// create a C array that is allocated via the C allocator.
// It is the caller's responsibility to arrange for it to be freed
// via the C allocator by calling the Free method on it.
// The "Clone" name suffix is to explicitly clarify that it clones the underlying array,
// in oppose to creating a pointer which references it.
func (b GoBytes) CBytesClone() CBytes {
	p := C.CBytes(b)
	return CBytes{data: p, len: len(b)}
}

// CBytes represents a C array allocated via the C allocator.
type CBytes struct {
	data unsafe.Pointer
	len  int
}

// GoBytesClone is using the built-in `GoBytes` cgo function to
// create a new Go []byte slice from the C array.
// It is the caller's responsibility to arrange for the C array to
// eventually be freed via the C allocator, by calling the Free method on it.
// The "Clone" name suffix is to explicitly clarify that it clones the C array,
// in oppose to creating a pointer which references it.
func (s CBytes) GoBytesClone() []byte {
	return C.GoBytes(s.data, C.int(s.len))
}

func cStringArrayToGoString(src [256]C.char) string {
	var dst []byte
	for i := 0; i < 256; i++ {
		if src[i] == 0 {
			break
		}
		dst = append(dst, byte(src[i]))
	}
	return string(dst)
}

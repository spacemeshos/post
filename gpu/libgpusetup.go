package gpu

// #cgo !windows LDFLAGS: -lgpu-setup
// #cgo windows LDFLAGS: -L${SRCDIR} -lgpu-setup-win64
import "C"

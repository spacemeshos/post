package gpu

// #cgo linux LDFLAGS: -Wl,-rpath,$ORIGIN
// #cgo darwin LDFLAGS: -Wl,-rpath -Wl,@loader_path
// #cgo !windows LDFLAGS: -lgpu-setup
// #cgo windows LDFLAGS: -L${SRCDIR} -lgpu-setup-win64
import "C"

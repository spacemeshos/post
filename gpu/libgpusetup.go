package gpu

// #cgo linux LDFLAGS: -Wl,-rpath,$ORIGIN
// #cgo darwin LDFLAGS: -Wl,-rpath,@loader_path
// #cgo linux,!no_ext_rpath darwin,!no_ext_rpath LDFLAGS: -Wl,-rpath,./build -Wl,-rpath,../build -Wl,-rpath,../../build -Wl,-rpath,../../../build
// #cgo linux,!no_ext_rpath darwin,!no_ext_rpath LDFLAGS: -L./build -L../build -L../../build -L../../../build
// #cgo !windows LDFLAGS: -lgpu-setup
// #cgo windows LDFLAGS: -L${SRCDIR} -lgpu-setup-win64
import "C"

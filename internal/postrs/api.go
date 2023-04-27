package postrs

type ComputeAPIClass int

const (
	ComputeAPIClassUnspecified = iota - 1
	ComputeAPIClassCPU
	ComputeAPIClassGPU
)

type ComputeProvider struct {
	ID         uint
	Model      string
	ComputeAPI ComputeAPIClass
}

func (c ComputeAPIClass) String() string {
	switch c {
	case ComputeAPIClassCPU:
		return "CPU"
	case ComputeAPIClassGPU:
		return "GPU"
	default:
		return "Unspecified"
	}
}

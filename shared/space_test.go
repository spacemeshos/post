package shared

//func TestSpace_Validate(t *testing.T) {
//	r := require.New(t)
//	r.EqualError(ValidateSpace(uint64(LabelGroupSize-1)), fmt.Sprintf("space (%d) must be a power of 2", LabelGroupSize-1))
//	r.NoError(ValidateSpace(uint64(LabelGroupSize)))
//	r.EqualError(ValidateSpace(uint64(LabelGroupSize+1)), fmt.Sprintf("space (%d) must be a power of 2", LabelGroupSize+1))
//
//	r.EqualError(ValidateSpace(uint64(MaxSpace-1)), fmt.Sprintf("space (%d) must be a power of 2", MaxSpace-1))
//	r.NoError(ValidateSpace(uint64(MaxSpace)))
//	r.EqualError(ValidateSpace(uint64(MaxSpace+1)), fmt.Sprintf("space (%d) is greater than the supported max (%d)", MaxSpace+1, MaxSpace))
//}

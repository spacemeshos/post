package gpu

func cStringArrayToGoString(src [256]cChar) string {
	var dst []byte
	for i := 0; i < 256; i++ {
		if src[i] == 0 {
			break
		}
		dst = append(dst, byte(src[i]))
	}
	return string(dst)
}

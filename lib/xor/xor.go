package xor

// Bytes XORs bytes.
func Bytes(
	dst []byte,
	a []byte,
	b []byte,
) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		dst[i] = a[i] ^ b[i]
	}
	return n
}

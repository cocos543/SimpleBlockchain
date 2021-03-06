package main

func IntToHex(n int64) []byte {
	// int64 占用64个bit, 8个字节
	var dst [8]byte
	for i := 0; i < len(dst); i++ {
		lsh := len(dst) - (i + 1)
		dst[i] = uint8(n >> uint8(8*lsh))
	}
	return dst[:]
}

/*
// IntToHex converts an int64 to a byte array
func IntToHex(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}
*/

// ReverseBytes reverses a byte array
func ReverseBytes(data []byte) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
}

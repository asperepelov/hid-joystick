package hidjoystick

// Report — сырой HID-репорт от устройства.
// Никакой нормализации и интерпретации — только байты и методы чтения.
type Report struct {
	Data []byte
}

// Len возвращает длину репорта в байтах.
func (r Report) Len() int {
	return len(r.Data)
}

// Byte возвращает байт по офсету. Возвращает 0 если офсет за пределами репорта.
func (r Report) Byte(offset int) byte {
	if offset < 0 || offset >= len(r.Data) {
		return 0
	}
	return r.Data[offset]
}

// U16LE читает uint16 little-endian по офсету.
func (r Report) U16LE(offset int) uint16 {
	if offset+1 >= len(r.Data) {
		return 0
	}
	return uint16(r.Data[offset]) | uint16(r.Data[offset+1])<<8
}

// U16BE читает uint16 big-endian по офсету.
func (r Report) U16BE(offset int) uint16 {
	if offset+1 >= len(r.Data) {
		return 0
	}
	return uint16(r.Data[offset])<<8 | uint16(r.Data[offset+1])
}

// Bit возвращает значение бита n в байте по офсету.
func (r Report) Bit(offset int, bit uint) bool {
	return (r.Byte(offset)>>bit)&1 == 1
}

// BitU16 возвращает значение бита n в uint16 LE по офсету.
func (r Report) BitU16(offset int, bit uint) bool {
	return (r.U16LE(offset)>>bit)&1 == 1
}

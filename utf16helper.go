package utf16helper

import (
	"bufio"
	//	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"unicode/utf16"
	//"unicode/utf8"
	"unsafe"

	"github.com/northbright/byteorder"
)

var (
	UTF16LE  = [2]byte{0xFF, 0xFE}
	UTF16BE  = [2]byte{0xFE, 0xFF}
	UTF8BOM  = [3]byte{0xEF, 0xBB, 0xBF}
	ErrNoBOM = fmt.Errorf("No UTF-16 BOM found")
)

func ReadUTF16BOM(r io.Reader) ([]byte, binary.ByteOrder, error) {
	var buf []byte

	reader := bufio.NewReader(r)

	// Read first 2 bytes.
	for i := 0; i < 2; i++ {
		b, err := reader.ReadByte()
		if err != nil {
			return nil, nil, err
		}
		buf = append(buf, b)
	}

	switch {
	case buf[0] == 0xFF && buf[1] == 0xFE:
		return buf, binary.LittleEndian, nil
	case buf[0] == 0xFE && buf[1] == 0xFF:
		return buf, binary.BigEndian, nil
	default:
		return buf, nil, nil
	}
}

func WriteUTF16BOM(order binary.ByteOrder, dst io.Writer) error {
	var BOM []byte

	switch order {
	case nil:
		return ErrNoBOM
	case binary.LittleEndian:
		BOM = UTF16LE[0:2]
	case binary.BigEndian:
		BOM = UTF16BE[0:2]
	default:
		return ErrNoBOM
	}

	_, err := dst.Write(BOM)
	if err != nil {
		return err
	}
	return nil
}

func WriteUTF8BOM(dst io.Writer) error {
	_, err := dst.Write(UTF8BOM[0:3])
	if err != nil {
		return err
	}
	return nil
}

func RuneToUTF16Bytes(r rune) []byte {
	utf16Buf := utf16.Encode([]rune{r})
	b := (*[2]byte)(unsafe.Pointer(&utf16Buf[0]))
	return b[0:2]
}

func UTF8ToUTF16Ctx(ctx context.Context, src io.Reader, dst io.Writer, outputUTF16BOM bool) error {
	reader := bufio.NewReader(src)
	writer := bufio.NewWriter(dst)

	if outputUTF16BOM {
		if err := WriteUTF16BOM(byteorder.Get(), writer); err != nil {
			return err
		}
	}

	// Read first rune and check if it is UTF-8 BOM.
	r, _, err := reader.ReadRune()
	if err != nil {
		return err
	}
	// If first rune is NOT UTF-8 BOM(0xEF,0xBB,0xBF -> rune: 0xFEFF),
	// convert it to UTF-16 bytes, write the bytes.
	if r != 0xFEFF {
		b := RuneToUTF16Bytes(r)
		if _, err := writer.Write(b); err != nil {
			return err
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		r, _, err := reader.ReadRune()
		if err != nil {
			return err
		}

		b := RuneToUTF16Bytes(r)
		if _, err := writer.Write(b); err != nil {
			return err
		}
	}

	return writer.Flush()
}

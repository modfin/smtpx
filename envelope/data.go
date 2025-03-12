package envelope

import (
	"bytes"
	"io"
)

type Data struct {
	bufs []*bytes.Buffer
}

func (d *Data) head() *bytes.Buffer {
	d.bufs = append([]*bytes.Buffer{bytes.NewBuffer(nil)}, d.bufs...)
	return d.bufs[0]
}

func (d *Data) tail() *bytes.Buffer {
	if len(d.bufs) == 0 {
		d.bufs = append(d.bufs, bytes.NewBuffer(nil))
	}
	return d.bufs[len(d.bufs)-1]
}

func (d *Data) Len() int {
	length := 0
	for _, b := range d.bufs {
		length += b.Len()
	}
	return length
}

func (d *Data) Bytes() []byte {
	result := make([]byte, d.Len())
	off := 0
	for _, b := range d.bufs {
		copy(result[off:off+b.Len()], b.Bytes())
		off += b.Len()
	}
	return result
}

func (d *Data) String() string {
	return string(d.Bytes())
}

func (d *Data) WriteString(s string) (n int, err error) {
	return d.Write([]byte(s))
}

func (d *Data) Write(p []byte) (n int, err error) {
	return d.tail().Write(p)
}

func (d *Data) Prepend(p []byte) (n int, err error) {
	return d.head().Write(p)
}
func (d *Data) PrependString(s string) (n int, err error) {
	return d.head().WriteString(s)
}

func (d *Data) ReadFrom(r io.Reader) (n int64, err error) {
	tail := d.tail()
	return tail.ReadFrom(r)
}

func (d *Data) Reader() io.Reader {
	var readers []io.Reader
	for _, b := range d.bufs {
		readers = append(readers, bytes.NewReader(b.Bytes()))
	}
	return io.MultiReader(readers...)
}

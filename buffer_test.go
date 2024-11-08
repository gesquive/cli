package cli

import "testing"

func Test(t *testing.T) {
	b := newBuffer()
	defer b.Free()
	b.WriteString("hello")
	b.WriteByte(',')
	b.Write([]byte(" world"))
	b.WritePosIntWidth(17, 4)

	got := b.String()
	want := "hello, world0017"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

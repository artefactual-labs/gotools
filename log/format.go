package log

import (
	"io"

	"golang.org/x/term"
)

type fileDescriptorWriter interface {
	Fd() uintptr
}

func resolveFormat(format Format, terminal bool) Format {
	if format != FormatAuto {
		return format
	}
	if terminal {
		return FormatText
	}

	return FormatJSON
}

func writerIsTerminal(w io.Writer) bool {
	f, ok := w.(fileDescriptorWriter)
	return ok && term.IsTerminal(int(f.Fd()))
}

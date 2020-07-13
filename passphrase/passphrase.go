package passphrase

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

func Read(name string) ([]byte, error) {
	var err error
	var tty *os.File
	fd := uintptr(syscall.Stdin)
	if tty, err = os.OpenFile("/dev/tty", os.O_RDWR, 0666); err == nil {
		fd = tty.Fd()
	} else {
		tty = os.Stderr
	}
	fmt.Fprintf(tty, "Please enter your %s: ", name)
	var buf []byte
	if buf, err = terminal.ReadPassword(int(fd)); err != nil {
		return nil, err
	}
	fmt.Fprintln(tty)
	return buf, nil
}

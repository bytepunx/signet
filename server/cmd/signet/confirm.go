package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// confirmDestructive prints warning to out and, unless skip is true, requires
// the operator to type exactly "yes" on in before proceeding. Returns an
// error (without printing anything further) if the operator declines.
//
// Callers pass skip=true when a --yes flag was provided, bypassing the
// interactive prompt for scripted/non-interactive use.
func confirmDestructive(in io.Reader, out io.Writer, warning string, skip bool) error {
	fmt.Fprintln(out, warning)
	if skip {
		return nil
	}
	fmt.Fprint(out, "Type 'yes' to continue: ")
	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		return fmt.Errorf("aborted: no confirmation received")
	}
	if strings.TrimSpace(scanner.Text()) != "yes" {
		return fmt.Errorf("aborted: confirmation not received")
	}
	return nil
}

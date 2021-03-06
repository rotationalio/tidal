package tidal

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

// regular expressions for parsing migration files
var (
	pkgre = regexp.MustCompile(`(?i)^\s*--\s+package:\s+([\w\d\_]+)\s*$`)
	migre = regexp.MustCompile(`(?i)^\s*--\s+migrate:\s+(up|down|end)\s*$`)
)

// NewDescriptor reads the data from the source migration file and gzip compresses it
// for in-memory storage. The reader should not be compressed before hand. Note that
// the name is not optional, it is used to identify descriptors via the gzip header
// information -- autogenerated descriptors use this property to ensure that the
// migrations can be created from a raw descriptor with no other information.
func NewDescriptor(src io.Reader, name string) (_ Descriptor, err error) {
	var (
		buf bytes.Buffer
		zw  *gzip.Writer
	)

	if zw, err = gzip.NewWriterLevel(&buf, gzip.BestCompression); err != nil {
		return nil, err
	}

	// Set the header fields for debugging
	zw.Name = name
	zw.ModTime = time.Now().UTC()

	if _, err = io.Copy(zw, src); err != nil {
		return nil, err
	}

	if err = zw.Close(); err != nil {
		return nil, err
	}

	return Descriptor(buf.Bytes()), nil
}

// Descriptor is the compressed bytes of the encoded SQL file that contains migration
// data. Descriptors are generated by the tidal command and embedded into the source
// code of applications. In order to minimize memory usage and binary size, the data is
// always stored in a compressed format, decompressed as necessary to run migration
// commands. This slows down the migration process a bit, but as migrations are rare, is
// an acceptable trade-off.
type Descriptor []byte

// Info returns header information from the compressed data, generated Descriptors will
// have the associated filename and modification time returned.
func (d Descriptor) Info() (name string, modTime time.Time, err error) {
	var zr *gzip.Reader
	if zr, err = gzip.NewReader(bytes.NewBuffer(d)); err != nil {
		return "", time.Time{}, err
	}
	defer zr.Close()

	return zr.Name, zr.ModTime, nil
}

// Package looks for a package directive, e.g. -- package: foo and returns the name of
// the specified package, otherwise it returns an empty string.
func (d Descriptor) Package() (s string, err error) {
	var zr *gzip.Reader
	if zr, err = gzip.NewReader(bytes.NewBuffer(d)); err != nil {
		return "", err
	}
	defer zr.Close()

	scanner := bufio.NewScanner(zr)
	for scanner.Scan() {
		line := scanner.Text()
		if pkgre.MatchString(line) {
			return pkgre.FindStringSubmatch(line)[1], nil
		}
	}

	return "", scanner.Err()
}

// Up reads and returns the up migration command, including all comments and statements
// following the -- migrate: up comment and before the -- migrate: down or
// --migrate: end comments (or EOF).
func (d Descriptor) Up() (sql string, err error) {
	return d.readBetween("up")
}

// Down reads and returns the down migration command, including all comments and
// statements following the -- migrate: down comment and before the -- migrate: up or
// --migrate: end comments (or EOF).
func (d Descriptor) Down() (sql string, err error) {
	return d.readBetween("down")
}

// Helper function to read the descriptor between the target directive (e.g. up/down)
// and the next directive or end. This function does not handle the case where multiple
// directives of the same name are in consecutive order, with the exception that it does
// omit the directive comments from the returned string.
func (d Descriptor) readBetween(target string) (s string, err error) {
	var zr *gzip.Reader
	if zr, err = gzip.NewReader(bytes.NewBuffer(d)); err != nil {
		return "", err
	}
	defer zr.Close()

	var (
		sb      strings.Builder
		between bool
	)

	scanner := bufio.NewScanner(zr)
	for scanner.Scan() {
		// Read the line and check for a migration directive
		line := scanner.Text()
		if migre.MatchString(line) {
			directive := strings.ToLower(migre.FindStringSubmatch(line)[1])
			between = directive == target
			continue // skip the directive line
		}

		if between {
			// Write the line to the builder, adding back the newlines
			sb.WriteString(line)
			sb.WriteRune('\n')
		}
	}

	return sb.String(), scanner.Err()
}

// Repr returns a string representation of the bytes data for embedding into source code.
// This function is primarily used by the code generation tool.
func (d Descriptor) Repr() string {
	var b strings.Builder
	b.WriteString("[]byte{\n")
	fmt.Fprintf(&b, "\t// %d bytes of compressed tidal.Descriptor data", len(d))

	for i := 0; i < len(d); i += 16 {
		b.WriteString("\n\t")

		j := i + 16
		if j > len(d) {
			j = len(d)
		}

		for k, bit := range d[i:j] {
			fmt.Fprintf(&b, "0x%02x,", bit)
			// avoid spaces at the end of lines
			if k < j-i-1 {
				b.WriteRune(' ')
			}
		}
	}

	b.WriteString("\n}")
	return b.String()
}

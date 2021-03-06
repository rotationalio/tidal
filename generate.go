package tidal

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"text/template"
)

const bindata string = `// Code generated by tidal. DO NOT EDIT.
// source: {{ .Source }}

package {{ .PackageName }}

import "github.com/rotationalio/tidal"

func init() {
	{{- range $varname, $value := .Descriptors }}
	tidal.RegisterDescriptor({{ $varname }})
	{{- end }}
}

{{- range $varname, $value := .Descriptors }}
var {{ $varname }} = {{ $value }}

{{- end }}
`

var bindataTemplate = template.Must(template.New("").Parse(bindata))

// generateContext is used to populate data into the code template.
type generateContext struct {
	Source      string
	PackageName string
	Descriptors map[string]string
}

// Generate code and descriptors to embed migrations into an application package. The
// generate command requires the path to the migrations directory and the location to
// write the generated code file out to. Optionally, a packageName can be supplied,
// otherwise any package directives in the migration files will be used or simply the
// basename of the specified outpath.
func Generate(migrations, outpath, packageName string) (err error) {
	// Find all migration files in the migrations directory and parse them.
	var objs []Migration
	if objs, err = parseMigrations(migrations); err != nil {
		return err
	}

	// Migrations must be sorted, ensure that they are
	sort.Sort(ByRevision(objs))

	// Find the package name if not specified
	if packageName == "" {
		if packageName, err = determinePackage(objs, outpath); err != nil {
			return err
		}
	}

	// Create the code generation context
	ctx := &generateContext{
		Source:      migrations,
		PackageName: packageName,
		Descriptors: make(map[string]string),
	}

	for _, m := range objs {
		key := fmt.Sprintf("revision%d", m.Revision)
		ctx.Descriptors[key] = m.descriptor.Repr()
	}

	// Execute the template
	builder := &bytes.Buffer{}
	if err = bindataTemplate.Execute(builder, ctx); err != nil {
		return err
	}

	// Format the generated code
	var data []byte
	if data, err = format.Source(builder.Bytes()); err != nil {
		return err
	}

	// Create the generated code file
	var f *os.File
	if f, err = os.Create(outpath); err != nil {
		return err
	}
	defer f.Close()

	if _, err = f.Write(data); err != nil {
		return err
	}

	return nil
}

// Find all *.sql files in the specified directory, open them and return the loaded and
// parsed migrations (unregistered, this is separate from the migrations list).
func parseMigrations(dir string) (migrations []Migration, err error) {
	// Find the migration files to generate descriptors from.
	var paths []string
	if paths, err = filepath.Glob(filepath.Join(dir, "*.sql")); err != nil {
		return nil, fmt.Errorf("could not find *.sql files in %q: %s", dir, err)
	}

	if len(paths) == 0 {
		return nil, errors.New("no migrations files found")
	}

	// Parse the migrations from the files
	migrations = make([]Migration, 0, len(paths))
	for _, path := range paths {
		var m Migration
		if m, err = Open(path); err != nil {
			return nil, err
		}
		migrations = append(migrations, m)
	}
	return migrations, nil
}

func determinePackage(migrations []Migration, outpath string) (packageName string, err error) {
	names := make(map[string]bool)
	for _, m := range migrations {
		var name string
		if name, err = m.Package(); err != nil {
			return "", err
		}
		if name != "" {
			names[name] = true
		}
	}

	if len(names) > 1 {
		return "", fmt.Errorf("discovered %d unique package names, please specify package name", len(names))
	}

	if len(names) == 1 {
		for name := range names {
			return name, nil
		}
	}

	if outpath != "" {
		packageName = filepath.Base(filepath.Dir(outpath))
		if packageName == "." {
			if wd, err := os.Getwd(); err == nil {
				return filepath.Base(wd), nil
			}
		} else {
			return packageName, nil
		}
	}

	return "", fmt.Errorf("could not determine package name from %d migrations and %q outpath", len(migrations), outpath)
}

package fsutil_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"go.artefactual.dev/tools/fsutil"
)

func ExampleBaseNoExt() {
	fmt.Println(fsutil.BaseNoExt("/home/dir/archive.tar.gz"))
	fmt.Println(fsutil.BaseNoExt("/home/dir/README.md"))
	fmt.Println(fsutil.BaseNoExt("/home/dir/README"))
	fmt.Println(fsutil.BaseNoExt("/home/dir/"))
	// Output: archive
	// README
	// README
	// dir
}

func TestBaseNoExt(t *testing.T) {
	t.Parallel()

	type test struct {
		name string
		path string
		want string
	}

	for _, tc := range []test{
		{
			name: "Returns a filename with multiple extensions removed",
			path: filepath.Join("home", "dir", "archive.tar.gz"),
			want: "archive",
		},
		{
			name: "Returns a filename with no extension",
			path: filepath.Join("home", "dir", "README"),
			want: "README",
		},
		{
			name: "Returns a directory name",
			path: filepath.Join("home", "dir"),
			want: "dir",
		},
		{
			name: "Returns a single path separator unaltered",
			path: string(filepath.Separator),
			want: string(filepath.Separator),
		},
		{
			name: "Returns a single period unaltered",
			path: ".",
			want: ".",
		},
		{
			name: "Returns a double period unaltered",
			path: "..",
			want: "..",
		},
		{
			name: "Returns a single period when path is empty",
			path: "",
			want: ".",
		},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b := fsutil.BaseNoExt(tc.path)
			assert.Equal(t, b, tc.want)
		})
	}
}

func TestSetFileModes(t *testing.T) {
	td := fs.NewDir(t, "enduro-test-fsutil",
		fs.WithDir("transfer", fs.WithMode(0o755),
			fs.WithFile("test1", "I'm a test file.", fs.WithMode(0o644)),
			fs.WithDir("subdir", fs.WithMode(0o755),
				fs.WithFile("test2", "Another test file.", fs.WithMode(0o644)),
			),
		),
	)

	err := fsutil.SetFileModes(td.Join("transfer"), 0o700, 0o600)
	assert.NilError(t, err)
	assert.Assert(t, fs.Equal(
		td.Path(),
		fs.Expected(t,
			fs.WithDir("transfer", fs.WithMode(0o700),
				fs.WithFile("test1", "I'm a test file.", fs.WithMode(0o600)),
				fs.WithDir("subdir", fs.WithMode(0o700),
					fs.WithFile("test2", "Another test file.", fs.WithMode(0o600)),
				),
			),
		),
	))
}

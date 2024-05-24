package fsutil_test

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	tfs "gotest.tools/v3/fs"

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
	t.Parallel()

	t.Run("Sets file modes", func(t *testing.T) {
		t.Parallel()

		var (
			initDirMode  fs.FileMode = 0o755
			initFileMode fs.FileMode = 0o644
			wantDirMode  fs.FileMode = 0o700
			wantFileMode fs.FileMode = 0o600
		)

		td := tfs.NewDir(t, "enduro-test-fsutil",
			tfs.WithDir("transfer", tfs.WithMode(initDirMode),
				tfs.WithFile("test1", "I'm a test file.", tfs.WithMode(initFileMode)),
				tfs.WithDir("subdir", tfs.WithMode(initDirMode),
					tfs.WithFile("test2", "Another test file.", tfs.WithMode(initFileMode)),
				),
			),
		)

		err := fsutil.SetFileModes(td.Join("transfer"), wantDirMode, wantFileMode)
		assert.NilError(t, err)
		assert.Assert(t, tfs.Equal(
			td.Path(),
			tfs.Expected(t,
				tfs.WithDir("transfer", tfs.WithMode(wantDirMode),
					tfs.WithFile("test1", "I'm a test file.", tfs.WithMode(wantFileMode)),
					tfs.WithDir("subdir", tfs.WithMode(wantDirMode),
						tfs.WithFile("test2", "Another test file.", tfs.WithMode(wantFileMode)),
					),
				),
			),
		))
	})
}

package fsutil_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"go.artefactual.dev/tools/fsutil"
)

var Renamer = os.Rename

var dirOpts = []fs.PathOp{
	fs.WithDir(
		"child1",
		fs.WithFile(
			"foo.txt",
			"foo",
		),
	),
	fs.WithDir(
		"child2",
		fs.WithFile(
			"bar.txt",
			"bar",
		),
	),
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
			b := fsutil.BaseNoExt(tc.path)
			assert.Equal(t, b, tc.want)
		})
	}
}

func TestMove(t *testing.T) {
	t.Parallel()

	t.Run("It fails if destination already exists", func(t *testing.T) {
		t.Parallel()

		tmpDir := fs.NewDir(t, "enduro")
		fs.Apply(t, tmpDir, fs.WithFile("foobar.txt", ""))
		fs.Apply(t, tmpDir, fs.WithFile("barfoo.txt", ""))

		src := tmpDir.Join("foobar.txt")
		dst := tmpDir.Join("barfoo.txt")
		err := fsutil.Move(src, dst)

		assert.Error(t, err, "destination already exists")
	})

	t.Run("It moves files", func(t *testing.T) {
		t.Parallel()

		tmpDir := fs.NewDir(t, "enduro")
		fs.Apply(t, tmpDir, fs.WithFile("foobar.txt", ""))

		src := tmpDir.Join("foobar.txt")
		dst := tmpDir.Join("barfoo.txt")
		err := fsutil.Move(src, dst)

		assert.NilError(t, err)

		_, err = os.Stat(src)
		assert.ErrorIs(t, err, os.ErrNotExist)

		_, err = os.Stat(dst)
		assert.NilError(t, err)
	})

	t.Run("It moves directories", func(t *testing.T) {
		t.Parallel()

		tmpSrc := fs.NewDir(t, "enduro", dirOpts...)
		src := tmpSrc.Path()
		srcManifest := fs.ManifestFromDir(t, src)
		tmpDst := fs.NewDir(t, "enduro")
		dst := tmpDst.Join("nested")

		err := fsutil.Move(src, dst)

		assert.NilError(t, err)
		_, err = os.Stat(src)
		assert.ErrorIs(t, err, os.ErrNotExist)
		assert.Assert(t, fs.Equal(dst, srcManifest))
	})

	t.Run("It copies directories when using different filesystems", func(t *testing.T) {
		fsutil.Renamer = func(src, dst string) error {
			return &os.LinkError{
				Op:  "rename",
				Old: src,
				New: dst,
				Err: errors.New("invalid cross-device link"),
			}
		}
		t.Cleanup(func() {
			fsutil.Renamer = os.Rename
		})

		tmpSrc := fs.NewDir(t, "enduro", dirOpts...)
		src := tmpSrc.Path()
		srcManifest := fs.ManifestFromDir(t, src)
		tmpDst := fs.NewDir(t, "enduro")
		dst := tmpDst.Join("nested")

		err := fsutil.Move(src, dst)

		assert.NilError(t, err)
		_, err = os.Stat(src)
		assert.ErrorIs(t, err, os.ErrNotExist)
		assert.Assert(t, fs.Equal(dst, srcManifest))
	})
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

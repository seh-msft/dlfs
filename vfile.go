// Copyright (c) 2020 Microsoft Corporation, Sean Hinchee.
// Licensed under the MIT License.

// These are special overrides to make styx interface casting work correctly
// The only important bit is that these interfaces need non-pointer receivers
// As such, we store a reference to a File and implement non-pointer receivers
// This simplifies the implementation significantly from the File side
// See: https://play.golang.org/p/Au12mZbdiHX
package main

import (
	"os"
	"time"
)

// Virtual file wrapper for 9p operations on a File
type VFile struct {
	*File
}

// Uid
func (vf VFile) Uid() string {
	return vf.File.Uid()
}

// Gid
func (vf VFile) Gid() string {
	return vf.File.Gid()
}

// Muid
func (vf VFile) Muid() string {
	return vf.File.Muid()
}

// Close file
func (vf VFile) Close() error {
	return vf.File.Close()
}

// Write from a certain offset - not called for directories
func (vf VFile) WriteAt(p []byte, off int64) (int, error) {
	return vf.File.WriteAt(p, off)
}

// Read from a certain offset - not called for directories
func (vf VFile) ReadAt(p []byte, offset int64) (int, error) {
	return vf.File.ReadAt(p, offset)
}

// Returns the singleton name of the file `/foo/bar` is `bar`
func (vf VFile) Name() string {
	return vf.File.Name()
}

// Returns the size of the file contents
func (vf VFile) Size() int64 {
	return vf.File.Size()
}

// Returns the permission bits (uint32)
func (vf VFile) Mode() os.FileMode {
	return vf.File.Mode()
}

// Returns the time of the last modification of the file
func (vf VFile) ModTime() time.Time {
	return vf.File.ModTime()
}

// Returns "the underlying data source"
func (vf VFile) Sys() interface{} {
	return vf.File.Sys()
}

// Returns the info that styx wants
func (vf VFile) Stat() os.FileInfo {
	return vf.File.Stat()
}

// If we are a directory, avoid calling ReadAt()?
func (vf VFile) Readdir(n int) ([]os.FileInfo, error) {
	return vf.File.Readdir(n)
}

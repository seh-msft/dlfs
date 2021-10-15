// Copyright (c) 2020 Microsoft Corporation, Sean Hinchee.
// Licensed under the MIT License.

// Contains Azure blob information for use with File
package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"time"

	"github.com/Azure/azure-pipeline-go/pipeline"
	"github.com/Azure/azure-storage-file-go/azfile"
)

const (
	// See: https://godoc.org/github.com/Azure/azure-storage-blob-go/azblob#UploadStreamToBlockBlob
	maxBuffers = 3               // Max # rotating buffers for upload
	bufSize    = 2 * 1024 * 1024 // Rotating buffer size for upload
	maxRetry   = 20              // Maximum number of retries for download
)

// Tracks a blob and its state
type Blob struct {
	// TODO - way to check for changes in Azure
	name    *string             // Ref to File.name
	last    time.Time           // Time last accessed by us
	body    *bytes.Buffer       // Bytes contents of file
	isDir   bool                // Are we a directory?	TODO - should this be a ptr into the file?
	tracked bool                // Are we tracking this for synchronization? (were we walked?)
	fileURL azfile.FileURL      // Azure file object
	dirURL  azfile.DirectoryURL // Azure directory object
	parent  azfile.DirectoryURL // Azure directory object parent
}

// Self-delete a blob
// TODO - return more?
func (b *Blob) Delete(ctx context.Context) error {
	if b.isDir {
		_, err := b.dirURL.Delete(ctx)
		return err
	}
	// Empty files can't be uploaded in the first place
	// TODO - check if exists remotely first?
	if len(b.body.Bytes()) < 1 {
		return nil
	}
	_, err := b.fileURL.Delete(ctx)
	return err
}

// List blobs 'in' the current directory, divided into files and sub-directories
func Ls(ctx context.Context, rootURL azfile.DirectoryURL) (files, dirs []string) {
	for marker := (azfile.Marker{}); marker.NotDone(); {
		// Get a result segment starting with the file indicated by the current Marker.
		listResponse, err := rootURL.ListFilesAndDirectoriesSegment(ctx, marker, azfile.ListFilesAndDirectoriesOptions{})
		if err != nil {
			log.Fatal(err)
		}

		// For next iteration, advance the marker
		marker = listResponse.NextMarker

		// Files
		for _, fileEntry := range listResponse.FileItems {
			files = append(files, fileEntry.Name)
		}

		// Directories
		for _, directoryEntry := range listResponse.DirectoryItems {
			dirs = append(dirs, directoryEntry.Name)
		}
	}

	return files, dirs
}

// Create a new blob
func NewBlob(name *string, parent azfile.DirectoryURL, isDir bool) *Blob {
	var fileURL azfile.FileURL
	var dirURL azfile.DirectoryURL
	if isDir {
		fileURL = parent.NewFileURL(*name)
	} else {
		dirURL = parent.NewDirectoryURL(*name)
	}

	return &Blob{
		parent:  parent,
		name:    name,
		last:    time.Now(),
		isDir:   isDir,
		fileURL: fileURL,
		dirURL:  dirURL,
		body:    &bytes.Buffer{},
	}
}

// Return the contents of the body buffer
func (b Blob) Contents() []byte {
	// TODO - sync with Azure to verify state?
	return b.body.Bytes()
}

// Upload a blob in full
func (b *Blob) Upload(ctx context.Context) error {
	log.Println("!!!! UPLOADING ", *b.name)
	size := int64(len(b.body.Bytes()))
	if size < 1 && !b.isDir {
		// Can't upload empty files
		return nil
	}

	if b.isDir {
		_, err := b.dirURL.Create(ctx, azfile.Metadata{}, azfile.SMBProperties{})
		// Check if exists remotely?
		return err
	}

	// Trigger a create
	_, err := b.fileURL.Create(ctx, size, azfile.FileHTTPHeaders{ContentType: "text/plain"}, azfile.Metadata{})
	if err != nil {
		// Check azfile.ServiceCodeResourceAlreadyExists ?
		return err
	}

	_, err = b.fileURL.UploadRange(ctx, 0, bytes.NewReader(b.body.Bytes()), nil)

	return err
}

// Download a blob in full
func (b *Blob) Download(ctx context.Context) error {
	if b.isDir {
		return nil
	}
	log.Println("!!!! DOWNLOADING", *b.name)
	resp, err := b.fileURL.Download(context.Background(), 0, azfile.CountToEnd, false)
	if err != nil {
		return errors.New("file download failed → " + err.Error())
	}

	contentLength := resp.ContentLength()
	opts := azfile.RetryReaderOptions{MaxRetryRequests: 3}
	retryReader := resp.Body(opts)

	// NewResponseBodyStream wraps the RetryReader with progress reporting; it returns an io.ReadCloser.
	progressReader := pipeline.NewResponseBodyProgress(retryReader,
		func(bytesTransferred int64) {
			log.Printf("!!!!» Downloaded %d of %d bytes.\n", bytesTransferred, contentLength)
		})
	defer progressReader.Close() // The client must close the response body when finished with it

	// Copy the body into a buffer
	log.Println("! readfrom")
	b.body.Reset()
	written, err := io.CopyN(b.body, progressReader, contentLength) // Write to the file by reading from the file (with intelligent retries).
	if err != nil {
		return errors.New("copy to body failed → " + err.Error())
	}

	log.Println("!!!!» Copied: ", written)

	return err
}

// Acquire information about a blob
func (b *Blob) Stat() error {
	/*
		ctx := context.Background()
		resp, err := b.fileURL.GetProperties(ctx, azblob.BlobAccessConditions{}, azblob.ClientProvidedKeyOptions{})

		log.Println("» Stat Resp → ", *resp.Response(), " err → ", err)
		log.Println("» Type → ", resp.BlobType())
	*/

	// TODO - make some kind of FileInfo or similar?
	return nil
}

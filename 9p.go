// Copyright (c) 2020 Microsoft Corporation, Sean Hinchee.
// Licensed under the MIT License.

// 9p server-specific functionality
package main

import (
	"context"
	"log"
	"path"

	"aqwari.net/net/styx"
	"github.com/Azure/azure-storage-file-go/azfile"
)

// 9p server container - implements interfaces for styx
type Server struct {
	*File
	share azfile.ShareURL
	svc   azfile.ServiceURL
	ctx   context.Context
}

// Init the server and its file system - call only once
func (s *Server) Initialize() {
	s.File = NewTree(s)
}

// Look up a file by path string
func lookup(srv Server, full string) (*File, error) {
	// Sync root
	srv.File.Sync()

	cleaned := path.Clean(full)

	// Short circuit base case for root
	if cleaned == "/" {
		return srv.File, nil
	}

	f, err := srv.File.Search(cleaned)
	if f == nil {
		return srv.File, err
	}

	return f, err
}

// Handle 9p requests to the server - each new connection will call this
func (srv *Server) Serve9P(s *styx.Session) {
Loop:
	for s.Next() {
		msg := s.Request()
		file := path.Clean(msg.Path())
		log.Println("Handling: ", file)

		// Switch on the kind of message we are receiving, not all will arrive here and are handled by styx
		// Only every give styx a VFile to ensure it can cast interfaces correctly
		switch t := msg.(type) {
		case styx.Twalk:
			log.Println("=== walk: ", t)
			f, err := lookup(*srv, file)
			if err != nil {
				t.Rerror("tree walk failed → %s", err)
			}
			if f.IsDir() {
				err = f.LoadChildren()
			}
			t.Rwalk(f.VF(), err)

		case styx.Topen:
			log.Println("=== open: ", t)
			f, err := lookup(*srv, file)
			t.Ropen(f.VF(), err)

		case styx.Tstat:
			log.Println("=== stat: ", t)
			f, err := lookup(*srv, file)
			t.Rstat(f.VF(), err)

		case styx.Tcreate:
			log.Println("=== create: ", t)
			// TODO - something special for directories?
			full := file + t.Name

			// Insert into file tree
			f, err := srv.File.Insert(full, false)
			if err != nil {
				t.Rerror("tree insert failed %s", err)
				continue Loop
			}

			// Upload to blob storage
			err = f.Blob.Upload(srv.ctx)
			if err != nil {
				t.Rerror("azure upload failed %s", err)
				continue Loop
			}

			t.Rcreate(f.VF(), nil)

		case styx.Tremove:
			log.Println("=== rm: ", t)
			full := t.Path()
			f, err := lookup(*srv, full)

			// Delete from blob storage
			// TODO - verify delete snapshot options
			err = f.Blob.Delete(srv.ctx)
			if err != nil {
				t.Rerror("azure delete failed %s", err)
			}

			// Delete from file tree
			err = srv.File.Delete(full)

			t.Rremove(err)

		case styx.Ttruncate:
			// TODO
			log.Println("=== truncate?: ", t)

		case styx.Tutimes:
			// Change last modified time
			log.Println("=== utimes?: ", t)
			// TODO
			t.Rutimes(nil)

		}
	}
}

// Logger handler for 9p requests?
var logger styx.HandlerFunc = func(s *styx.Session) {
	for s.Next() {
		if *verbose {
			log.Printf("%q %T %s", s.User, s.Request(), s.Request().Path())
		}
	}
}

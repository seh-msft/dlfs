// Copyright (c) 2021 Microsoft Corporation, Sean Hinchee.
// Licensed under the MIT License.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"aqwari.net/net/styx"
	"github.com/Azure/azure-storage-file-go/azfile"
)

const (
	maxBlobs = 4096 // Maximum number of blobs to track
)

var (
	//announce      = flag.String("a", "tcp!localhost!1337", "Dialstring to announce on") // TODO
	shareName = flag.String("fileshare", "dlfsfs", "Name of file share to fs-ify")
	port      = flag.String("p", ":1337", "TCP port to listen for 9p connections")
	chatty    = flag.Bool("D", false, "Chatty 9p tracing")
	verbose   = flag.Bool("V", false, "Verbose 9p error output")
)

// A 9p file server exposing an azure blob container
func main() {
	flag.Parse()

	var (
		styxServer styx.Server // 9p file server handle for styx
		srv        Server      // Our file system server
	)

	srv.Initialize()

	log.Printf("Using %s as the file share for the fs...\n", *shareName)

	/* Set up Azure */

	// Acquire azure credential information from environment variables
	accountName := os.Getenv("DLSA")
	accountKey := os.Getenv("DLKEY")
	if len(accountName) == 0 || len(accountKey) == 0 {
		fatal("$DLSA and $DLKEY environment variables must be set to authenticate")
	}

	// Create a new azure auth pipeline
	credential, err := azfile.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		fatal("err: could not authenticate → ", err)
	}
	p := azfile.NewPipeline(credential, azfile.PipelineOptions{})

	/* Set up the storage account's file share */

	urlStr, err := url.Parse(fmt.Sprintf("https://%s.file.core.windows.net", accountName))
	if err != nil {
		fatal("err: could not generate URL string → ", *urlStr, err)
	}

	svcURL := azfile.NewServiceURL(*urlStr, p)
	ctx := context.Background()

	// TODO - is sharename correct here?
	shareURL := svcURL.NewShareURL(*shareName)

	srv.share = shareURL
	srv.svc = svcURL
	srv.ctx = ctx

	// Create the share on the service
	_, err = shareURL.Create(ctx, azfile.Metadata{}, 0)
	exists := false

	if err != nil {
		// We have to search the error for the magic response string
		if strings.Contains(err.Error(), string(azfile.ServiceCodeShareAlreadyExists)) {
			exists = true
		}

		// The share didn't exist, but we couldn't create it
		if !exists {
			fatal("err: could not create share → ", err)
		}
	}

	if exists {
		log.Println(`File share "` + *shareName + `" found, using…`)
	} else {
		log.Println(`No existing share "` + *shareName + `", creating…`)
	}

	/* Populate tree with contents from the share */

	var rootURL azfile.DirectoryURL
	var files, dirs []string
	rootStr := "/"

	// Skip population if the container didn't exist, there's nothing contained
	if !exists {
		goto Styx
	}

	log.Println("Reading existing files from share…")

	// Root directory URL for the share
	rootURL = shareURL.NewRootDirectoryURL()
	srv.Blob = &Blob{
		name:   &rootStr,
		isDir:  true,
		dirURL: rootURL,
	}

	log.Println("¡ ROOTURL = ", rootURL)

	/* List file and directories */

	files, dirs = Ls(srv.ctx, rootURL)
	for _, file := range files {
		log.Println("Found:", file)
		log.Println("Before insert:\n", srv)
		f, err := srv.Insert("/"+file, false)
		if err != nil {
			fatal("err: could not insert extant blob file into fs → ", err)
		}
		log.Println("After insert:\n", srv)

		err = f.Blob.Download(srv.ctx)
		if err != nil {
			fatal("err: could not insert download extant blob file into fs → ", err)
		}
	}
	for _, dir := range dirs {
		log.Println("Found:", dir+"/")
		d, err := srv.Insert("/"+dir, true)
		if err != nil {
			fatal("err: could not insert extant blob dir into fs → ", err)
		}

		// Redundant?
		// TODO - populate children?
		err = d.Blob.Download(srv.ctx)
		if err != nil {
			fatal("err: could not insert download extant blob dir into fs → ", err)
		}
	}

	log.Println("Finished loading extant files…")

	/* Set up 9p server */
Styx:

	if *chatty {
		styxServer.TraceLog = log.New(os.Stderr, "", 0)
	}
	if *verbose {
		styxServer.ErrorLog = log.New(os.Stderr, "", 0)
	}

	// TODO - actually parse dial string (new module?)
	// TODO - allow options like /srv posting, unix socket, etc.
	//proto, addr, port := dialstring.Parse(*announce)
	styxServer.Addr = *port

	// Shim our own logger, in case we need it
	styxServer.Handler = styx.Stack(logger, &srv)

	log.Println("Listening on tcp!127.0.0.1!" + (*port)[1:] + " …")
	fatal(styxServer.ListenAndServe())
}

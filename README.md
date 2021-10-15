# dlfs

Azure storage account (data lake) v2 as a 9p file system.

Fork of [abfs](https://github.com/seh-msft/abfs).

Written in [Go](https://golang.org).

Created during the 2021 MS hackathon.

## Build

Currently the `go.mod` file looks for a `../styx` directory as there is a bug in the stock `styx` package.

[A PR has been opened.](https://github.com/droyo/styx/pull/30)

The fix is in the `rwalk_fix` branch of <https://github.com/seh-msft/styx>. 

	; git clone -b rwalk_fix https://github.com/seh-msft/styx ../styx
	; go build

## Requirements

What you need:

1. Azure Storage Account configured for Data Lake v2
2. A file in said storage account
	a. Should have a url such as: https://seandl.file.core.windows.net/dlfsfs

## Usage

Invocation:

```
; ./dlfs -h
Usage of dlfs:
  -D	Chatty 9p tracing
  -V	Verbose 9p error output
  -fileshare string
    	Name of file share to fs-ify (default "dlfsfs")
  -p string
    	TCP port to listen for 9p connections (default ":1337")
;
```

Under the [Inferno](https://bitbucket.org/inferno-os/inferno-os) fork [Purgatorio](http://git.9front.org/plan9front/purgatorio/HEAD/info.html):

```
; mount -Ac tcp!127.0.0.1!1337 /n/dlfs
; lc
sam.bat*   somedir/   something* test*
; cat something
hello
; echo more >> something
; touch newthing
; echo 'hi' >> newthing
; rm newthing
; cd somedir
; lc
cd.ps1*
; cat cd.ps1
if($Args.Count -lt 1) {
        Set-Location $HOME
} else {
        $params = $Args[0]
        Set-Location $params
}
; cd ..
; lc
sam.bat*   somedir/   something* test*
; touch foo
; echo bar >> foo
; ls
sam.bat
somedir
something
test
; ls
sam.bat
somedir
something
test
;
```

Under 9pfuse:

```
sehinche$ 9pfuse 'tcp!127.0.0.1!1337' ~/n/dlfs
sehinche$ cd ~/n/dlfs
ls
dlfs$ ls
sam.bat  somedir  something  test
dlfs$ touch hi
touch: cannot touch 'hi': Numerical result out of range
dlfs$ lc
sam.bat*   somedir/   something* test*
dlfs$ ls
sam.bat  somedir  something  test
dlfs$ cat test
foo
bar
whatever
dlfs$ echo blah >> test
dlfs$ cat test
foo
bar
whatever
blah
dlfs$ ls
sam.bat  somedir  something  test
dlfs$ ls
foo  sam.bat  somedir  something  test
dlfs$ cat foo
bar
dlfs$ rm foo
dlfs$ ls
ls: cannot access 'foo': Numerical result out of range
foo  sam.bat  somedir  something  test
dlfs$ ls
sam.bat  somedir  something  test
dlfs$
```

## Install

	; go install

## Bugs

- Directory info is stuttered on Close() calls on the directory as race conditions can arise
	- Grounded in the design of the 'styx' module
	- Results update on second call of ls(1) if a new file is created
- New file creation does not work under 9pfuse
	- Not a dlfs bug as far as I can tell
- Output is very verbose
- Every user is `none`

## Reference

- [Azure File Storage Go Doc](https://pkg.go.dev/github.com/Azure/azure-storage-file-go/azfile)
- [abfs](https://github.com/seh-msft/abfs)

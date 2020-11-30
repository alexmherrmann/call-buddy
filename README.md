# Terminal Call-Buddy

![Call-Buddy Mascot Banner](resources/gopher-banner.png)

`call-buddy` is an interactive HTTP terminal application targeted towards helping make or debug RESTful services. It can be seamlessly launched up into remote machines with a helper tool called `tcb`, no proxy or setup required, allowing you to interact with your HTTP endpoints in their native network environment.

## What does it do?

- Can make and debug templated (think Postman) HTTP calls with little fuss. 

- Store call templates and variables, consistent with multiple remote and local invocations.

- No lock in to any central syncing server, built to check into git.

- Out-of-the-box knowledge of bastion servers, Kubernetes/Docker networks. Helpful when dealing with internal services like Elasticsearch, CouchDB, or any HTTP service.

- Has a fun TUI that runs wherever you have a TTY with light resource usage.

## Complilation

Compilation requires go 1.13+.

```sh
PREFIX=/usr/local
mkdir -p "$PREFIX/src" && cd "$PREFIX/src"
VERSION=0.1.0
wget "https://github.com/call-buddy/call-buddy/archive/v$VERSION.tar.gz"
tar -xzf "call-buddy-$VERSION.tar.gz"
cd "call-buddy-$VERSION"
make
make install prefix=$PREFIX
```

## Launching cross-compiled binaries

Call-Buddy features a neat script called `tcb` that uses cross-compiled binaries at install time to launch `call-buddy` into virtually any UNIX environment. Common targets such as x86/x86_64/arm/arm64 Linux, x86_64 MacOS, and x86/x86_64 FreeBSD binaries are precompiled by default. If you have a environment you use that isn't covered by [this list](arch.txt) (or your particular machine has a uncommon `uname`) and [Go can cross-compile to it](https://gist.github.com/asukakenji/f15ba7e588ac42795f421b48b8aede63), you can modify [arch.txt](arch.txt) to and recompile to be able to launch into your particular environment.

## Documentation

The `call-buddy` and `tcb` commands have man pages.

```roff
call-buddy(1)                  Call-Buddy Manual                 call-buddy(1)

NAME
       call-buddy - interactive HTTP caller

SYNOPSIS
       call-buddy [-e env-file]

DESCRIPTION
       call-buddy  is  an interactive HTTP terminal application, often used to
       debug or test RESTful endpoints.

       The terminal user interface takes in  commands  to  issue  HTTP  calls,
       manipulate  those responses or to modify call-buddy's environment vari-
       ables, profiles or windows. Response headers and bodies can be modified
       in window pane editors.

       The  'help' command inside call-buddy should be used for internal docu-
       mentation on commands and use.

OPTIONS
       -e file
              Environment file to load into the internal "Home" environment.

FILES
       ~/.call-buddy/state-*.json
              State files corresponding  to  different  profiles  created.  If
              $XDG_HOME_DIR    is    set,    state   files   are   stored   in
              $XDG_HOME_DIR/.call-buddy. These files should only  be  modified
              using call-buddy.

ENVIRONMENT
       The  environment  in  which  call-buddy is invoked in is loaded into an
       internal 'Vars' environment. These variables can  be  accessed  in  the
       HTTP request headers and body using the '{{Vars.NAME}}' syntax.

SEE ALSO
       tcb(1), curl(1), wget(1)

BUGS
       Small terminal sizes may cause an ugly fatal crash.

AUTHOR
       Written by the Terminal Call-Buddy team at the University of Utah: Alex
       Herrmann, Cooper Pender, Derek Dixon and Dylan Gardner

v0.1.0                            2020-11-23                     call-buddy(1)
```

```roff
tcb(1)                         Call-Buddy Manual                        tcb(1)

NAME
       tcb - remote call-buddy launcher using SSH

SYNOPSIS
       tcb [user@]host [file ...]

DESCRIPTION
       tcb  is a script for launching call-buddy into remote machines over SSH
       using precompiled architecture-specific call-buddy  binaries  installed
       at  compile-time.  This  facilitates  remote HTTP debugging and testing
       without proxies or remote daemons. Files created in the remote environ-
       ment  can  be synced down to the current directory if the rsync utility
       is installed on both endpoints. The state of the program (call history,
       environment)  is  also  synced  with  the launching machine if rsync is
       installed. The launching OS environment is accessible via  call-buddy's
       'Home' environment.

       Note  that  this  script  currently requires that the remote target has
       passwordless SSH set up and enabled to work.

OPTIONS
       There are no options for tcb.

FILES
       ~/.call-buddy/state-*.json
              State files corresponding to  different  profiles  created  that
              will  be  synced  to and from the remote environment if rsync is
              installed.  If $XDG_HOME_DIR is set, state files are  stored  in
              $XDG_HOME_DIR/.call-buddy.  These  files should only be modified
              using call-buddy.

ENVIRONMENT
       TCB_ARCH_DIR
              The location of architecture-specific  call-buddy  binaries.  If
              this  is  not  present, prefix/lib/call-buddy is attempted to be
              used where prefix corresponds to  the  prefix  location  of  the
              call-buddy binary found. e.g. /usr/local/lib/call-buddy

SEE ALSO
       call-buddy(1), ssh(1), rsync(1)

BUGS
       Files  launched  into  the remote environment will implicitly be synced
       down to the current directory, regardless of whether they  are  updated
       in  the  remote environment as a side effect of the syncing functional-
       ity.

AUTHOR
       Written by Dylan Gardner as part of the Terminal Call-Buddy team at the
       University of Utah.
      
v0.1.0                            2020-11-23                            tcb(1)
```

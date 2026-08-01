# CI

## Why the workflow grants six permissions

`wow-look-at-my/go-toolchain@v1` fails the build outright without each of
these -- none degrades to a warning:

| permission | what needs it |
|---|---|
| `id-token: write` | OIDC, for secret-server and the buildhost autorelease |
| `contents: write` | the dependency-graph submission (403, and a failed build, under `read`) |
| `actions: read` | its no-all-builds guard scans the run's jobs |
| `checks: read` | the same guard reads the head commit's check runs |
| `deployments: write` | autorelease registers a GitHub Deployment as part of publishing |
| `artifact-metadata: write` | autorelease posts an artifact storage record, with no opt-out |

A job-level `permissions:` block REPLACES the workflow-level one rather than
adding to it, so any job that declares its own must restate everything it
needs. The `docker` job does exactly that: it narrows to `contents: read` +
`packages: write` for the registry push and re-adds `actions: read`, which
`cache-download` needs to list this run's hand-offs.

## Pin the action to `@v1`

`@latest` and `@master` are frozen orphan tags (2026-05-05 and 2026-04-12);
the default branch is `v1`. An unrecognized `with:` key is a **warning, not an
error**, so a workflow pinned to a frozen tag silently drops every input added
since -- the failure looks like the input having no effect, with a green build.

## How the binary reaches the docker job

There is no `actions/upload-artifact` step. The org's Actions artifact storage
is out of quota: uploads report success and the download then 404s, which is
why every repo here moved to the org cache transport. go-toolchain hands its
`build/` outputs off automatically on every run, and the `docker` job restores
them with `wow-look-at-my/actions@cache-download#latest`.

Two details bite:

- **Name the hand-off.** go-toolchain saves both a bare `go-build` and a
  job-scoped `go-build-<job>` per run. A nameless `cache-download` discovers
  the run's entries and refuses to pick between two, failing with
  `2 distinct hand-offs`. Naming `go-build-build-and-test` is an exact key
  match for this run and job.
- **Artifacts carry platform suffixes.** The build is a cross-compile matrix,
  so `build/` holds `frontend_linux_amd64`, `frontend_darwin_arm64`,
  `frontend_windows_amd64.exe` and so on -- there is no bare `frontend`. The
  Dockerfile COPYs `build/frontend`, so the job renames the linux/amd64
  artifact into place. (The old artifact upload named a single file
  `frontend`, which is why the Dockerfile expects that name.)

## Why the image is not an Actually Portable Executable

An APE would be one binary for every platform, and go-toolchain builds one on
request (`targets: cosmo`). This program cannot be built that way today --
four of its dependencies have no `GOOS=cosmo` port:

    logrus/terminal_check_notappengine.go  undefined: isTerminal
    grpc/internal/tcp_keepalive_unix.go    undefined: unix.SetsockoptInt, unix.SOL_SOCKET, unix.SO_KEEPALIVE
    in-toto-golang/in_toto/util_unix.go    undefined: unix.Access, unix.W_OK
    containerd/continuity/sysx             undefined: syscall.ENOATTR

Three of the four are `golang.org/x/sys/unix`, which has no cosmo port at all;
go-toolchain carries `src/compat/go-isatty/isatty_cosmo.go` as a shim for
exactly this class of breakage. Making this an APE means porting the
`x/sys/unix` surface these packages use, in gosmopolitan -- not a change to
this repo.

There is a second, independent blocker. An APE is `MZ`-headed, and `execve(2)`
accepts only `\x7fELF`, a native Mach-O, or a `#!` line, so the kernel refuses
to load one directly; what makes an APE runnable is a shell prologue that
rewrites the file's own header in place on first run. A `FROM scratch` image
has no shell, so a container runtime `execve`ing `/bin/frontend` would fail
with `ENOEXEC`. Shipping an APE here would additionally mean assimilating it
in a builder stage that does have a shell, and copying the resulting native
binary into the scratch stage -- at which point the file in the image is no
longer an APE.

(The refusal-then-assimilation behavior is pinned upstream by gosmopolitan's
`testdata/ape/apetest/execve_test.go`.)

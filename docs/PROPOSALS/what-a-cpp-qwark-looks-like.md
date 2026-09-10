# What a C++ qwark looks like

**Status: assessment, asked for 2026-09-09. Nothing is decided and the Rust
ruling stands until it is changed.** Written because the estate records an
assessment whether or not it changes the answer, the way
`wrench/docs/DECISIONS/a-zig-pack-was-assessed-and-declined.md` does.

The Rust ruling turned on three things: parser interop, the containment
primitive, and availability. C++ wins the first two more cleanly than Rust does
and loses on a fourth that the Rust ruling never had to consider.

## The shape of the program

Four dependencies, and two of them are already how tree-sitter is written.

    tree-sitter        C. `#include <tree_sitter/api.h>`, and that is all
    tree-sitter-bash   C. one generated parser.c, compiled in
    nlohmann/json      the hook contract on stdin and stdout
    yaml-cpp           the rule files

### The parser is a plain include

This is the argument that moved qwark off Go, and C++ takes it further than
Rust does.

    extern "C" const TSLanguage *tree_sitter_bash(void);

    TSParser *p = ts_parser_new();
    ts_parser_set_language(p, tree_sitter_bash());
    TSTree *t = ts_parser_parse_string(p, nullptr, src.data(), src.size());
    TSNode root = ts_tree_root_node(t);

No bindgen, no `-sys` crate, no build script vendoring C and driving `cc`. Rust
reaches the same place through the `tree-sitter` and `tree-sitter-bash` crates,
which compile that same C. **The C is not avoided in either language**, and that
matters again below.

> **Corrected 2026-09-09 by building it. The generated parser is not valid
> C++.** `parser.c` uses out-of-order designated initializers and `scanner.c`
> assigns `void*` to a typed pointer; both are legal C11, neither compiles with
> `g++`, and the errors are `sorry, unimplemented` and `invalid conversion`.
> The runtime also calls `fdopen`, so it wants `-std=gnu11` rather than
> `-std=c11`.
>
> So the C is compiled as C and linked into the C++ program:
>
>     gcc -std=gnu11 -O2 -c tree-sitter/lib/src/lib.c ...
>     gcc -std=gnu11 -O2 -c tree-sitter-bash/src/parser.c ...
>     g++ -std=c++20 -O2 -c ts-ast.cpp ...
>     g++ -o ts-ast ts-ast.o lib.o parser.o scanner.o
>
> Four lines in a build file instead of one, and the `extern "C"` declaration
> above is unchanged. It is still simpler than a crate with a build script, and
> it is not the single `#include` this section implied. Worth the correction
> because the whole point of writing it was to find out.

### Containment is the kernel call, not a wrapper around it

`silo/docs/DECISIONS/what-language-each-component-is-written-in.md` states the
cost of leaving Go as: `cap-std` is a dependency where `os.Root` is standard
library, for a tool whose whole job is containment.

**C++ does not pay that cost, and it is worth being precise about why.**
`os.Root` and `cap-std` are both wrappers over one Linux syscall:

    struct open_how how = {
        .flags   = O_PATH | O_CLOEXEC,
        .resolve = RESOLVE_BENEATH | RESOLVE_NO_SYMLINKS | RESOLVE_NO_MAGICLINKS,
    };
    int fd = syscall(SYS_openat2, root_fd, rel, &how, sizeof how);

That is the containment. Symlinks pointing out, `..` walking up and absolute
paths are refused by the kernel, which is the property
`go-because-of-os-root.md` picked Go for. In C++ it is thirty lines calling the
kernel directly, with no supply chain at all, which is a stronger version of the
argument that chose Go rather than a weaker one.

`openat2` is Linux 5.6 and up. This machine is 6.12.107. Not portable, and qwark
is not a portable program.

### The tree can be mmapped instead of parsed

A `PreToolUse` hook is a fresh process per tool call, so nothing amortises and
every millisecond of load is paid on every command. The phase-two tree is
read-only after load, which means it does not have to be YAML at judgement time:
build it once into a flat table and `mmap` it, and startup is a page fault
rather than a parse.

C++ does this with a struct and a pointer. Rust does it with `rkyv` or the same
`unsafe` cast. It is available in both and it is easier in C++.

## What is on this machine, measured 2026-09-09

    g++            14.2.0 (Debian 14.2.0-19)   PRESENT
    rustc          1.98.1                       PRESENT
    cargo          1.98.1                       PRESENT

    clang++, clang-tidy, clang-format, cppcheck, include-what-you-use,
    cmake, ninja, meson, gcovr, lcov, valgrind, conan, vcpkg, bear
                                                ALL ABSENT

Every absent one is in Debian at a usable version, so this is an install rather
than a problem:

    clang-tidy 1:19.0-63     cppcheck 2.17.1-2      clang-format 1:19.0-63
    cmake 3.31.6-2           ninja-build 1.12.1-1   gcovr 7.2
    libyaml-cpp-dev 0.8.0    nlohmann-json3-dev 3.11.3

**The build toolchain is one `apt install` and the analysis suite is another.**
Nothing here needs a version manager, which is not true of the Rust pack.

## The quality jig, which is wanted anyway

C++ has no `go vet` and no `cargo clippy`: the suite is separate tools, and that
is accepted. What a C++ jig would name:

    clang-format --dry-run --Werror     format, gated
    clang-tidy                          the analyser, with a checks list
    cppcheck --enable=all               a second analyser that finds different
                                        things, which is the point of two
    include-what-you-use                header hygiene, and it has no analogue
                                        in either Go or Rust
    g++ -fanalyzer                      GCC's own static analysis, free
    gcovr                               coverage, per file, as the standard
                                        here requires
    -fsanitize=address,undefined        in the test build only

Six tools where Rust has two. It is more jig to write and more to install, and
it is also more analysis than either of the single-tool languages gets.

Toolbox has jigs for `base`, `common`, `go`, `python`, `rust` and `secrets`
today, plus a Ruby jig not yet in `jigs.yaml`. A C++ jig is a seventh.

## The two real costs

### A sixth wrench pack, or an explicit decision not to need one

`silo/docs/DECISIONS/yaml-everywhere-validated-against-the-decoded-structure.md`
says every structured file is YAML validated by JSON Schema, and wrench holds a
library per language. wrench has Go, Python, Ruby, Rust and TypeScript. There is
no C++ pack, and that decision anticipated exactly this:

> C has nothing comparable to Go's `santhosh-tekuri/jsonschema` or Python's
> `jsonschema`, so a C core means implementing the specification rather than
> binding to one.

That is accurate for C++ too. `pboettch/json-schema-validator` exists and is not
in Debian, and it is not in the same class as the Go and Python validators.

**There is a way out and it should be stated rather than assumed.** qwark reads
rule files and never writes them, so it needs to REFUSE a malformed one, not to
validate it against a schema at judgement time. Schema validation could run at
install time, in any language, from the `just install` recipe, while the runtime
does a structural load that fails closed. That is arguably the better design in
any language, because a per-call process should not be compiling a schema.

If that is accepted, C++ needs no wrench pack. If it is not, C++ costs the
hardest pack in the set.

**Ruled 2026-09-09: the pack waits.** It gets built if C++ turns out to be liked
here, and not before, because it is the most expensive thing to write on a
question nobody has answered yet. That makes the install-time-validation route
above the working assumption rather than a fallback.

### Memory safety at a security boundary

This is the argument that should decide it.

qwark's input is a command string an agent chose, and the agent is the thing
qwark exists to contain. `docs/PROJECT.md` puts the standard as: a gate that
degrades to permissive whenever it is confused is a gate whose confusion is the
way through it. A use-after-free in the option splitter is a confusion with no
verdict attached at all.

C++ has that class of defect and Rust does not. Hardening reduces it and does
not remove it: `-D_GLIBCXX_ASSERTIONS`, `-fstack-protector-strong`,
`-fsanitize=address,undefined` under test, `std::string_view` discipline over
the parse.

**One thing cuts the other way and it is worth stating fairly.** tree-sitter and
its bash grammar are C in both languages. Rust's guarantee covers the code qwark
writes around the parser, not the parser, so the difference is narrower than
"Rust is memory-safe and C++ is not" suggests. It is still real, and the code
qwark writes around the parser is where the option splitting and the path
handling live, which is the part that touches attacker-chosen text most.

## What I would say

**C++ is a more credible option than the Rust ruling's framing allows**, and on
two specific counts it is better: tree-sitter is a plain include rather than two
crates, and containment is `openat2` called directly rather than `cap-std`
depended on. The second recovers the exact thing the silo amendment recorded as
the price of leaving Go.

**It loses on memory safety, and for a containment tool that is the one to lose
on.** The reason Zig lost was that a gate's failure mode is availability; the
reason C++ loses is that a gate's other failure mode is being wrong in a way it
cannot report.

**Nothing here is urgent, because the format is language-neutral.**
`the-format-for-phases-one-and-two.md` specifies YAML, a node map and a two-pass
load, none of which cares. The parser measurement queued at
`clank/tasks/qwark/rewrite/20-measure-the-two-parsers.ready` does not care
either.

**The C++ jig is worth building regardless of what qwark is written in**, and it
does not need qwark as its first adopter. `toolbox/tests/` and a small fixture
tree would exercise it, which is how the other jigs were proved.

## Do not make qwark the first C++ program here

Asked whether to start somewhere smaller: yes, and the reason is not caution.

**qwark cannot answer the question it would be asked to answer.** The question
is whether C++ is pleasant to work in here, with this jig, this gate and these
standards. qwark is a security boundary with 133 requirements, a decision log,
an installed hook and a rule set that gates the session writing it. Every
difficulty it produces would be ambiguous between the language and the subject,
and it is the one program on this machine that cannot be half-finished, because
a half-finished gate either refuses everything or gates nothing.

**A first adopter wants the opposite properties.** Small, self-contained,
nothing depending on it being up, and a correct answer that is obvious by
inspection. That is the same test `palette-print` passed when Zig needed one,
and it is why Zig is being proved there rather than in qwark.

Three candidates, and the third is the one I would pick:

**The wrench C++ pack.** Best fit on paper: the contract is specified before any
pack is written, so it is written from a document rather than by reading the Go
one, and comparing it against five existing packs makes the language the only
variable. Deferred by the ruling above, and correctly, because it is the most
expensive of the three.

**A qwark component, not qwark.** The option splitter alone: `grep -rn`,
`grep -A12`, `go test -run X`, `dd if=x`, `tar xzvf`. It is pure input to output,
it has a corpus of 2,266 real invocations to run against, and it is the piece
whose behaviour the format work needs pinned down anyway. It answers the C++
question and the parser question in one build, and it throws away cleanly.

**The parser comparison itself**, `20-measure-the-two-parsers.ready`. It has to
be written in something, it has to link tree-sitter, and writing that side in
C++ tests the exact interop claim this document makes while producing the
measurement the rewrite is blocked on. It is the smallest of the three, it is
already agreed work, and nothing is wasted whichever way either question comes
out.

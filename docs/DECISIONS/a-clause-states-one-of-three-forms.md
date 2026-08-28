# A clause states one of three forms, and says which reading it tests

A clause states what it tests for in one of three forms, and may say which
reading of the word it tests.

    value   = "rm"          the whole word, exactly
    partial = ".claude"     anywhere within the word
    pattern = "rm|rmdir"    a regular expression over the whole word

**Naming `partial` is the point of having three forms and not two.** An earlier
draft had only exact-and-pattern, with "contains" written `.*\.claude.*`. That
works, and it hides the breadth of the rule inside a regex the reader has to
parse. A form named `partial` announces itself in the key.

`pattern` is anchored to the whole value, because otherwise every pattern is
quietly a partial and the broad reading becomes the one obtained by accident.
That is the predecessor's mistake exactly: `archive-guard.sh` matched the
substring `.archive`, blocked `web.archive.org`, and cost a legitimate research
route. Nothing here prevents an author choosing that breadth, since
`partial = ".archive"` does the same thing; choosing it is now a visible act
instead of a default.

Exactly one form per clause. Stating none is an error and not a clause matching
everything; stating several is an error and not a precedence order nobody would
remember. An empty `partial` is refused for the same reason, since
every string contains the empty string. An empty `value` stands, being precise.

The interpreted reading is the default.

    reading = "interpreted"   what the shell will pass       (default)
    reading = "written"       the source, escapes intact

Testing what was written is what lets `/home/ancient/.cl\aude/x` past a rule
about `.claude`. The written reading still earns its place, since a rule about
*how* something was spelled is a real thing to want and the alias ban is one, but
it has to be asked for.

A word with no interpreted value is not matched. Nothing is expanded, so `$HOME`
has none; a clause reading it finds nothing to test rather than testing the empty
string, which `partial` and `.*` would both match.

Go's regexp is RE2: no backtracking, linear in the input, so a rule file cannot
carry a pattern that a crafted command makes pathological. No clause about a
command word needs backreferences.

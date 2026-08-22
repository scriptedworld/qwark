# The decision model

A rule's action is one of four:

    allow   auto-approve; the command runs without a prompt
    deny    block it
    ask     force the normal permission prompt
    tag     decide nothing; attach a name to the evaluation

`tag` is what lets the other three compose. A tag rule adds to the context that
later and more expensive rules match against, so a cheap structural rule can
annotate a command and an expensive rule can be written over names instead of
walking the tree again.

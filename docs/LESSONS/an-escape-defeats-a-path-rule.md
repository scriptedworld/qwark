# An escape defeats a path rule, unless it is resolved

The parser keeps escapes in a literal's value. `a\ b` arrives as `a\ b`, while
bash passes `a b`. So `rm /home/ancient/.cl\aude/x` reaches `.claude` when the
shell runs it, and a rule comparing the unresolved text never matches. The
resolution rules differ inside double quotes from outside, and both were read off
bash rather than recalled.

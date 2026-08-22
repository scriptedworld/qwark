# The strictest action wins, so order never matters

Where several rules match one command, the verdict is the strictest of them: deny
over ask over allow.

**No rule can be weakened by where it sits**, in its file or among the files. A
rule set assembled from several sources cannot be defeated by arranging for a
permissive file to be read last, and a reader establishing what a command will do
never has to know what came before it. Under first-match-wins, every rule's
meaning depends on all the rules above it, which is the property DNF was chosen
to avoid, reappearing at the level of the file.

The cost is real and should be stated: a narrow exception cannot override a broad
deny. An exception has to be written into the deny rule as a clause that excludes
it. That is more to type, and it keeps the exception where a reader of that rule
will see it rather than in another file that quietly outranks it.

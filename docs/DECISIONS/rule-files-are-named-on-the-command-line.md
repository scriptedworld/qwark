# Rule files are named on the command line

Rule files are given as arguments: a path naming a directory contributes every
rule file in it, and a path naming a file contributes that one.

The gain is that the policy in force is readable where qwark is invoked, in the
`settings.json` entry that registers the hook, instead of being implied by which
files happen to be sitting in a directory qwark knows about.

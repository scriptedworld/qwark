# Rules are layered by cost

Four classes, cheapest first, so a decision reachable in an early class never
pays for a later one:

1. **Node presence.** Certain nodes in the tree are an instant rule.
2. **Conjunctive.** Several elements, *all* of which must match.
3. **Context.** Whether the paths involved fall inside a given directory tree.
4. **State.** An ongoing tracker across commands.

Class 1 costs nothing per rule. Every structural fact is gathered in a single
walk (`internal/shell/facts.go`), so a rule consulting them is a set lookup and
adding one is free. The cost is set by the size of the tree, not by the number of
rules.

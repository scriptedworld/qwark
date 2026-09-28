# A declaration is a permission, and so is an allow

An agent that adds a *declaration* moves a command from "undeclared, therefore
refused" to eligible. `[command.sh]` is a complete bypass written as
configuration.

An agent that adds an `allow` rule lifts every `deny` the command matched.
Only a `block` survives it (FR-4.33).

So a file that may add declarations or allow rules needs the same protection as
the global set. A lower-trust project file is possible only if it may add
refusals, `block`, `deny` and `ask`, and never allows or declarations.

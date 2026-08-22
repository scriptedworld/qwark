# A declaration is a permission

Under strictest-wins, an agent that adds an `allow` rule changes nothing, because
deny still beats it. An agent that adds a *declaration* changes a great deal: it
moves a command from "undeclared, therefore denied" to eligible. `[command.sh]`
is a complete bypass written as configuration.

So a project-level file that may add declarations needs the same protection as
the global set. A lower-trust project file is possible only if it may add
**rules** and never **declarations**.

# catwalk

**catwalk** is a unit test library for Bubbletea TUI models.

It enables implementers to verify the state of models as
they process `tea.Msg` objects through their `Update` method.

It is implemented on top of
[datadriven](https://github.com/cockroachdb/datadriven), an extension
to Go's simple "table-driven testing" idiom.

Datadriven tests use data files containing both the reference input
and output, instead of a data structure. Each data file can contain
multiple tests. When the implementation changes, the reference output
can be quickly updated by re-running the tests with the `-rewrite`
flag.


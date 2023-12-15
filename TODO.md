# General

* make test tables easier to read by removing the need to add a trailing newline in every test case
* can the assembly contain unicode which is not only ASCII?
* profile the assembler using Pong. What is something I could improve?

# Error Handling

I would like to return errors in the style of filename:line:col like for example the Go compiler.

* docs say:
// If dest is empty, the = is omitted;
// If jump is empty, the ; is omitted
but is it illegal to have them?
* the way I ignore a comment line (line starting with /) can hide an invalid comment. I could
delegate to a skipComment(in string) error function that makes sure its a well formed comment
* extract some common error helper so that I use a consistent format without having to retype it all
  the time

# To do

* Implement commands
    * `deploy`
        * Package, diff, confirm (or `--yes` flag), deploy, and store stack name in the template metadata
    * `diff`
        * Compare a template with another template or a stack
    * `rm`
        * Delete a stack with confirmation or `--yes` flag
    * `validate`
        * Run `cfn-lint` if installed
        * Run `cfn-nag` if installed
        * Run `aws cloudformation validate-template` (quietly)

* Make it work with multiple templates at once
    * Detect templates in in the current folder

* Bring in default flow
    If no command is specified, do this:

    1. `validate`
    2. `format`
    3. `diff`
    4. Manual confirm or `--yes` flag
    5. `deploy`
    6. Store stack name in template metadata

* Other ideas
    * `doc` - load documentation for a resource type
    * `minify` - try hard to get a template below the size limit

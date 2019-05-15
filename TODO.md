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
    * Add colour to diff output
    * `doc` - load documentation for a resource type
    * `minify` - try hard to get a template below the size limit
    * Move cfn-format and cfn-skeleton into this package
    * Do parameter validation (especially multiple-template stacks - checking clashing outputs etc.)
        * S3 buckets that exist or can't be created (e.g. recent deleted bucket with same name)
        * Certificates that don't exist in the correct region (e.g. non us-east-1)
        * Mismatching or existing "CNAMEs" for CloudFront distros
    * Blueprints (higher level constructs - maybe from CDK)
    * Store metadata in template
        * stack name
    * Magically add tags
        * Commit ID

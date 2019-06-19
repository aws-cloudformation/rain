# To do

* `deploy`
    * Add flag to enable no-confirm deployment.
    * Add `--yes` flag
    * Only show changing resources
    * Ensure update count reflects everything that has changed

* `rm`
    * List stack contents and ask for confirmation
    * Add `--yes` flag

* `diff`
    * Change the `<<<`, `>>>`, and `===` symbols into something else for clarity
    * Expand on hidden content with `{...}` and `[...]`

* `ls`
    * Hide regions with no stacks
    * Display in yaml(ish) format

* Add `watch` command
    * Same as the last stage of a deploy - watch a stack that's in progress

## Other ideas

* Multiple deployments. Use a rain.yaml to specify multiple stacks in multiple regions/accounts.
* `doc` - load documentation for a resource type
* `minify` - try hard to get a template below the size limit
* Do template parameter validation (especially multiple-template stacks - checking clashing outputs etc.)
    * S3 buckets that exist or can't be created (e.g. recent deleted bucket with same name)
    * Certificates that don't exist in the correct region (e.g. non us-east-1)
    * Mismatching or existing "CNAMEs" for CloudFront distros
* Blueprints (higher level constructs - maybe from CDK)
* Store metadata in template?
    * stack name
* Magically add tags?
    * Commit ID

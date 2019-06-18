# To do

* `deploy`
    * Add flag to enable no-confirm deployment.
    * Add `--yes` flag

* `rm`
    * List stack contents and ask for confirmation
    * Add `--yes` flag

* `diff`
    * Change the `<<<`, `>>>`, and `===` symbols into something else for clarity

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

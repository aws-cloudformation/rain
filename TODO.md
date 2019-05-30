# To do

* `deploy`
    * Change diff timer to:
        1. Do you want to view the diff?
        2. Are you sure you want to deploy?
    * Add flag to enable no-confirm deployment.
    * Add `--yes` flag
    * Prompt for parameter values

* `rm`
    * List stack contents and ask for confirmation
    * Add `--yes` flag

## Other ideas

* `doc` - load documentation for a resource type
* `minify` - try hard to get a template below the size limit
* Move cfn-format and cfn-skeleton into this package
* Do template parameter validation (especially multiple-template stacks - checking clashing outputs etc.)
    * S3 buckets that exist or can't be created (e.g. recent deleted bucket with same name)
    * Certificates that don't exist in the correct region (e.g. non us-east-1)
    * Mismatching or existing "CNAMEs" for CloudFront distros
* Blueprints (higher level constructs - maybe from CDK)
* Store metadata in template?
    * stack name
* Magically add tags?
    * Commit ID

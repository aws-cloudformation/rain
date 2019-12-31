# To do

* `deploy`
    * Add global Include feature - with warning
    * Ask for stack name if none supplied (default to template file name minus extension)
    * Only show changing resources
    * Ensure update count reflects everything that has changed
    * Detect whether a deployment requires capabilities rather than automatically applying them
    * Allow deploying over a stack that's REVIEW_IN_PROGRESS by killing the changeset?
    * After a failed deployment, show the logs
    * Show details from nested stacks while deploying
    * Handle deploying from a template URL

* `rm`
    * List stack contents and ask for confirmation
    * Add `--force` flag
    * Detect `y` as `Y`

* `ls`
    * Display in yaml(ish) format

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

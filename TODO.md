# To do

* Make it `rain`!
    * Should we have no-command do a series of fun things to a template?
        * format it
        * lint it
        * run nag
        * opinionated set of default actions
        * flags for tweaking it
    * Any default actions for a stack?
        * Use `stack://name` notation?
        * Maybe it should grab the template and perform the same actions, saving the template in the local folder?

* Tidy up this (old) list of features:
    * build   - construct templates from resources required and their dependencies
    * check   - validate templates against the published specification
    * clean   - perform opinionated improvements to templates
    * deploy  - deploy a template :)
    * doc     - Display the documentation for a resource or property type
    * flip    - convert templates between JSON and YAML formats
    * graph   - prints out a graph of the resources and the dependencies between them
    * ls      - List running stacks and, optionally, their resources
    * rm      - Delete a stack
    * minify  - A tool of last resort. Try hard to get your stack past that 51,200 byte limit.

#compdef _rain rain


function _rain {
  local -a commands

  _arguments -C \
    '--debug[Output debugging information]' \
    '(-h --help)'{-h,--help}'[help for rain]' \
    '(-p --profile)'{-p,--profile}'[AWS profile name; read from the AWS CLI configuration file]:' \
    '(-r --region)'{-r,--region}'[AWS region to use]:' \
    "1: :->cmnds" \
    "*::arg:->args"

  case $state in
  cmnds)
    commands=(
      "build:Create CloudFormation templates"
      "cat:Get the CloudFormation template from a running stack"
      "check:Validate a CloudFormation template against the spec"
      "deploy:Deploy a CloudFormation stack from a local template"
      "diff:Compare CloudFormation templates"
      "fmt:Format CloudFormation templates"
      "help:Help about any command"
      "info:Show your current configuration"
      "logs:Show the event log for the named stack"
      "ls:List running CloudFormation stacks"
      "rm:Delete a running CloudFormation stack"
      "tree:Find dependencies of Resources and Outputs in a local template"
      "watch:Display an updating view of a CloudFormation stack"
    )
    _describe "command" commands
    ;;
  esac

  case "$words[1]" in
  build)
    _rain_build
    ;;
  cat)
    _rain_cat
    ;;
  check)
    _rain_check
    ;;
  deploy)
    _rain_deploy
    ;;
  diff)
    _rain_diff
    ;;
  fmt)
    _rain_fmt
    ;;
  help)
    _rain_help
    ;;
  info)
    _rain_info
    ;;
  logs)
    _rain_logs
    ;;
  ls)
    _rain_ls
    ;;
  rm)
    _rain_rm
    ;;
  tree)
    _rain_tree
    ;;
  watch)
    _rain_watch
    ;;
  esac
}

function _rain_build {
  _arguments \
    '(-b --bare)'{-b,--bare}'[Produce a minimal template, omitting all optional resource properties]' \
    '(-h --help)'{-h,--help}'[help for build]' \
    '(-j --json)'{-j,--json}'[Output the templates as JSON (default format: YAML)]' \
    '(-l --list)'{-l,--list}'[List all CloudFormation resource types]' \
    '--debug[Output debugging information]' \
    '(-p --profile)'{-p,--profile}'[AWS profile name; read from the AWS CLI configuration file]:' \
    '(-r --region)'{-r,--region}'[AWS region to use]:'
}

function _rain_cat {
  _arguments \
    '(-h --help)'{-h,--help}'[help for cat]' \
    '(-t --transformed)'{-t,--transformed}'[Get the template with transformations applied by CloudFormation.]' \
    '(-u --unformatted)'{-u,--unformatted}'[Output the template in its raw form and do not attempt to format it.]' \
    '--debug[Output debugging information]' \
    '(-p --profile)'{-p,--profile}'[AWS profile name; read from the AWS CLI configuration file]:' \
    '(-r --region)'{-r,--region}'[AWS region to use]:'
}

function _rain_check {
  _arguments \
    '(-h --help)'{-h,--help}'[help for check]' \
    '--debug[Output debugging information]' \
    '(-p --profile)'{-p,--profile}'[AWS profile name; read from the AWS CLI configuration file]:' \
    '(-r --region)'{-r,--region}'[AWS region to use]:'
}

function _rain_deploy {
  _arguments \
    '(-d --detach)'{-d,--detach}'[Once deployment has started, don'\''t wait around for it to finish.]' \
    '(-f --force)'{-f,--force}'[Don'\''t ask questions; just deploy.]' \
    '(-h --help)'{-h,--help}'[help for deploy]' \
    '*--params[Set parameter values. Use the format key1=value1,key2=value2.]:' \
    '*--tags[Add tags to the stack. Use the format key1=value1,key2=value2.]:' \
    '--debug[Output debugging information]' \
    '(-p --profile)'{-p,--profile}'[AWS profile name; read from the AWS CLI configuration file]:' \
    '(-r --region)'{-r,--region}'[AWS region to use]:'
}

function _rain_diff {
  _arguments \
    '(-h --help)'{-h,--help}'[help for diff]' \
    '(-l --long)'{-l,--long}'[Include unchanged elements in diff output]' \
    '--debug[Output debugging information]' \
    '(-p --profile)'{-p,--profile}'[AWS profile name; read from the AWS CLI configuration file]:' \
    '(-r --region)'{-r,--region}'[AWS region to use]:'
}

function _rain_fmt {
  _arguments \
    '(-c --compact)'{-c,--compact}'[Produce more compact output.]' \
    '(-h --help)'{-h,--help}'[help for fmt]' \
    '(-j --json)'{-j,--json}'[Output the template as JSON (default format: YAML).]' \
    '(-v --verify)'{-v,--verify}'[Check if the input is already correctly formatted and exit.
The exit status will be 0 if so and 1 if not.]' \
    '(-w --write)'{-w,--write}'[Write the output back to the file rather than to stdout.]' \
    '--debug[Output debugging information]' \
    '(-p --profile)'{-p,--profile}'[AWS profile name; read from the AWS CLI configuration file]:' \
    '(-r --region)'{-r,--region}'[AWS region to use]:'
}

function _rain_help {
  _arguments \
    '--debug[Output debugging information]' \
    '(-p --profile)'{-p,--profile}'[AWS profile name; read from the AWS CLI configuration file]:' \
    '(-r --region)'{-r,--region}'[AWS region to use]:'
}

function _rain_info {
  _arguments \
    '(-c --creds)'{-c,--creds}'[Include current AWS credentials]' \
    '(-h --help)'{-h,--help}'[help for info]' \
    '--debug[Output debugging information]' \
    '(-p --profile)'{-p,--profile}'[AWS profile name; read from the AWS CLI configuration file]:' \
    '(-r --region)'{-r,--region}'[AWS region to use]:'
}

function _rain_logs {
  _arguments \
    '(-a --all)'{-a,--all}'[Include uninteresting logs]' \
    '(-h --help)'{-h,--help}'[help for logs]' \
    '(-l --long)'{-l,--long}'[Display full details]' \
    '(-t --time)'{-t,--time}'[Show results in order of time instead of grouped by resource]' \
    '--debug[Output debugging information]' \
    '(-p --profile)'{-p,--profile}'[AWS profile name; read from the AWS CLI configuration file]:' \
    '(-r --region)'{-r,--region}'[AWS region to use]:'
}

function _rain_ls {
  _arguments \
    '(-a --all)'{-a,--all}'[List stacks across all regions]' \
    '(-h --help)'{-h,--help}'[help for ls]' \
    '(-n --nested)'{-n,--nested}'[Show nested stacks (hidden by default)]' \
    '--debug[Output debugging information]' \
    '(-p --profile)'{-p,--profile}'[AWS profile name; read from the AWS CLI configuration file]:' \
    '(-r --region)'{-r,--region}'[AWS region to use]:'
}

function _rain_rm {
  _arguments \
    '(-d --detach)'{-d,--detach}'[Once removal has started, don'\''t wait around for it to finish.]' \
    '(-f --force)'{-f,--force}'[Do not ask; just delete]' \
    '(-h --help)'{-h,--help}'[help for rm]' \
    '--debug[Output debugging information]' \
    '(-p --profile)'{-p,--profile}'[AWS profile name; read from the AWS CLI configuration file]:' \
    '(-r --region)'{-r,--region}'[AWS region to use]:'
}

function _rain_tree {
  _arguments \
    '(-a --all)'{-a,--all}'[Display all elements, even those without any dependencies]' \
    '(-b --both)'{-b,--both}'[For each element, display both its dependencies and its dependents]' \
    '(-d --dot)'{-d,--dot}'[Output the graph in GraphViz DOT format]' \
    '(-h --help)'{-h,--help}'[help for tree]' \
    '--debug[Output debugging information]' \
    '(-p --profile)'{-p,--profile}'[AWS profile name; read from the AWS CLI configuration file]:' \
    '(-r --region)'{-r,--region}'[AWS region to use]:'
}

function _rain_watch {
  _arguments \
    '(-h --help)'{-h,--help}'[help for watch]' \
    '(-w --wait)'{-w,--wait}'[Wait for changes to begin rather than refusing to watch an unchanging stack]' \
    '--debug[Output debugging information]' \
    '(-p --profile)'{-p,--profile}'[AWS profile name; read from the AWS CLI configuration file]:' \
    '(-r --region)'{-r,--region}'[AWS region to use]:'
}


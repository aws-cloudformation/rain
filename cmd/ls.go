package cmd

import (
	"codecommit/builders/cfn-cli/util"
)

var liveStatuses = []string{
	"CREATE_IN_PROGRESS",
	"CREATE_FAILED",
	"CREATE_COMPLETE",
	"ROLLBACK_IN_PROGRESS",
	"ROLLBACK_FAILED",
	"ROLLBACK_COMPLETE",
	"DELETE_IN_PROGRESS",
	"DELETE_FAILED",
	"UPDATE_IN_PROGRESS",
	"UPDATE_COMPLETE_CLEANUP_IN_PROGRESS",
	"UPDATE_COMPLETE",
	"UPDATE_ROLLBACK_IN_PROGRESS",
	"UPDATE_ROLLBACK_FAILED",
	"UPDATE_ROLLBACK_COMPLETE_CLEANUP_IN_PROGRESS",
	"UPDATE_ROLLBACK_COMPLETE",
	"REVIEW_IN_PROGRESS",
}

func init() {
	Commands["ls"] = Command{
		Func: lsCommand,
		Help: "List running CloudFormation stacks",
	}
}

func lsCommand(args []string) {
	args = append([]string{
		"cloudformation",
		"list-stacks",
		"--output", "table",
		"--query", "StackSummaries[].[StackName,StackStatus]",
		"--stack-status-filter",
	}, liveStatuses...)

	util.RunAttached("aws", args...)
}

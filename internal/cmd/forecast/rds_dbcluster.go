package forecast

import (
	"fmt"
	"math"

	"github.com/aws-cloudformation/rain/internal/aws/iam"
	"github.com/aws-cloudformation/rain/internal/aws/rds"
	"github.com/aws-cloudformation/rain/internal/aws/servicequotas"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

// Checks configuration issues with RDS clusters
func checkRDSDBCluster(input PredictionInput) Forecast {
	forecast := makeForecast(input.typeName, input.logicalId)

	// Resource handler returned message: "Cannot find version 11.16 for aurora-postgresql (Service: Rds, Status Code: 400
	// https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-rds-dbcluster.html#cfn-rds-dbcluster-engineversion

	_, props, _ := s11n.GetMapValue(input.resource, "Properties")
	if props == nil {
		config.Debugf("expected %s to have Properties", input.logicalId)
		return forecast
	}

	spin(input.typeName, input.logicalId, "db cluster has correct engine version?")

	_, engine, _ := s11n.GetMapValue(props, "Engine")
	_, engineVersion, _ := s11n.GetMapValue(props, "EngineVersion")
	if engineVersion != nil {
		switch engine.Value {
		case "aurora-mysql":
			fallthrough
		case "aurora-postgresql":
			fallthrough
		case "mysql":
			fallthrough
		case "postgres":
			versions, err := rds.GetValidEngineVersions(engine.Value)
			if err != nil {
				config.Debugf("unable to get engine versions: %v", err)
			}
			unexpected := true
			for _, version := range versions {
				if version == engineVersion.Value {
					unexpected = false
					break
				}
			}
			if unexpected {
				LineNumber = input.resource.Line
				config.Debugf("db cluster resource: %s", node.ToJson(input.resource))
				forecast.Add(false, fmt.Sprintf("unexpected EngineVersion: %s", engineVersion.Value))
			} else {
				forecast.Add(true, "EngineVersion ok")
			}
		default:
			config.Debugf("unexpected Engine value for %s: %s",
				input.logicalId, engine.Value)
			forecast.Add(false, "unexpected Engine value")
		}
	}

	spinner.Pop()

	spin(input.typeName, input.logicalId, "db cluster has MonitoringRoleARN?")

	// Resource handler returned message: A MonitoringRoleARN value is required if you specify a MonitoringInterval value other than 0.
	_, monitoringRoleARN, _ := s11n.GetMapValue(props, "MonitoringRoleARN")
	_, monitoringInterval, _ := s11n.GetMapValue(props, "MonitoringInterval")
	if monitoringInterval != nil && monitoringInterval.Value != "0" {
		if monitoringRoleARN == nil {
			forecast.Add(false, "a MonitoringRoleARN value is required if you specify a MonitoringInterval value other than 0.")
		} else {
			// Make sure the role actually exists
			if monitoringRoleARN.Kind == yaml.ScalarNode &&
				!iam.RoleExists(monitoringRoleARN.Value) {
				forecast.Add(false,
					fmt.Sprintf("MonitoringRoleARN not found: %s",
						monitoringRoleARN.Value))
			} else {
				forecast.Add(true, "MonitoringRoleARN set")
			}
		}
	} else {
		forecast.Add(true, "MonitoringInterval not set to something other than 0")
	}

	// Check to make sure we're not at quota
	if !input.stackExists {
		quota, err := servicequotas.GetQuota("rds", "L-952B80B8")
		if err != nil {
			forecast.Add(false, fmt.Sprintf("failed: %v", err))
		} else {
			// Get the number of clusters
			numClusters, err := rds.GetNumClusters()
			if err != nil {
				forecast.Add(false, fmt.Sprintf("failed: %v", err))
			} else {
				if numClusters >= int(math.Round(quota)) {
					forecast.Add(false, "already at quota for number of clusters")
				} else {
					forecast.Add(true,
						fmt.Sprintf("quota for clusters ok: %v/%v",
							numClusters, quota))
				}
			}
		}
	}

	return forecast
}

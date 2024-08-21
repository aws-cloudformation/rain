package forecast

import (
	"fmt"
	"math"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/internal/aws/iam"
	"github.com/aws-cloudformation/rain/internal/aws/rds"
	"github.com/aws-cloudformation/rain/internal/aws/servicequotas"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	fc "github.com/aws-cloudformation/rain/plugins/forecast"
	"gopkg.in/yaml.v3"
)

// Checks configuration issues with RDS clusters
func CheckRDSDBCluster(input fc.PredictionInput) fc.Forecast {
	forecast := fc.MakeForecast(&input)

	// Resource handler returned message: "Cannot find version 11.16 for aurora-postgresql (Service: Rds, Status Code: 400
	// https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-rds-dbcluster.html#cfn-rds-dbcluster-engineversion

	_, props, _ := s11n.GetMapValue(input.Resource, "Properties")
	if props == nil {
		config.Debugf("expected %s to have Properties", input.LogicalId)
		return forecast
	}

	spin(input.TypeName, input.LogicalId, "db cluster has correct engine version?")

	var clusterEngineVersion string

	code := F0003

	_, engine, _ := s11n.GetMapValue(props, "Engine")
	_, engineVersion, _ := s11n.GetMapValue(props, "EngineVersion")
	if engineVersion != nil {
		clusterEngineVersion = engineVersion.Value
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
				config.Debugf("db cluster resource: %s", node.ToJson(input.Resource))
				forecast.Add(code, false, fmt.Sprintf("unexpected EngineVersion: %s", engineVersion.Value), getLineNum(input.LogicalId, input.Resource))
			} else {
				forecast.Add(code, true, "EngineVersion ok", getLineNum(input.LogicalId, input.Resource))
			}
		default:
			config.Debugf("unexpected Engine value for %s: %s",
				input.LogicalId, engine.Value)
			forecast.Add(code, false, "unexpected Engine value", getLineNum(input.LogicalId, input.Resource))
		}
	}

	spinner.Pop()

	code = F0004

	spin(input.TypeName, input.LogicalId, "db cluster has MonitoringRoleARN?")

	// Resource handler returned message: A MonitoringRoleARN value is required if you specify a MonitoringInterval value other than 0.
	_, monitoringRoleARN, _ := s11n.GetMapValue(props, "MonitoringRoleARN")
	_, monitoringInterval, _ := s11n.GetMapValue(props, "MonitoringInterval")
	if monitoringInterval != nil && monitoringInterval.Value != "0" {
		if monitoringRoleARN == nil {
			forecast.Add(code, false, "a MonitoringRoleARN value is required if you specify a MonitoringInterval value other than 0.", getLineNum(input.LogicalId, input.Resource))
		} else {
			// Make sure the role actually exists
			if monitoringRoleARN.Kind == yaml.ScalarNode &&
				!iam.RoleExists(monitoringRoleARN.Value) {
				forecast.Add(code, false,
					fmt.Sprintf("MonitoringRoleARN not found: %s",
						monitoringRoleARN.Value), getLineNum(input.LogicalId, input.Resource))
			} else {
				forecast.Add(code, true, "MonitoringRoleARN set", getLineNum(input.LogicalId, input.Resource))
			}
		}
	} else {
		forecast.Add(code, true, "MonitoringInterval not set to something other than 0",
			getLineNum(input.LogicalId, input.Resource))
	}

	spinner.Pop()

	code = F0005

	spin(input.TypeName, input.LogicalId, "db clusters not at quota")

	// Check to make sure we're not at quota
	if !input.StackExists {
		quota, err := servicequotas.GetQuota("rds", "L-952B80B8")
		if err != nil {
			forecast.Add(code, false, fmt.Sprintf("failed: %v", err), getLineNum(input.LogicalId, input.Resource))
		} else {
			// Get the number of clusters
			numClusters, err := rds.GetNumClusters()
			if err != nil {
				forecast.Add(code, false, fmt.Sprintf("failed: %v", err), getLineNum(input.LogicalId, input.Resource))
			} else {
				if numClusters >= int(math.Round(quota)) {
					forecast.Add(code, false, "already at quota for number of clusters",
						getLineNum(input.LogicalId, input.Resource))
				} else {
					forecast.Add(code, true,
						fmt.Sprintf("quota for clusters ok: %v/%v",
							numClusters, quota), getLineNum(input.LogicalId, input.Resource))
				}
			}
		}
	}

	spinner.Pop()

	code = F0006

	// The engine version that you requested for your DB instance (a.b) does not match the engine version of your DB cluster (c.d)
	// This kind of thing might be better in cfn-lint

	spin(input.TypeName, input.LogicalId, "db cluster engine version matches instances")

	// TODO: Move this to DBInstance checks when we implement them

	resources, err := input.Source.GetSection(cft.Resources)
	if err == nil {
		for i := 0; i < len(resources.Content); i += 2 {
			logicalId := resources.Content[i].Value
			config.Debugf("Looking for instances: %s", logicalId)
			r := resources.Content[i+1]
			_, t, _ := s11n.GetMapValue(r, "Type")
			if t != nil && t.Value == "AWS::RDS::DBInstance" {
				config.Debugf("Found instance")
				_, instanceProps, _ := s11n.GetMapValue(r, "Properties")
				if instanceProps != nil {
					for j := 0; j < len(instanceProps.Content); j += 2 {
						propName := instanceProps.Content[j].Value
						if propName == "EngineVersion" {
							evNode := instanceProps.Content[j+1]
							config.Debugf("instanceVersion: %s", node.ToSJson(evNode))

							// Resolve refs first
							resolveParamRefs(propName, evNode, input.Dc, instanceProps)

							config.Debugf("instanceVersion after: %s", node.ToSJson(evNode))

							instanceVersion := evNode.Value
							if evNode.Kind == yaml.ScalarNode && instanceVersion != clusterEngineVersion {
								forecast.Add(code, false, fmt.Sprintf(
									"engine mismatch with %s: %s != %s",
									logicalId, instanceVersion, clusterEngineVersion),
									getLineNum(input.LogicalId, input.Resource))
							} else {
								forecast.Add(code, true, "instance engine version matches",
									getLineNum(input.LogicalId, input.Resource))
							}
						}
					}
				}
			}
		}
	}

	spinner.Pop()

	return forecast
}

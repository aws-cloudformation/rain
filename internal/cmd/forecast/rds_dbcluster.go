package forecast

import (
	"fmt"

	"github.com/aws-cloudformation/rain/internal/aws/rds"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/s11n"
)

// Checks configuration issues with RDS clusters
func checkRDSDBCluster(input PredictionInput) Forecast {
	forecast := makeForecast(input.typeName, input.logicalId)

	// TODO

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

	return forecast
}

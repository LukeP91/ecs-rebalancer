package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

type ecsCloudWatchEvent struct {
	ContainerInstanceArn string `json:"containerInstanceArn"`
	AgentConnected       bool   `json:"agentConnected"`
}

func parseEcsCloudWatchEvent(event events.CloudWatchEvent) (ecsCloudWatchEvent, error) {
	var ecsEvent ecsCloudWatchEvent
	err := json.Unmarshal([]byte(event.Detail), &ecsEvent)
	if err != nil {
		return ecsCloudWatchEvent{}, err
	}

	return ecsEvent, nil
}

func HandleRequest(ctx context.Context, event events.CloudWatchEvent) error {
	log.Printf("Cloudwatch event: %v", event.Detail)
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}
	ecsClient := ecs.NewFromConfig(cfg)
	ecsEvent, err := parseEcsCloudWatchEvent(event)
	if err != nil {
		log.Fatal(err)
	}

	response, err := ecsClient.DescribeContainerInstances(
		ctx,
		&ecs.DescribeContainerInstancesInput{
			ContainerInstances: []string{ecsEvent.ContainerInstanceArn},
			Cluster:            aws.String(clusterName),
		},
	)
	numberOfInstances := len(response.ContainerInstances)
	log.Println("Number of instances:", numberOfInstances)
	if numberOfInstances != 0 {
		containerInstance := response.ContainerInstances[0]
		numberOfRunningTasks := containerInstance.RunningTasksCount
		numberOfPendingTasks := containerInstance.PendingTasksCount

		if numberOfRunningTasks == int32(0) && numberOfPendingTasks == int32(0) && ecsEvent.AgentConnected {
			services, err := getServices(ecsClient)
			if err != nil {
				log.Fatal(err)
			}
			err = updateServices(ecsClient, services)
			if err != nil {
				log.Fatal(err)
			}
			return nil
		} else {
			log.Println("Cluster does not require rebalacing.")
			return nil
		}
	}
	return nil
}

var clusterName = os.Getenv("ECS_CLUSTER_NAME")

func main() {
	lambda.Start(HandleRequest)
}

func getServices(client *ecs.Client) ([]string, error) {
	services := make([]string, 0)
	for {
		response, err := client.ListServices(
			context.TODO(),
			&ecs.ListServicesInput{Cluster: aws.String(clusterName)},
		)
		if err != nil {
			return nil, err
		}
		services = append(services, response.ServiceArns...)
		if response.NextToken == nil {
			break
		}
	}

	return services, nil
}

func updateServices(client *ecs.Client, services []string) error {
	response, err := client.DescribeServices(
		context.TODO(),
		&ecs.DescribeServicesInput{
			Cluster:  aws.String(clusterName),
			Services: services,
		},
	)
	if err != nil {
		return err
	}

	describedServices := response.Services
	for _, service := range describedServices {
		log.Println("Service to be updated", *service.ServiceName)
		_, err = client.UpdateService(
			context.TODO(),
			&ecs.UpdateServiceInput{
				Cluster:            aws.String(clusterName),
				ForceNewDeployment: true,
				Service:            aws.String(*service.ServiceArn),
			},
		)
		if err != nil {
			return err
		}
		log.Println(
			"Updated service:",
			*service.ServiceName,
			"with new task definition.",
		)
	}
	return nil
}

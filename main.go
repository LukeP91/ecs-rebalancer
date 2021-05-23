package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

// func HandleRequest(ctx context.Context, event events.CloudWatchEvent) (error) {
// 	log.Printf("Cloudwatch event: %v", event.Detail)

// }

func main() {
	// lambda.Start(HandleRequest)
	// Load the Shared AWS Configuration (~/.aws/config)
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile("documobi"), config.WithRegion("eu-west-1"))
	if err != nil {
		log.Fatal(err)
	}
	ecs_client := ecs.NewFromConfig(cfg)
	services, err := get_services(ecs_client)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Service arns: ", services)
}

func get_services(ecs_client *ecs.Client) ([]string, error) {
	services := make([]string, 1)
	for {
		response, err := ecs_client.ListServices(context.TODO(), &ecs.ListServicesInput{Cluster: aws.String("staging-portal-cluster")})
		if err != nil {
			log.Fatal(err)
			return make([]string, 0), err
		}
		services = append(services, response.ServiceArns...)
		if response.NextToken == nil {
			break
		}
	}

	return services, nil
}

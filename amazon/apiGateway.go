package amazon

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigateway"
)

//ApigatewayOutput is a struct for handinling AWS output
type ApigatewayOutput struct {
	BasePath string `json:"basePath"`
	Host     string `json:"host"`
	Info     struct {
		Title   string `json:"title"`
		Version string `json:"version"`
	} `json:"info"`
	Paths struct {
		__proxy__ struct {
			X_amazon_apigateway_any_method struct {
				Parameters []struct {
					In       string `json:"in"`
					Name     string `json:"name"`
					Required bool   `json:"required"`
					Type     string `json:"type"`
				} `json:"parameters"`
				Responses struct{} `json:"responses"`
			} `json:"x-amazon-apigateway-any-method"`
		} `json:"/{proxy+}"`
	} `json:"paths"`
	Schemes []string `json:"schemes"`
	Swagger string   `json:"swagger"`
}

func createGateway(region string, accessID string, secret string) *apigateway.APIGateway {
	//Create new Api Gateway Session
	svc := apigateway.New(session.New(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessID, secret, ""),
	}))
	return svc
}

func createAPI(svc *apigateway.APIGateway, name string, description string) *apigateway.RestApi {
	restAPI, err := svc.CreateRestApi(&apigateway.CreateRestApiInput{
		Name:        aws.String(name),
		Description: aws.String(description),
	})
	if err != nil {
		log.Println("Problem when creating rest API", err)
		os.Exit(1)
	}
	return restAPI
}

func deployResource(svc *apigateway.APIGateway, restAPI *apigateway.RestApi, url string, projectName string) ApigatewayOutput {
	var apiOutput ApigatewayOutput
	//Get the resources of the new api gateway instance
	rootResource, err := svc.GetResources(&apigateway.GetResourcesInput{
		RestApiId: restAPI.Id,
	})
	if err != nil {
		log.Println("Problem when getting root resource", err)
		return apiOutput
	}
	//Create the new child resource
	childResource, err := svc.CreateResource(&apigateway.CreateResourceInput{
		ParentId:  aws.String(*rootResource.Items[0].Id),
		PathPart:  aws.String("{proxy+}"),
		RestApiId: aws.String(*restAPI.Id),
	})
	if err != nil {
		log.Println("Problem when creating child resource", err)
		return apiOutput
	}
	childResource.SetPath("/{proxy+}")
	//Add method to child resource
	_, err = svc.PutMethod(&apigateway.PutMethodInput{
		ApiKeyRequired:    aws.Bool(false),
		HttpMethod:        aws.String("ANY"),
		RestApiId:         restAPI.Id,
		AuthorizationType: aws.String("NONE"),
		ResourceId:        childResource.Id,
		RequestParameters: map[string]*bool{"method.request.path.proxy": aws.Bool(true)},
	})
	if err != nil {
		log.Println("Problem when putting Method", err)
		return apiOutput
	}
	//Add Integration to child resource and method
	_, err = svc.PutIntegration(&apigateway.PutIntegrationInput{
		CacheKeyParameters:    aws.StringSlice([]string{"method.request.path.proxy"}),
		CacheNamespace:        childResource.Id,
		HttpMethod:            aws.String("ANY"),
		IntegrationHttpMethod: aws.String("ANY"),
		PassthroughBehavior:   aws.String("WHEN_NO_MATCH"),
		RequestParameters:     aws.StringMap(map[string]string{"integration.request.path.proxy": "method.request.path.proxy"}),
		Type:                  aws.String("HTTP_PROXY"),
		Uri:                   aws.String(url + "/{proxy}"),
		RestApiId:             restAPI.Id,
		ResourceId:            childResource.Id,
	})
	if err != nil {
		log.Println("Problem when putting Integration", err)
		return apiOutput
	}
	//Deployment
	_, err = svc.CreateDeployment(&apigateway.CreateDeploymentInput{
		RestApiId: restAPI.Id,
		StageName: aws.String(projectName),
	})
	if err != nil {
		log.Println("Problem when deploying API", err)
		return apiOutput
	}
	//Exporting
	export, err := svc.GetExport(&apigateway.GetExportInput{
		ExportType: aws.String("swagger"),
		StageName:  aws.String(projectName),
		RestApiId:  restAPI.Id,
	})
	json.Unmarshal(export.Body, &apiOutput)
	return apiOutput
}

func deleteRestAPI(restAPI *apigateway.RestApi, svc *apigateway.APIGateway) error {
	_, err := svc.DeleteRestApi(&apigateway.DeleteRestApiInput{
		RestApiId: restAPI.Id,
	})
	return err
}

func parseURI(url string) string {
	lastPosition := len(url) - 1
	tempURL := url
	if url[lastPosition:] == "/" {
		tempURL = url[0 : len(url)-1]
	}
	return tempURL
}

func commandOutput(apiOutput ApigatewayOutput, url string) {
	fmt.Println("")
	fmt.Println("Congrats, you now have a new api that points to: " + url)
	fmt.Println("Make all calls with this URL: https://" + apiOutput.Host + "" + apiOutput.BasePath + "/")
	fmt.Println("")
}

func verifyVariables(awsID string, awsSecret string, apiName string, url string) bool {
	if awsID == "" {
		flag.Usage()
		return false
	}
	if awsSecret == "" {
		flag.Usage()
		return false
	}
	if apiName == "" {
		flag.Usage()
		return false
	}
	if url == "" {
		flag.Usage()
		return false
	}
	return true
}

// func gateway() {
// 	awsID := flag.String("id", "", "AWS Access Key ID (Required)")
// 	awsSecret := flag.String("secret", "", "AWS API Secret (Required)")
// 	region := flag.String("region", "us-east-1", "AWS region to deploy - (Default: us-east-1)")
// 	desc := flag.String("description", "API Gateway", "Description of your API - (Default: API Gateway)")
// 	apiName := flag.String("api", "", "Name to give your new API (Required)")
// 	url := flag.String("url", "", "Target URL (Required)")
// 	stageName := flag.String("stage", "test", "Stage name to use (Default: test)")
// 	flag.Parse()
// 	if verifyVariables(*awsID, *awsSecret, *apiName, *url) {
// 		//Create new APIGateway
// 		svc := createGateway(*region, *awsID, *awsSecret)

// 		//Create New Rest API
// 		restAPI := createAPI(svc, *apiName, *desc)

// 		//Deploy and Configure Resource
// 		apiOutput := deployResource(svc, restAPI, parseURI(*url), *stageName)
// 		commandOutput(apiOutput, *url)

// 		if err := deleteRestAPI(restAPI, svc); err != nil {
// 			log.Printf("Could not delete API: %s\n", *restAPI.Id)
// 		} else {
// 			log.Printf("Deleted API: %s\n", *restAPI.Id)
// 		}

// 	}
// }

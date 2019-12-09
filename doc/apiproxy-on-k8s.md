# Run ESPv2 on Google GKE

This tutorial describes how to run ESPv2 as a sidecar for the Bookstore endpoint,
on a Google GKE cluster.

## Prerequisites

* [Set up a Kubernetes Cluster](http://kubernetes.io/docs/getting-started-guides/)
* [Installing `kubectl`](http://kubernetes.io/docs/user-guide/prereqs/)

## Before you begin

1. Select or create a [Cloud Platform Console project](https://console.cloud.google.com/project).

2. [Enable billing](https://support.google.com/cloud/answer/6293499#enable-billing) for your project.

3. Note the project ID, because you'll need it later.

4. Install [CURL](https://curl.haxx.se/download.html) for testing purposes.

5. Enable [Cloud Endpoints API](https://console.cloud.google.com/apis/api/endpoints.googleapis.com/overview),
   [Cloud Service Management API](https://console.cloud.google.com/apis/api/servicemanagement.googleapis.com/overview),
   and [Cloud Service Control API](https://console.cloud.google.com/apis/api/servicecontrol.googleapis.com/overview)
   for your project in the Google Cloud Endpoints page in the API Manager.
   Ignore any prompt to create credentials.

6. [Download the Google Cloud SDK](https://cloud.google.com/sdk/docs/quickstarts).

## Configuring Endpoints

The bookstore_grpc sample contains the files that you need to copy locally and configure.

To configure Endpoints:

Go to directory: gcpproxy/tests/endpoints/bookstore_grpc

Create a self-contained protobuf descriptor file from your service .proto file, or use the existing [one](../tests/endpoints/bookstore_grpc/proto/api_descriptor.pb)

Open the [service configuration file](../tests/endpoints/bookstore_grpc/proto/api_config_auth.yaml). This file defines the gRPC API configuration for the Bookstore service.

Note the following:
  Replace <YOUR_PROJECT_ID> in your api_config.yaml file with your GCP project ID.
  For example:

    #
    # Name of the service configuration.
    #
    name: bookstore.endpoints.example-project-12345.cloud.goog

  Note that the apis.name field value in this file exactly matches the fully-qualified API name from the .proto file; otherwise deployment won't work. The Bookstore service is defined in bookstore.proto inside package endpoints.examples.bookstore. Its fully-qualified API name is endpoints.examples.bookstore.Bookstore, just as it appears in the api_config.yaml file.

    apis:
    - name: endpoints.examples.bookstore.Bookstore

Note that bookstore.endpoints.YOUR_PROJECT_ID.cloud.goog is the Endpoints service name. It isn't the fully qualified domain name (FQDN) that you use for sending requests to the API.

## Deploying the Endpoints configuration

To deploy the Endpoints configuration, you use the [gcloud endpoints services deploy](https://cloud.google.com/sdk/gcloud/reference/endpoints/services/deploy) command. This command uses [Service Management](https://cloud.google.com/service-infrastructure/docs/manage-config), Google's foundational services platform, used by Endpoints and other services to create and manage APIs and services.

To deploy the Endpoints configuration:

Make sure you are in the directory where the api_descriptor.pb and api_config_auth.yaml files are located.

Confirm that the default project that the gcloud command-line tool is currently using is the GCP project that you want to deploy the Endpoints configuration to. Validate the project ID returned from the following command to make sure that the service doesn't get created in the wrong project.

```
gcloud config list project
```

If you need to change the default project, run the following command:

```
gcloud config set project YOUR_PROJECT_ID
```

Deploy the proto descriptor file and the configuration file by using the gcloud command-line tool:

```
gcloud endpoints services deploy api_descriptor.pb api_config_auth.yaml
```

As it is creating and configuring the service, Service Management outputs information to the terminal. When it finishes configuring the service, Service Management outputs the service configuration ID and the service name, similar to the following:

```
Service Configuration [2019-05-13r0] uploaded for service [bookstore.endpoints.example-project.cloud.goog]
```

In the previous example, [2019-05-13r0] is the service configuration ID and bookstore.endpoints.example-project.cloud.goog is the service name. The service configuration ID consists of a date stamp followed by a revision number. If you deploy the Endpoints configuration again on the same day, the revision number is incremented in the service configuration ID.

If you get an error message, see Troubleshooting Endpoints configuration deployment, see Deploying the Endpoints configuration for additional information.

## Deploying the API backend on GKE

So far you have deployed the service configuration to Service Management, but you have not yet deployed the code that serves the API backend. This section walks you through deploying prebuilt containers for the sample API and APIProxy to Kubernetes.

* Create a GKE cluster and connect to it

Check and modify the Kubernetes configuration: bookstore-k8s.yaml
Check the apiproxy image name and args.

* Update the args, change the service name to YOUR_PROJECT_ID.

The --rollout_strategy=managed option configures APIProxy to use the latest deployed service configuration. When you specify this option, within a minute after you deploy a new service configuration, APIProxy detects the change and automatically begins using it. We recommend that you specify this option instead of a specific configuration ID for APIProxy to use.

* Deploy service on kubernetes

```
kubectl create -f tests/e2e/testdata/bookstore_grpc/bookstore-k8s.yaml
```

## Testing the API

You need the service's external IP address to send requests to the sample API. It can take a few minutes after you start your service in the container before the external IP address is ready.

```
HOST=$(kubectl get service app | awk '{print $4}' | grep -v EXTERNAL-IP)
```

### Tests with gRPC client

We supply a test client library in Golang, which can be used to send requests to your backend API.
We can test with different scenario by running the following scripts:

**1. Reject if no jwt token**

```
go run tests/endpoints/bookstore_grpc/client_main.go --addr=$HOST:80 --method=ListShelves --client_protocol=grpc
```

**2. Reject if no API KEY**

Download GoogleServiceAccount from your cloud project to generate JWT token, gen-auth-token is an example script to generate jwt token, please update accordingly

```
JWT_TOKEN=`./tests/e2e/scripts/gen-auth-token.sh -a cloudesf-test-client -s YourSecretFile`

go run tests/endpoints/bookstore_grpc/client_main.go --addr=$HOST:80 --method=ListShelves --client_protocol=grpc --token=$JWT_TOKEN
```

**3. Succeed**

```
API_KEY=YOUR API KEY

go run tests/endpoints/bookstore_grpc/client_main.go --addr=$HOST:80 --method=ListShelves --client_protocol=grpc --token=$JWT_TOKEN --apikey=$API_KEY
```

**4. Invalid audience**

```
JWT_TOKEN_INVALID_AUD=`./tests/e2e/scripts/gen-auth-token.sh -a unauthorized-client  -s YourSecretFile`

go run tests/endpoints/bookstore_grpc/client_main.go --addr=$HOST:80 --method=ListShelves --client_protocol=grpc --token=$JWT_TOKEN_INVALID_AUD --apikey=$API_KEY
```

### Tests with HTTP client

**1. With transcoding**

```
curl --header "x-api-key: $API_KEY " http://$HOST/v1/shelves?access_token=$JWT_TOKEN&echo
```

**2. With url binding parameter**

```
curl --header "x-api-key: $API_KEY " http://$HOST/v1/shelves/1&echo
curl "http://$HOST/v1/shelves/1/books/1?key=$API_KEY"&echo
```

**3. With query parameter with request body**

```
curl -X POST --header "x-api-key: $API_KEY " http://$HOST/v1/shelves?access_token=$JWT_TOKEN&id=3

curl --header "x-api-key: $API_KEY " http://$HOST/v1/shelves?access_token=$JWT_TOKEN&echo
```

## Monitoring the API

APIProxy is integrated with multi Google Services for API management and monitoring, including Service Management, Stackdriver Logging, and Stackdriver trace.  So, you can monitor your API on Pantheon UI by: Service Stats, including QPS, ErrorRate, Latency,  Request/Response Size etc.

## Cleaning up

To avoid incurring charges to your Google Cloud Platform account for the resources used in this tutorial:
Delete the API:

```
gcloud endpoints services delete SERVICE_NAME
```

Replace SERVICE_NAME with the name of your API.


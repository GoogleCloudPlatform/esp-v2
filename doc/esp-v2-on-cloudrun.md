# Run ESPv2 on Google Cloud Run

This tutorial describes how to run ESPv2 as a gateway for the Bookstore
endpoint, on Google Cloud Run.

## Before you begin

1.  Select or create a
    [Cloud Platform Console project](https://console.cloud.google.com/project).

2.  [Enable billing](https://support.google.com/cloud/answer/6293499#enable-billing)
    for your project.

3.  Note the project ID, because you'll need it later.

4.  Install [CURL](https://curl.haxx.se/download.html) for testing purposes.

5.  Enable
    [Cloud Endpoints API](https://console.cloud.google.com/apis/api/endpoints.googleapis.com/overview),
    [Cloud Service Management API](https://console.cloud.google.com/apis/api/servicemanagement.googleapis.com/overview),
    and
    [Cloud Service Control API](https://console.cloud.google.com/apis/api/servicecontrol.googleapis.com/overview)
    for your project in the Google Cloud Endpoints page in the API Manager.
    Ignore any prompt to create credentials.

6.  [Download the Google Cloud SDK](https://cloud.google.com/sdk/docs/quickstarts).

## Deploying Backend service on Cloud Run

For this tutorial, we will deploy a simple HTTP bookstore manager as the backend application.
We supply the Docker image for the backend at
[gcr.io/endpoints-release/bookstore:1](https://gcr.io/endpoints-release/bookstore:1),
which is built from this
[Dockerfile](../tests/endpoints/bookstore/bookstore.Dockerfile).

To deploy Bookstore service on Cloud Run, you can either do it on Pantheon UI,
(by choosing Cloud Run, then create Service), or you can directly run the
following command, with the name that you want to use for the service, as well
as the project ID created above.

```
gcloud beta run deploy CLOUD_RUN_SERVICE_NAME \
    --image="gcr.io/endpoints-release/bookstore:1" \
    --allow-unauthenticated \
    --platform managed \
    --project=YOUR_PROJECT_ID
```

On successful completion, the command displays a message similar to the
following:

```
Service [bookstore] revision [bookstore-00001] has been deployed and is serving
traffic at https://BACKEND_SERVICE_URL
```

You can verify its status by sending a request to the service by:

```
curl https://BACKEND_SERVICE_URL/shelves
```

## Deploying ESPv2

Similarly, you need to deploy ESPv2 on Google Cloud Run using a docker image.
We supply the Docker image for ESPv2 at
[gcr.io/endpoints-release/endpoints-runtime-serverless:2](https://gcr.io/endpoints-release/endpoints-runtime-serverless:2).
Note the `-serverless` suffix in this image, which denotes this is specifically
for use on Cloud Run.

```
gcloud beta run deploy ESPv2_SERVICE_NAME \
    --image="gcr.io/endpoints-release/endpoints-runtime-serverless:2" \
    --allow-unauthenticated \
    --platform managed \
    --project=YOUR_PROJECT_ID
```

Replace `ESPv2_SERVICE_NAME` and `YOUR_PROJECT_ID` accordingly.

On successful completion, similar message is displayed:

```
Service [apiproxy] revision [apiproxy-00001] has been deployed and is serving
traffic at https://PROXY_SERVICE_URL
```

## Configuring Endpoints

You must have an OpenAPI document based on OpenAPI Specification v2.0 that
describes the surface of your backend service and any authentication
requirements. You also need to add a Google-specific field that contains the URL
for each service so that ESPv2 has the information it needs to invoke a
service.

We supply a
[template](../tests/endpoints/bookstore/bookstore_swagger_template.json) for
the bookstore service. You must make the following changes to it:

1) Change the `host` name to the `PROXY_SERVICE_URL`, **without** the protocol identifier.
2) Add the `x-google-backend` object with the address to `BACKEND_SERVICE_URL`,
**with** the protocol identifier.

For example:

```
...

"host": "PROXY_SERVICE_URL",
"x-google-backend": {
  "address": "https://BACKEND_SERVICE_URL"
},
...

```

## Deploying the Endpoints configuration

To deploy the Endpoints configuration, use the
[gcloud endpoints services deploy](https://cloud.google.com/sdk/gcloud/reference/endpoints/services/deploy) command. This command pushes the configuration to
[Google Service Management](https://cloud.google.com/service-infrastructure/docs/manage-config), Google's foundational services platform used by Endpoints and other services for creating and managing APIs and services.

To deploy the Endpoints configuration:

```
gcloud endpoints services deploy bookstore_swagger_template.json
```

As it is creating and configuring the service, Service Management outputs
information to the terminal. When it finishes configuring the service, Service
Management outputs the service configuration ID and the service name, similar to
the following:

```
Service Configuration [ENDPOINTS_SERVICE_CONFIG_ID] uploaded for service [ENDPOINTS_SERVICE_NAME]
```

Note that on Cloud Run, `ENDPOINTS_SERVICE_NAME` is usually the same as `PROXY_SERVICE_URL`
(minus the protocol identifier).


## Build the service config to the ESPv2 docker image

You need to build the service config into a new ESPv2 image and redeploy that new image to Cloud Run.
We provide a bash script to automate this process. Ensure you have the gcloud SDK installed and download
this [script](../docker/serverless/gcloud_build_image).

Run it with the following commands:

```
chmod +x gcloud_build_image &&
./gcloud_build_image -s ENDPOINTS_SERVICE_NAME -c ENDPOINTS_SERVICE_CONFIG_ID -p YOUR_PROJECT_ID
```

It will use gcloud command to download the service config, build the
service config into a new docker image, and upload the new image to your project
container registry located here:

```
gcr.io/YOUR_PROJECT_ID/apiproxy-serverless:ENDPOINTS_SERVICE_NAME-ENDPOINTS_SERVICE_CONFIG_ID
```

## Redeploy the ESPv2 Cloud Run service with the new image

Replace ESPv2_SERVICE_NAME with the name of your Cloud Run service.

```
gcloud beta run deploy ESPv2_SERVICE_NAME \
  --image="gcr.io/YOUR_PROJECT_ID/apiproxy-serverless:ENDPOINTS_SERVICE_NAME-ENDPOINTS_SERVICE_CONFIG_ID" \
  --allow-unauthenticated \
  --platform managed \
  --project=YOUR_PROJECT_ID
```

## Testing the API

Now you can sent request to the backend service through the ESPv2 proxy:

```
curl https://PROXY_SERVICE_URL/shelves
```
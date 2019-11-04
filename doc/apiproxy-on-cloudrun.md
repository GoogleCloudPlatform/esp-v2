# Run API Proxy on Google Cloud Run

This tutorial describes how to run API Proxy as a gateway for the Bookstore
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
    [Cloud Service Management API](https://pantheon.corp.google.com/apis/api/servicemanagement.googleapis.com/overview),
    and
    [Cloud Service Control API](https://pantheon.corp.google.com/apis/api/servicecontrol.googleapis.com/overview)
    for your project in the Google Cloud Endpoints page in the API Manager.
    Ignore any prompt to create credentials.

6.  [Download the Google Cloud SDK](https://cloud.google.com/sdk/docs/quickstarts).

## Deploying Backend service on Cloud Run

First, you need a docker image. We supply an image at
gcr.io/apiproxy-release/bookstore:1, which is built from this
[Dockerfile](/tests/e2e/testdata/bookstore/bookstore.Dockerfile).

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

Service [bookstore] revision [bookstore-00001] has been deployed and is serving
traffic at https://bookstore-12345-uc.a.run.app

You can verify its status by sending a request to the service by:

```
curl https://bookstore-12345-uc.a.run.app/shelves
```

## Deploying API Proxy

Similarly, you need to deploy API Proxy on Google Cloud Run, by:

```
gcloud beta run deploy API_PROXY_SERVICE_NAME \
    --image="gcr.io/apiproxy-release/apiproxy-serverless:0" \
    --allow-unauthenticated \
    --platform managed \
    --project=YOUR_PROJECT_ID
```

Replace API_PROXY_SERVICE_NAME and YOUR_PROJECT_ID accordingly.

On successful completion, similar message is displayed:

Service [apiproxy] revision [apiproxy-00001] has been deployed and is serving
traffic at https://apiproxy-45678-uc.a.run.app

## Configuring Endpoints

You must have an OpenAPI document based on OpenAPI Specification v2.0 that
describes the surface of your backend service and any authentication
requirements. You also need to add a Google-specific field that contains the URL
for each service so that API Proxy has the information it needs to invoke a
service.

We supply a
[template](/tests/e2e/testdata/bookstore/bookstore_swagger_template.json) for
the bookstore service. You need to change the host name to the
API_PROXY_SERVICE_NAME URL, without prefix "https://". Also, you need to define
the address of your backend service with `x-google-backend`.

For example:

```
...

"host": "apiproxy-45678-uc.a.run.app",
"x-google-backend": {
  "address": "https://bookstore-12345-uc.a.run.app" }
...

```

## Deploying the Endpoints configuration

To deploy the Endpoints configuration, you use the
[gcloud endpoints services deploy](https://cloud.google.com/sdk/gcloud/reference/endpoints/services/deploy)
command. This command uses
[Service Management](https://cloud.google.com/service-infrastructure/docs/manage-config),
Google's foundational services platform, used by Endpoints and other services to
create and manage APIs and services.

To deploy the Endpoints configuration:

```
gcloud endpoints services deploy bookstore_swagger_template.json
```

As it is creating and configuring the service, Service Management outputs
information to the terminal. When it finishes configuring the service, Service
Management outputs the service configuration ID and the service name, similar to
the following:

```
Service Configuration [2019-05-13r0] uploaded for service [apiproxy-45678-uc.a.run.app]
```

## Re-Deploying API Proxy to update the service config

After the service configuration is deployed to service management API, you need to re-deploy the API Proxy so that it can pick up the service configuration that just deployed.


```
gcloud beta run deploy API_PROXY_SERVICE_NAME \
    --image="gcr.io/apiproxy-release/apiproxy-serverless:0" \
    --set-env-vars=ENDPOINTS_SERVICE_NAME=apiproxy-45678-uc.a.run.app
    --allow-unauthenticated \
    --platform managed \
    --project=YOUR_PROJECT_ID
```

## Testing the API

Now you can sent request to the backend servie, through API Proxy:

```
curl https://apiproxy-45678-uc.a.run.app/shelves
```
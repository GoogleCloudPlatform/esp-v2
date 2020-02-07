#!/bin/bash

# Janitor script triggered by prow to cleanup ESPv2 on serverless resources.
# This includes:
# - Cloud run services
# - Cloud functions
# - Endpoints services

# Fail on any error.
set -eo pipefail

# Set the project id
gcloud config set project cloudesf-testing
gcloud config set run/region us-central1

# Resources older than 1 day should be cleaned up
LIMIT_DATE=$(date -d "1 day ago" +%F)
echo "Cleaning up resources before ${LIMIT_DATE}"

### GKE Cluster ###
GKE_SERVICES=$(gcloud container clusters list --format="value(NAME)" \
  --filter="name ~ e2e-cloud-run AND creationTime  < ${LIMIT_DATE}" )

for service in ${GKE_SERVICES};
do
  echo "Deleting GKEservice: ${service}"
  gcloud container clusters delete ${service} \
    --zone=us-central1-a \
    --quiet
done

### Cloud Run ###
CLOUD_RUN_SERVICES=$(gcloud run services list \
    --platform=managed \
    --filter="metadata.name ~ ^cloudesf-testing-e2e-test- \
    AND metadata.creationTimestamp < ${LIMIT_DATE}" \
  --format="value(metadata.name)")

# Note: This variable should NOT be in quotation marks,
# allowing iteration over multi-line string
for service in $CLOUD_RUN_SERVICES ; do
  echo "Deleting Cloud Run service: ${service}"
  gcloud run services delete ${service} \
    --platform=managed \
    --quiet
done
echo "Done cleaning up Cloud Run services"

### Cloud Functions ###
GOOGLE_FUNCTIONS=$(gcloud functions list \
    --filter="name ~ cloudesf-testing-e2e-test- \
    AND updateTime < ${LIMIT_DATE}" \
  --format="value(name)")

# Note: This variable should NOT be in quotation marks,
# allowing iteration over multi-line string
for gf in $GOOGLE_FUNCTIONS ; do
  echo "Deleting Google Function: ${gf}"
  gcloud functions delete ${gf} \
    --quiet
done
echo "Done cleaning up Cloud Functions"

### Endpoints Services ###
ENDPOINTS_SERVICES=$(gcloud endpoints services list \
    --filter="serviceName ~ cloudesf-testing-e2e-test-" \
  --format="value(serviceName)")

# Note: This variable should NOT be in quotation marks,
# allowing iteration over multi-line string
for service in $ENDPOINTS_SERVICES ; do
  echo "Checking if Endpoints Service is old enough to be deleted: ${service}"

  # The endpoints API does not expose creation date, so infer it from the config id.
  CONFIG_ID=$(gcloud endpoints configs list \
      --service="${service}" \
      --limit=1 \
    --format="value(id)")

  if [ -z "${CONFIG_ID}" ]
  then
    echo "Cannot determine config id, skipping cleanup for this service"
    continue
  fi

  CONFIG_DATE="${CONFIG_ID::-2}"
  echo "Found date: ${CONFIG_DATE}"
  if [[ "${CONFIG_DATE}" < "${LIMIT_DATE}" ]] ;
  then
    echo "Cleaning up service"
    gcloud endpoints services delete ${service} \
      --quiet \
      --async
  fi

done
echo "Done cleaning up Endpoints Services"

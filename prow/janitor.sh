#!/bin/bash

# Janitor script triggered by prow to cleanup ESPv2 on serverless resources.
# This includes:
# - Cloud run services
# - Cloud functions
# - Endpoints services

# Fail on any error.
set -eo pipefail

# Set the project id
PROJECT="cloudesf-testing"
REGION="us-central1"
gcloud config set project ${PROJECT}
gcloud config set run/region "${REGION}"

# Resources older than 1 day should be cleaned up
LIMIT_DATE=$(date -d "1 day ago" +%F)
echo "Cleaning up resources before ${LIMIT_DATE}"

### GKE Cluster ###
GKE_SERVICES=$(gcloud container clusters list --format='value[separator=","](NAME,ZONE)' \
  --filter="name ~ ^e2e- AND createTime  < ${LIMIT_DATE}" )

for service in ${GKE_SERVICES};
do
  name=$(echo "${service}" | cut -d "," -f 1)
  zone=$(echo "${service}" | cut -d "," -f 2)
  echo "Deleting GKE service: name=${name}, zone=${zone}"
  gcloud container clusters delete "${name}" \
    --zone="${zone}" \
    --async \
    --quiet
done
echo "Done cleaning up GKE services"

### Cloud Run ###
CLOUD_RUN_SERVICES=$(gcloud run services list \
    --platform=managed \
    --filter="metadata.name ~ ^e2e-test- \
    AND metadata.creationTimestamp < ${LIMIT_DATE}" \
  --format="value(metadata.name)")

for service in $CLOUD_RUN_SERVICES ; do
  echo "Deleting Cloud Run service: ${service}"
  gcloud run services delete "${service}" \
    --platform=managed \
    --quiet
done
echo "Done cleaning up Cloud Run services"

### Cloud Functions ###
GOOGLE_FUNCTIONS=$(gcloud functions list \
    --filter="name ~ e2e-test- \
    AND updateTime < ${LIMIT_DATE}" \
  --format="value(name)")

for gf in $GOOGLE_FUNCTIONS ; do
  echo "Deleting Google Function: ${gf}"
  gcloud functions delete ${gf} \
    --quiet
done
echo "Done cleaning up Cloud Functions"

### App Engines ###
APP_ENGINES=$(gcloud app services list \
--filter="SERVICE ~ ^e2e-test-" \
--format="value(SERVICE)")
for ap in ${APP_ENGINES} ; do
  echo "Deleting App Engine: ${ap}"
  gcloud app services delete "${ap}" \
    --quiet
done
echo "Done cleaning up App Engines"

### Firewall Rules ###
# Clean up firewall rules that point to deleted VMs.

FIREWALL_RULES=$(gcloud compute firewall-rules list \
    --filter="targetTags:(gke-e2e-cloud-run) \
    AND creationTimestamp < ${LIMIT_DATE}" \
    --format="value(name)")

for rule in $FIREWALL_RULES ; do
  echo "Deleting Firewall rule: ${rule}"
  gcloud compute firewall-rules delete ${rule} \
    --quiet
done
echo "Done cleaning up Firewall rules"

### Target Pools ###
# Clean up target pools that are unused: no forwarding rules point to these.
# Source: https://gist.github.com/prasvats/b2a4e33ad12b40191dd9d7e222d1abde

TARGET_POOLS=$(gcloud compute target-pools list \
    --regions="${REGION}" \
    --filter="creationTimestamp < ${LIMIT_DATE}" \
    --format='value(name)')

for targetpool in $TARGET_POOLS; do
  echo "Query Forwarding Rule for target pool ${targetpool}"
  forwardingitem=$(gcloud compute forwarding-rules list \
    --filter=TARGET="https://www.googleapis.com/compute/v1/projects/$PROJECT/regions/$REGION/targetPools/$targetpool" \
    --format='value(name)')
    if [[ -z "$forwardingitem" ]]; then
      echo "Deleting unused target pool ${targetpool}"
      gcloud compute target-pools delete "${targetpool}" \
        --region="${REGION}" \
        --quiet
    fi
done
echo "Done cleaning up target pools without forwarding rules"

### Forwarding Rules ###
# Clean up forwarding rules that point to deleted VMs.
# Must run AFTER target pool cleanup so that we don't have any orphaned target pools.

TARGET_POOLS=$(gcloud compute target-pools list \
    --regions="${REGION}" \
    --filter="instances:(gke-e2e-cloud-run) \
    AND creationTimestamp < ${LIMIT_DATE}" \
    --format='value(name)')

for targetpool in $TARGET_POOLS; do
  echo "Detected cloud run target pool ${targetpool}, querying forwarding rule"
  forwardingitem=$(gcloud compute forwarding-rules list \
    --filter=TARGET="https://www.googleapis.com/compute/v1/projects/$PROJECT/regions/$REGION/targetPools/$targetpool" \
    --format='value(name)')
  echo "Deleting forwarding rule ${forwardingitem}"
  gcloud compute forwarding-rules delete "${forwardingitem}" \
    --region="${REGION}" \
    --quiet
done
echo "Done cleaning up forwarding rules"

### Endpoints Services ###
ENDPOINTS_SERVICES=$(gcloud endpoints services list \
    --filter="serviceName ~ ^e2e-test-" \
  --format="value(serviceName)")

for service in $ENDPOINTS_SERVICES ; do
  echo "Checking if Endpoints Service is old enough to be deleted: ${service}"

  # The endpoints API does not expose creation date, so infer it from the config id.
  CONFIG_ID=$(gcloud endpoints configs list \
      --service="${service}" \
      --limit=1 \
    --format="value(id)")

  if [ -z "${CONFIG_ID}" ]
  then
    echo "Cannot determine config id, this is probably a failed rollout"
    echo "Cleaning up service"
    gcloud endpoints services delete ${service} \
      --quiet \
      --async
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

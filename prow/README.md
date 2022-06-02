## Running CI Scripts Manually

This directory contains entrypoints for our Continuous Integration system, [Prow](https://github.com/kubernetes/test-infra/tree/master/prow).

The scripts in this directory can also be run manually on your workstation,
but might need minor changes.
This document explains the changes required for each test.

### Prerequisites for all tests

All the tests will only run after these prerequisites are met.

#### IAM Permissions

Most of the tests are hard-coded to only work on the `cloudesf-testing` project.
Therefore, you will need the `Editor` role in `cloudesf-testing`.

Running the tests on a separate GCP project is not supported yet.

#### Build the Images

Assuming you are in the root of the repo, run:

```shell script
./prow/gcpproxy-build.sh
```

This script will:

1) Build all the binaries locally and place them in the [/bin](../bin) directory.
2) Build the docker images using [Google Cloud Build](https://cloud.google.com/cloud-build/).
3) Push the docker images to [Google Container Registry](https://cloud.google.com/container-registry/) in `cloudesf-testing`.

### Running the tests

The following tests can be run locally with minor changes.

### HTTP Bookstore on Google Cloud Run

You will need the `Cloud Run Admin` role for your user to run this script.
**Note this is not part of the default `Editor` role.**

You need install `jq` by
```shell script
sudo apt-get install jq
```

No other changes are needed. Run the script from the root of the repo:

```shell script
./prow/e2e-cloud-run-http-bookstore.sh
```

If you comment out the `tearDown` function in [cloud-run/deploy.sh](../tests/e2e/scripts/cloud-run/deploy.sh),
please make sure to manually clean-up the resources.

### HTTP Bookstore on Google Kubernetes Engine

Create a kubernetes cluster on [GKE](https://cloud.google.com/kubernetes-engine/).
Connect to the cluster in your shell:

```shell script
gcloud container clusters get-credentials ${CLUSTER_NAME} --zone ${ZONE} --project cloudesf-testing
```

Then run the script, no changes are needed:

```shell script
./prow/e2e-tight_http_bookstore_managed_long_run.sh
```

Please make sure to manually delete your entire cluster when you are done.
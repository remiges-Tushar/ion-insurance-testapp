# GCP deployment (Cloud Run, scale-to-zero)

Deploys the whole insurance app to Cloud Run. Every app/frontend service scales
to zero when idle and cold-starts on the next request — nothing runs
continuously except Cloud SQL (Postgres) and Memorystore (Redis), which have
no scale-to-zero tier on GCP.

This directory replaces `docker-compose.yml`'s Docker-network hostnames
(`http://bap:8083`, `redis:6379`, ...) with Cloud Run's per-service HTTPS URLs.

## How changes actually get applied now

State lives in a GCS bucket (`gs://remiges-ion-tfstate`), not a local file —
see `backend.tf`. **`terraform apply` is no longer meant to be run from a
laptop.** The supported flow is:

1. Open a PR touching `terraform/`. `.github/workflows/terraform-plan.yml`
   runs `terraform plan` automatically and posts the diff as a PR comment.
2. Review the plan comment, not just the code, before approving.
3. Merge to `main`. `.github/workflows/terraform-apply.yml` runs `plan` again
   (ungated), then pauses the `apply` job for a manual approval click — via
   the `production` GitHub Environment — before actually running
   `terraform apply` against the exact plan that was just reviewed.

Both workflows authenticate to GCP via Workload Identity Federation — no
service account keys are stored in GitHub. `plan` runs as the read-only
`tf-plan@remiges-ion.iam.gserviceaccount.com`; `apply` runs as
`tf-apply@remiges-ion.iam.gserviceaccount.com`, which GCP will only issue a
credential for to a workflow job that declares `environment: production` —
the approval gate is enforced at the credential level, not just in YAML.

The steps below (1–4) describe how this infrastructure was originally
bootstrapped by hand, before the pipeline existed — kept for reference and
for anyone standing up a fresh copy of this stack in a new project, but day
to day, changes should go through a PR as described above.

## Prerequisites

- `gcloud` CLI, authenticated (`gcloud auth login`), with a GCP project that has billing enabled.
- **Application Default Credentials set up too**: `gcloud auth login` alone is not enough — Terraform's `google` provider reads a separate credential store. Also run `gcloud auth application-default login` and `gcloud auth application-default set-quota-project <project-id>`.
- `terraform` >= 1.5.
- `docker`, for building images.
- The wiring step (see below) shells out to `gcloud run services update` during `terraform apply` — make sure `gcloud` is on `PATH` and already authenticated against the right project (`gcloud config set project <id>`).

## 1. Bootstrap: create the Artifact Registry repo and empty secret containers

Images have to exist before Cloud Run services can reference them, and every
secret Cloud Run reads has to have at least one version before a Cloud Run
revision referencing it can deploy — so create both up front, before anything
else:

Create `terraform/terraform.tfvars` (not committed) with just your project ID:

```hcl
project_id = "<your-project>"
```

```sh
cd terraform
terraform init
terraform apply -var-file=terraform.tfvars \
  -target=google_artifact_registry_repository.repo \
  -target=google_secret_manager_secret.this
```

No app secrets go in `terraform.tfvars` — Terraform only ever generates the
Postgres password itself; everything else is an empty Secret Manager
container at this point.

## 2. Seed the secrets Terraform didn't generate

```sh
cd ..
PROJECT_ID=<your-project> ./scripts/seed-secrets.sh
```

This pushes `bpp-jwt-secret`, `doku-client-id`, `doku-secret-key`, and the
onix signing keys into Secret Manager directly via `gcloud secrets versions
add` — the values never pass through a Terraform-visible file. Defaults match
today's `docker-compose.yml` / `config/insurance-*.yaml` sandbox values;
override via env vars (see the script) for real credentials.

## 3. Build and push all images

```sh
PROJECT_ID=<your-project> REGION=asia-southeast2 ./scripts/build-and-push.sh
```

## 4. Full apply

```sh
cd terraform
terraform apply -var-file=terraform.tfvars
```

This creates Cloud SQL, Memorystore, the VPC, all 8 Cloud Run services, both
migration jobs, and — as the last step — patches the handful of env vars that
reference other Cloud Run services' URLs (see `cloud-run-wiring.tf`; several
of these services reference each other in both directions, which Terraform's
dependency graph can't express directly, so that one file breaks the cycle
with a post-apply `gcloud run services update` pass).

## 5. Run migrations (once per schema change, not automatic)

```sh
gcloud run jobs execute migrate-bap --region=asia-southeast2 --wait
gcloud run jobs execute migrate-bpp --region=asia-southeast2 --wait
```

Confirm both show "Succeeded". If a job hangs instead of completing, see the
troubleshooting note in `cloud-run-jobs.tf` — the Cloud SQL Auth Proxy sidecar
pattern needs a fallback in that case.

## 6. Seed the BPP catalog (manual, local, one-time)

```sh
BPP_URL="$(terraform output -raw bpp_url)" ../catalog-seed/publish-catalog.sh
```

## Open the app

```sh
terraform output bap_frontend_url
terraform output bpp_frontend_url
```

First request after idle will be slower (cold start); subsequent requests are fast until it scales back to zero.

## Updating a service after code changes

```sh
../scripts/build-and-push.sh <new-tag>
gcloud run deploy bap --image=$(terraform output -raw artifact_registry_repo)/bap-application:<new-tag> --region=asia-southeast2
```

(`gcloud run deploy` here, not `terraform apply` — every service has
`lifecycle { ignore_changes = [image] }` specifically so day-to-day deploys
don't require a Terraform run.)

## Behavior changes from local docker-compose

- `DOKU_CALLBACK_URL` no longer points at a local dev tunnel (`lhr.life`) — it's set to this deployment's own `bap-frontend` Cloud Run URL + `/ion-webhook/doku`.
- `schema-server` and `catalog-seed/publish-catalog.sh` are intentionally not deployed — the former isn't referenced anywhere in the request path, the latter is a one-time local script (step 6 above).
- Double-check `doku_ion_bank_account_id_on_ion_service` vs `doku_ion_bank_account_id_on_bpp` in `variables.tf` — the original `docker-compose.yml` sets different values for these on `ion-service` vs `bpp`, which looked like it might be a mismatch; preserved as-is rather than silently "corrected."

## Teardown

```sh
terraform destroy -var-file=terraform.tfvars
```

`sql_deletion_protection` defaults to `false` for this reason — flip it to `true` once this stops being a throwaway deployment.

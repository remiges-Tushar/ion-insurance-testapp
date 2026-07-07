# Remote state — shared source of truth with native locking, so two people
# (or a person and CI) can never corrupt state by applying at the same time.
# See terraform/README.md for the one-time bootstrap that created this bucket.

terraform {
  backend "gcs" {
    bucket = "remiges-ion-tfstate"
    prefix = "insurance-app"
  }
}

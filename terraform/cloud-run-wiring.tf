# This app's services form a genuinely cyclic dependency graph: bap <-> onix-bap,
# bpp <-> onix-bpp, onix-bap <-> onix-bpp, bap <-> bap-frontend, and
# bap-frontend <-> ion all reference each other's URL. Terraform's resource
# graph can't express a cycle, so each service above is created with those
# specific cross-referencing env vars left blank, and this single step patches
# them all in afterwards via `gcloud run services update`, once every
# service's URL is actually known.
#
# `always_run` makes this re-apply on every `terraform apply` — otherwise a
# later apply that touches any of these services would reset its env back to
# the blank placeholder declared in its resource block, undoing this patch.
# Requires the gcloud CLI, authenticated, on the machine running `terraform
# apply`.

resource "null_resource" "wire_service_urls" {
  triggers = {
    always_run = timestamp()
  }

  provisioner "local-exec" {
    interpreter = ["bash", "-c"]
    command     = <<-EOT
      set -euo pipefail

      gcloud run services update ${google_cloud_run_v2_service.bap.name} \
        --project=${var.project_id} --region=${var.region} --quiet \
        --update-env-vars="BAP_ONIX_BAP_CALLER_URL=${google_cloud_run_v2_service.onix_bap.uri}/bap/caller,BAP_FRONTEND_URL=${google_cloud_run_v2_service.bap_frontend.uri}"

      gcloud run services update ${google_cloud_run_v2_service.bpp.name} \
        --project=${var.project_id} --region=${var.region} --quiet \
        --update-env-vars="BPP_ONIX_BPP_CALLER_URL=${google_cloud_run_v2_service.onix_bpp.uri}/bpp/caller,ION_SERVICE_URL=${google_cloud_run_v2_service.ion.uri}"

      gcloud run services update ${google_cloud_run_v2_service.bap_frontend.name} \
        --project=${var.project_id} --region=${var.region} --quiet \
        --update-env-vars="BAP_URL=${google_cloud_run_v2_service.bap.uri},ION_URL=${google_cloud_run_v2_service.ion.uri}"

      gcloud run services update ${google_cloud_run_v2_service.onix_bap.name} \
        --project=${var.project_id} --region=${var.region} --quiet \
        --update-env-vars="BAP_URL=${google_cloud_run_v2_service.bap.uri},ONIX_BPP_URL=${google_cloud_run_v2_service.onix_bpp.uri}"

      gcloud run services update ${google_cloud_run_v2_service.onix_bpp.name} \
        --project=${var.project_id} --region=${var.region} --quiet \
        --update-env-vars="BPP_URL=${google_cloud_run_v2_service.bpp.uri},ONIX_BAP_URL=${google_cloud_run_v2_service.onix_bap.uri}"

      gcloud run services update ${google_cloud_run_v2_service.ion.name} \
        --project=${var.project_id} --region=${var.region} --quiet \
        --update-env-vars="BPP_PAYMENT_URL=${google_cloud_run_v2_service.bpp.uri}/webhook/payment-received,BAP_FRONTEND_URL=${google_cloud_run_v2_service.bap_frontend.uri},DOKU_CALLBACK_URL=${google_cloud_run_v2_service.bap_frontend.uri}/ion-webhook/doku"
    EOT
  }

  depends_on = [
    google_cloud_run_v2_service.bap,
    google_cloud_run_v2_service.bpp,
    google_cloud_run_v2_service.bap_frontend,
    google_cloud_run_v2_service.bpp_frontend,
    google_cloud_run_v2_service.onix_bap,
    google_cloud_run_v2_service.onix_bpp,
    google_cloud_run_v2_service.ion,
    google_cloud_run_v2_service.cs,
  ]
}

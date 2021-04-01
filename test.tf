data "google_compute_ssl_certificate" "asos_com_tls_cert" {
  name = "asos-com-expires-june-01-2024"
}

data "google_compute_ssl_certificate" "pre_prod_asos_com_tls_cert" {
  name = "pre-prod-asos-com-expires-june-09-2024"
}

---
page_title: "tss_secret Data Source - terraform-provider-tss"
subcategory: ""
description: |-
  Fetches a secret from Delinea Secret Server. Omit `field` to get all fields as a map, or set `field` to get a single value.
---

# tss_secret (Data Source)

Fetches a secret from Delinea Secret Server. Omit `field` to get all fields as a map. Set `field` to get a single named value via the `value` attribute.

## Schema

### Required

- `id` (String) The ID of the secret to retrieve.

### Optional

- `field` (String) The specific field slug to extract from the secret. When set, the `value` attribute is populated.

### Read-Only

- `fields` (Map of String, Sensitive) All field slugs → values. Populated when `field` is **not** set; null otherwise.
- `value` (String, Sensitive) The value of the specified `field`. Populated when `field` **is** set; null otherwise.

## Example Usage

### All fields (omit `field`)

```hcl
data "tss_secret" "my_secret" {
  id = 3
}

output "username" {
  value     = data.tss_secret.my_secret.fields["username"]
  sensitive = true
}

output "password" {
  value     = data.tss_secret.my_secret.fields["password"]
  sensitive = true
}
```

### Single field (set `field`, use `value`)

```hcl
data "tss_secret" "my_username" {
  id    = 3
  field = "username"
}

output "username" {
  value     = data.tss_secret.my_username.value
  sensitive = true
}
```

> **Note:** The `fields` map keys are the field *slugs* as defined in your Secret Server template (e.g. `"username"`, `"password"`, `"url"`, `"notes"`). Use `terraform state show -show-sensitive data.tss_secret.<name>` to inspect available slugs for a given secret.
---
page_title: "tss_secret Ephemeral - terraform-provider-tss"
subcategory: ""
description: |-
  Fetches a secret from Delinea Secret Server ephemerally. Omit `field` to get all fields as a map, or set `field` to get a single value.
---

# tss_secret (Ephemeral Resource)

Fetches a secret from Delinea Secret Server ephemerally. Omit `field` to get all fields as a map. Set `field` to get a single named value via the `value` attribute. Values are never written to Terraform state.

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
ephemeral "tss_secret" "my_secret" {
  id = var.tss_secret_id
}

# Reference any field directly:
# ephemeral.tss_secret.my_secret.fields["username"]
# ephemeral.tss_secret.my_secret.fields["password"]
```

### Single field (set `field`, use `value`)

```hcl
ephemeral "tss_secret" "my_username" {
  id    = var.tss_secret_id
  field = "username"
}

# ephemeral.tss_secret.my_username.value
```

> **Note:** Ephemeral resource values are available only during the current Terraform operation and are never written to state. The secret is automatically re-fetched at renewal time (every 5 minutes by default).
>
> The `fields` map keys are the field *slugs* as defined in your Secret Server template (e.g. `"username"`, `"password"`, `"url"`, `"notes"`).

package delinea

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/DelineaXPM/tss-sdk-go/v2/server"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TSSSecretResource defines the resource implementation
type TSSSecretEphemeralResource struct {
	clientConfig *server.Configuration // Store the provider configuration
}

func (r *TSSSecretEphemeralResource) Metadata(ctx context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = "tss_secret"
}

// Define the model for your resource state
type TSSSecretEphemeralResourceModel struct {
	SecretID    types.String `tfsdk:"id"`
	Field       types.String `tfsdk:"field"`
	SecretValue types.String `tfsdk:"value"`
	Fields      types.Map    `tfsdk:"fields"`
}

// Define private data structure (optional)
type TSSSecretPrivateData struct {
	SecretID    string `json:"id"`
	Field       string `json:"field"`
	SecretValue string `json:"value"`
}

func (r *TSSSecretEphemeralResource) Schema(ctx context.Context, req ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a secret from Delinea Secret Server ephemerally. All fields are always returned in the 'fields' map. Optionally specify 'field' to also populate the 'value' attribute.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the secret to retrieve.",
			},
			"field": schema.StringAttribute{
				Optional:    true,
				Description: "The specific field to extract from the secret. When set, the 'value' attribute is populated.",
			},
			"value": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The value of the field specified by 'field'. Null when 'field' is not set.",
			},
			"fields": schema.MapAttribute{
				Computed:    true,
				Sensitive:   true,
				ElementType: types.StringType,
				Description: "A map of all field slugs to their values from the secret.",
			},
		},
	}
}

func (r *TSSSecretEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	// Create a model to hold the input configuration
	var data TSSSecretEphemeralResourceModel

	// Read the Terraform config data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.clientConfig == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot fetch secrets because the provider is not configured.")
		return
	}

	if data.SecretID.IsNull() {
		resp.Diagnostics.AddError("Missing Required Field", "id is required")
		return
	}

	// Initialize the Delinea API client
	client, err := server.New(*r.clientConfig)
	if err != nil {
		resp.Diagnostics.AddError("Client Creation Error", err.Error())
		return
	}

	// Convert SecretID to integer
	secretID, err := strconv.Atoi(data.SecretID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Secret ID", "Secret ID must be an integer")
		return
	}

	log.Printf("[DEBUG] getting secret with id %d", secretID)

	// Fetch the secret from the server using Delinea SDK
	secret, err := client.Secret(secretID)
	if err != nil {
		resp.Diagnostics.AddError("Secret Fetch Error", err.Error())
		return
	}

	if !data.Field.IsNull() && !data.Field.IsUnknown() && data.Field.ValueString() != "" {
		// 'field' is set — fetch just that one value, leave 'fields' map null
		log.Printf("[DEBUG] using '%s' field of secret with id %d", data.Field.ValueString(), secretID)

		// Extract the requested field value (assuming Field() method is available)
		fieldValue, ok := secret.Field(data.Field.ValueString())
		if !ok {
			resp.Diagnostics.AddError("Field Not Found", fmt.Sprintf("Field %s not found in the secret", data.Field.ValueString()))
			return
		}

		// Set the secret value in the result
		data.SecretValue = types.StringValue(fieldValue)
		data.Fields = types.MapNull(types.StringType)
	} else {
		// No specific field requested — build the full 'fields' map, leave 'value' null
		allFields := make(map[string]attr.Value, len(secret.Fields))
		for _, f := range secret.Fields {
			allFields[f.Slug] = types.StringValue(f.ItemValue)
		}
		fieldsMap, fieldsDiags := types.MapValue(types.StringType, allFields)
		resp.Diagnostics.Append(fieldsDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Fields = fieldsMap
		data.SecretValue = types.StringNull()
	}

	// Save the data into the ephemeral result state
	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)

	// Set a renewal time for the resource
	resp.RenewAt = time.Now().Add(5 * time.Minute)

	// Store private data for use during renewal
	privateData, _ := json.Marshal(TSSSecretPrivateData{
		SecretID:    data.SecretID.ValueString(),
		Field:       data.Field.ValueString(),
		SecretValue: data.SecretValue.ValueString(),
	})
	resp.Private.SetKey(ctx, "tss_secret_data", privateData)
}

func (r *TSSSecretEphemeralResource) Renew(ctx context.Context, req ephemeral.RenewRequest, resp *ephemeral.RenewResponse) {
	// Retrieve the private data that was stored during Open
	privateBytes, _ := req.Private.GetKey(ctx, "tss_secret_data")
	if privateBytes == nil {
		resp.Diagnostics.AddError("Missing Private Data", "Private data was not found for renewal.")
		return
	}

	// Unmarshal private data
	var privateData TSSSecretPrivateData
	if err := json.Unmarshal(privateBytes, &privateData); err != nil {
		resp.Diagnostics.AddError("Invalid Private Data", "Failed to unmarshal private data.")
		return
	}

	// Ensure that secret_id and field are available in the private data
	if privateData.SecretID == "" || privateData.Field == "" {
		resp.Diagnostics.AddError("Missing Private Data Fields", "Secret ID and field are required.")
		return
	}

	// Initialize your Delinea API client
	client, err := server.New(*r.clientConfig)
	if err != nil {
		resp.Diagnostics.AddError("Client Creation Error", err.Error())
		return
	}

	// Convert SecretID to integer
	secretID, err := strconv.Atoi(privateData.SecretID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Secret ID", "Secret ID must be an integer.")
		return
	}

	log.Printf("[DEBUG] getting secret with id %d to renew data", secretID)

	// Fetch the secret from the server
	secret, err := client.Secret(secretID)
	if err != nil {
		resp.Diagnostics.AddError("Secret Fetch Error", err.Error())
		return
	}

	log.Printf("[DEBUG] using '%s' field of secret with id %d to renew data", privateData.Field, secretID)

	// Extract the requested field value
	fieldValue, ok := secret.Field(privateData.Field)
	if !ok {
		resp.Diagnostics.AddError("Field Not Found", fmt.Sprintf("Field %s not found in the secret", privateData.Field))
		return
	}

	// Update the private data with the new secret value
	privateData.SecretValue = fieldValue

	// Store the updated private data for the next renewal
	privateDataBytes, _ := json.Marshal(privateData)
	resp.Private.SetKey(ctx, "tss_secret_data", privateDataBytes)

	// Set the renewal time (e.g., 5 minutes from now)
	resp.RenewAt = time.Now().Add(5 * time.Minute)
}

func (r *TSSSecretEphemeralResource) Close(ctx context.Context, req ephemeral.CloseRequest, resp *ephemeral.CloseResponse) {

}

func (r *TSSSecretEphemeralResource) Configure(ctx context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	log.Printf("DEBUG: ProviderData received in Configure")
	client, ok := req.ProviderData.(*server.Configuration)
	if !ok {
		resp.Diagnostics.AddError("Invalid Provider Data", "Expected provider data of type *server.Configuration")
		return
	}

	log.Printf("DEBUG: Successfully retrieved provider configuration")

	r.clientConfig = client
}

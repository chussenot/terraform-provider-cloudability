package cloudability

import (
	"strconv"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/skyscrapr/cloudability-sdk-go/cloudability"
)

func resourceAWSCredential() *schema.Resource {
	return &schema.Resource{
		Create: resourceAWSCredentialCreate,
		Read: resourceAWSCredentialRead,
		Delete: resourceAWSCredentialDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"vendor_account_name": {
				Type: schema.TypeString,
				Required: true,
				Description: "The name given to your AWS account",
			},
			"vendor_account_id": {
				Type: schema.TypeString,
				Required: true,
				Description: "12 digit string corresponding to your AWS account ID",
			},
			"vendor_key": {
				Type: schema.TypeString,
				Computed: true,
				Description: "'aws'",
			},
			"verification": {
				Type: schema.TypeSet,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource {
					Schema: map[string]*schema.Schema {
						"state": &schema.Schema {
							Type: schema.TypeString,
							Required: true,
							Description: "Examples: unverified, verified, error",
						},
						"last_verification_attempted_at": &schema.Schema {
							Type: schema.TypeString,
							Required: true,
							Description: "Date timestamp, example: 1970-01-01T00:00:00.000Z",
						},
						"message": &schema.Schema {
							Type: schema.TypeString,
							Required: true,
							Description: "Error message for credentials in error state",
						},
					},				
				},
				Description: "Object containing details of verification state",
			},
			"authorization": {
				Type: schema.TypeSet,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource {
					Schema: map[string]*schema.Schema {
						"type": &schema.Schema {
							Type: schema.TypeString,
							Required: true,
							Description: "'aws_role' or 'aws_user'",
						},
						"role_name": &schema.Schema {
							Type: schema.TypeString,
							Required: true,
							Description: "currently hardcoded to 'CloudabilityRole'",
						},
						"external_id": &schema.Schema {
							Type: schema.TypeString,
							Computed: true,
							Description: "The external ID used to prevent confused deputies. Generated by Cloudability",
						},
					},				
				},
				Description: "Object contain vendor specific authorization details",
			},
			"parent_account_id": {
				Type: schema.TypeString,
				Computed: true,
				Description: "12 digit string representing parent's account ID (if current cred is a linked account)",
			},
			"created_at": {
				Type: schema.TypeString,
				Computed: true,
				Description: "Date timestamp corresponding to cloudability credential creation time",
			},
		},
	}
}

func resourceAWSCredentialCreate(d *schema.ResourceData, meta interface{}) error {
	// Recipe - https://developers.cloudability.com/docs/vendor-credentials-end-point#section-recipe-for-adding-new-linked-account-credentials-aws
	// - Run verification on the master payer account (to ensure the account is in the list) Need to check if this blocks?
	// - Create the credential for the linked account
	// - Get the credential with the authorization and return it
	vendorKey := d.Get("vendor_key").(string)
	accountId := d.Get("vendor_account_id").(string)
	parentAccountId := d.Get("parent_account_id").(string)

	client := meta.(*cloudability.CloudabilityClient)	
	client.Vendors.VerifyCredential(vendorKey, parentAccountId)
	_, err := client.Vendors.NewCredential(vendorKey, accountId, "aws_role")
	if err != nil {
		return err
	}
	// probably need to loop until credential authorization is populated with timeout. I doubt it's done immediately
	return resourceAWSCredentialRead(d, meta)
}

func resourceAWSCredentialRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudability.CloudabilityClient)
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}
	credential, err := client.Vendors.GetCredential("aws", id)
	if err != nil {
		return err
	}

	if credential != nil {
		d.Set("vendor_account_name", credential.VendorAccountName)
		d.Set("vendor_account_id", credential.VendorAccountId)
		d.Set("vendor_key", credential.VendorKey)
		d.Set("verification", flattenVerification(credential.Verification))
		d.Set("authorization", flattenAuthorization(credential.Authorization))
		d.Set("parent_account_id", credential.ParentAccountId)
		d.Set("created_at", credential.CreatedAt)
		d.SetId(credential.Id)
	} else {
		// AWSCredential not found. Remove from state
		d.SetId("")
	}
	return nil
}

func resourceAWSCredentialDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudability.CloudabilityClient)
	id := d.Id()
	vendor := d.Get("vendor_key").(string)
	err := client.Vendors.DeleteCredential(vendor, id)
	return err
}

func flattenVerification(in cloudability.Verification) []map[string]interface{} {
	var out = make([]map[string]interface{}, 1, 1)
	m := make(map[string]interface{})
	m["state"] = in.State
	m["last_verification_attempted_at"] = in.LastVerificationAttemptedAt
	m["message"] = in.Message
	out[0]= m
	return out
}

func flattenAuthorization(in cloudability.Authorization) []map[string]interface{} {
	var out = make([]map[string]interface{}, 1, 1)
	m := make(map[string]interface{})
	m["type"] = in.Type
	m["role_name"] = in.RoleName
	m["external_id"] = in.ExternalId
	out[0]= m
	return out
}
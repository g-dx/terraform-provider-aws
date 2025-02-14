package route53resolver

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53resolver"
	"github.com/hashicorp/aws-sdk-go-base/v2/awsv1shim/v2/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/id"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @SDKResource("aws_route53_resolver_firewall_rule_group_association", name="Rule Group Association")
// @Tags(identifierAttribute="arn")
func ResourceFirewallRuleGroupAssociation() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceFirewallRuleGroupAssociationCreate,
		ReadWithoutTimeout:   resourceFirewallRuleGroupAssociationRead,
		UpdateWithoutTimeout: resourceFirewallRuleGroupAssociationUpdate,
		DeleteWithoutTimeout: resourceFirewallRuleGroupAssociationDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"firewall_rule_group_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"mutation_protection": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(route53resolver.MutationProtectionStatus_Values(), false),
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validResolverName,
			},
			"priority": {
				Type:     schema.TypeInt,
				Required: true,
			},
			names.AttrTags:    tftags.TagsSchema(),
			names.AttrTagsAll: tftags.TagsSchemaComputed(),
			"vpc_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},

		CustomizeDiff: verify.SetTagsDiff,
	}
}

func resourceFirewallRuleGroupAssociationCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).Route53ResolverConn(ctx)

	name := d.Get("name").(string)
	input := &route53resolver.AssociateFirewallRuleGroupInput{
		CreatorRequestId:    aws.String(id.PrefixedUniqueId("tf-r53-rslvr-frgassoc-")),
		FirewallRuleGroupId: aws.String(d.Get("firewall_rule_group_id").(string)),
		Name:                aws.String(name),
		Priority:            aws.Int64(int64(d.Get("priority").(int))),
		Tags:                getTagsIn(ctx),
		VpcId:               aws.String(d.Get("vpc_id").(string)),
	}

	if v, ok := d.GetOk("mutation_protection"); ok {
		input.MutationProtection = aws.String(v.(string))
	}

	output, err := conn.AssociateFirewallRuleGroupWithContext(ctx, input)

	if err != nil {
		return diag.Errorf("creating Route53 Resolver Firewall Rule Group Association (%s): %s", name, err)
	}

	d.SetId(aws.StringValue(output.FirewallRuleGroupAssociation.Id))

	if _, err := waitFirewallRuleGroupAssociationCreated(ctx, conn, d.Id()); err != nil {
		return diag.Errorf("waiting for Route53 Resolver Firewall Rule Group Association (%s) create: %s", d.Id(), err)
	}

	return resourceFirewallRuleGroupAssociationRead(ctx, d, meta)
}

func resourceFirewallRuleGroupAssociationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).Route53ResolverConn(ctx)

	ruleGroupAssociation, err := FindFirewallRuleGroupAssociationByID(ctx, conn, d.Id())

	if !d.IsNewResource() && tfresource.NotFound(err) {
		log.Printf("[WARN] Route53 Resolver Firewall Rule Group Association (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return diag.Errorf("reading Route53 Resolver Firewall Rule Group Association (%s): %s", d.Id(), err)
	}

	arn := aws.StringValue(ruleGroupAssociation.Arn)
	d.Set("arn", arn)
	d.Set("name", ruleGroupAssociation.Name)
	d.Set("firewall_rule_group_id", ruleGroupAssociation.FirewallRuleGroupId)
	d.Set("mutation_protection", ruleGroupAssociation.MutationProtection)
	d.Set("priority", ruleGroupAssociation.Priority)
	d.Set("vpc_id", ruleGroupAssociation.VpcId)

	return nil
}

func resourceFirewallRuleGroupAssociationUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).Route53ResolverConn(ctx)

	if d.HasChanges("name", "mutation_protection", "priority") {
		input := &route53resolver.UpdateFirewallRuleGroupAssociationInput{
			FirewallRuleGroupAssociationId: aws.String(d.Id()),
			Name:                           aws.String(d.Get("name").(string)),
			Priority:                       aws.Int64(int64(d.Get("priority").(int))),
		}

		if v, ok := d.GetOk("mutation_protection"); ok {
			input.MutationProtection = aws.String(v.(string))
		}

		_, err := conn.UpdateFirewallRuleGroupAssociationWithContext(ctx, input)

		if err != nil {
			return diag.Errorf("updating Route53 Resolver Firewall Rule Group Association (%s): %s", d.Id(), err)
		}

		if _, err := waitFirewallRuleGroupAssociationUpdated(ctx, conn, d.Id()); err != nil {
			return diag.Errorf("waiting for Route53 Resolver Firewall Rule Group Association (%s) update: %s", d.Id(), err)
		}
	}

	return resourceFirewallRuleGroupAssociationRead(ctx, d, meta)
}

func resourceFirewallRuleGroupAssociationDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).Route53ResolverConn(ctx)

	log.Printf("[DEBUG] Deleting Route53 Resolver Firewall Rule Group Association: %s", d.Id())
	_, err := conn.DisassociateFirewallRuleGroupWithContext(ctx, &route53resolver.DisassociateFirewallRuleGroupInput{
		FirewallRuleGroupAssociationId: aws.String(d.Id()),
	})

	if tfawserr.ErrCodeEquals(err, route53resolver.ErrCodeResourceNotFoundException) {
		return nil
	}

	if err != nil {
		return diag.Errorf("deleting Route53 Resolver Firewall Rule Group Association (%s): %s", d.Id(), err)
	}

	if _, err := waitFirewallRuleGroupAssociationDeleted(ctx, conn, d.Id()); err != nil {
		return diag.Errorf("waiting for Route53 Resolver Firewall Rule Group Association (%s) delete: %s", d.Id(), err)
	}

	return nil
}

func FindFirewallRuleGroupAssociationByID(ctx context.Context, conn *route53resolver.Route53Resolver, id string) (*route53resolver.FirewallRuleGroupAssociation, error) {
	input := &route53resolver.GetFirewallRuleGroupAssociationInput{
		FirewallRuleGroupAssociationId: aws.String(id),
	}

	output, err := conn.GetFirewallRuleGroupAssociationWithContext(ctx, input)

	if tfawserr.ErrCodeEquals(err, route53resolver.ErrCodeResourceNotFoundException) {
		return nil, &retry.NotFoundError{
			LastError:   err,
			LastRequest: input,
		}
	}

	if err != nil {
		return nil, err
	}

	if output == nil || output.FirewallRuleGroupAssociation == nil {
		return nil, tfresource.NewEmptyResultError(input)
	}

	return output.FirewallRuleGroupAssociation, nil
}

func statusFirewallRuleGroupAssociation(ctx context.Context, conn *route53resolver.Route53Resolver, id string) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := FindFirewallRuleGroupAssociationByID(ctx, conn, id)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return output, aws.StringValue(output.Status), nil
	}
}

const (
	firewallRuleGroupAssociationCreatedTimeout = 5 * time.Minute
	firewallRuleGroupAssociationUpdatedTimeout = 5 * time.Minute
	firewallRuleGroupAssociationDeletedTimeout = 5 * time.Minute
)

func waitFirewallRuleGroupAssociationCreated(ctx context.Context, conn *route53resolver.Route53Resolver, id string) (*route53resolver.FirewallRuleGroupAssociation, error) {
	stateConf := &retry.StateChangeConf{
		Pending: []string{route53resolver.FirewallRuleGroupAssociationStatusUpdating},
		Target:  []string{route53resolver.FirewallRuleGroupAssociationStatusComplete},
		Refresh: statusFirewallRuleGroupAssociation(ctx, conn, id),
		Timeout: firewallRuleGroupAssociationCreatedTimeout,
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*route53resolver.FirewallRuleGroupAssociation); ok {
		tfresource.SetLastError(err, errors.New(aws.StringValue(output.StatusMessage)))

		return output, err
	}

	return nil, err
}

func waitFirewallRuleGroupAssociationUpdated(ctx context.Context, conn *route53resolver.Route53Resolver, id string) (*route53resolver.FirewallRuleGroupAssociation, error) {
	stateConf := &retry.StateChangeConf{
		Pending: []string{route53resolver.FirewallRuleGroupAssociationStatusUpdating},
		Target:  []string{route53resolver.FirewallRuleGroupAssociationStatusComplete},
		Refresh: statusFirewallRuleGroupAssociation(ctx, conn, id),
		Timeout: firewallRuleGroupAssociationUpdatedTimeout,
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*route53resolver.FirewallRuleGroupAssociation); ok {
		tfresource.SetLastError(err, errors.New(aws.StringValue(output.StatusMessage)))

		return output, err
	}

	return nil, err
}

func waitFirewallRuleGroupAssociationDeleted(ctx context.Context, conn *route53resolver.Route53Resolver, id string) (*route53resolver.FirewallRuleGroupAssociation, error) {
	stateConf := &retry.StateChangeConf{
		Pending: []string{route53resolver.FirewallRuleGroupAssociationStatusDeleting},
		Target:  []string{},
		Refresh: statusFirewallRuleGroupAssociation(ctx, conn, id),
		Timeout: firewallRuleGroupAssociationDeletedTimeout,
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*route53resolver.FirewallRuleGroupAssociation); ok {
		tfresource.SetLastError(err, errors.New(aws.StringValue(output.StatusMessage)))

		return output, err
	}

	return nil, err
}

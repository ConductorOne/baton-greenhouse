package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-greenhouse/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
)

type roleBuilder struct {
	client *client.Client
}

func (o *roleBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return nil
}

// List returns all the roles from the database as resource objects.
// Roles include a RoleTrait because they are the 'shape' of a standard role.
func (o *roleBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	list, rl, next, err := o.client.ListRoles(ctx, pToken.Token)
	if err != nil {
		return nil, "", nil, fmt.Errorf("cannot list users, error: %w", err)
	}
	roles, err := Roles2Resources(list, parentResourceID)
	if err != nil {
		return nil, "", nil, err
	}

	var anno annotations.Annotations
	anno.WithRateLimiting(rl)

	return roles, next, anno, nil
}

func (o *roleBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func (o *roleBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func newRoleBuilder(c *client.Client) *roleBuilder {
	return &roleBuilder{
		client: c,
	}
}

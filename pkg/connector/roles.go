package connector

import (
	"context"
	"fmt"
	"github.com/conductorone/baton-greenhouse/pkg/models"

	"github.com/conductorone/baton-greenhouse/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	resourceSdk "github.com/conductorone/baton-sdk/pkg/types/resource"
)

type roleBuilder struct {
	Client *client.Client
}

func (b *roleBuilder) ResourceType(_ context.Context) *v2.ResourceType {
	return roleResourceType
}

func (b *roleBuilder) List(ctx context.Context, _ *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	var roleResources []*v2.Resource

	roles, rl, next, err := b.Client.ListRoles(ctx, pToken.Token)
	if err != nil {
		return nil, "", nil, fmt.Errorf("error while listing roles, error: %w", err)
	}

	for _, role := range roles {
		roleResource, err := parseIntoRoleResource(role)
		if err != nil {
			return nil, "", nil, err
		}

		roleResources = append(roleResources, roleResource)
	}

	var anno annotations.Annotations
	anno.WithRateLimiting(rl)

	return roleResources, next, anno, nil
}

func (b *roleBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func (b *roleBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func parseIntoRoleResource(role models.Role) (*v2.Resource, error) {
	profile := map[string]interface{}{
		"id":        role.ID,
		"name":      role.Name,
		"role_type": role.Type,
	}

	roleTraits := []resourceSdk.RoleTraitOption{
		resourceSdk.WithRoleProfile(profile),
	}

	ret, err := resourceSdk.NewRoleResource(
		role.Name,
		roleResourceType,
		role.ID,
		roleTraits,
	)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func newRoleBuilder(c *client.Client) *roleBuilder {
	return &roleBuilder{
		Client: c,
	}
}

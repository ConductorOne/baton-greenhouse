package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-greenhouse/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
)

type userBuilder struct {
	client *client.Client
}

func (o *userBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return userResourceType
}

// List returns all the users from the database as resource objects.
// Users include a UserTrait because they are the 'shape' of a standard user.
func (o *userBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	list, rl, next, err := o.client.ListUsers(ctx, pToken.Token)
	if err != nil {
		return nil, "", nil, fmt.Errorf("cannot list users, error: %w", err)
	}
	users, err := Users2Resources(list, parentResourceID)
	if err != nil {
		return nil, "", nil, err
	}

	var anno annotations.Annotations
	anno.WithRateLimiting(rl)

	return users, next, anno, nil
}

// Entitlements always returns an empty slice for users.
func (o *userBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

// Grants always returns an empty slice for users since they don't have any entitlements.
func (o *userBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func newUserBuilder(c *client.Client) *userBuilder {
	return &userBuilder{
		client: c,
	}
}

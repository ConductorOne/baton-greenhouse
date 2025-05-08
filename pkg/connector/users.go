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

func (o *userBuilder) ResourceType(_ context.Context) *v2.ResourceType {
	return userResourceType
}

func (o *userBuilder) List(ctx context.Context, _ *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	var userResources []*v2.Resource

	users, rl, next, err := o.client.ListUsers(ctx, pToken.Token)
	if err != nil {
		return nil, "", nil, fmt.Errorf("cannot list users, error: %w", err)
	}

	for _, user := range users {
		userResource, err := ParseIntoUserResource(user)
		if err != nil {
			return nil, "", nil, err
		}

		userResources = append(userResources, userResource)
	}

	var anno annotations.Annotations
	anno.WithRateLimiting(rl)

	return userResources, next, anno, nil
}

// Entitlements always returns an empty slice for users.
func (o *userBuilder) Entitlements(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

// Grants always returns an empty slice for users since they don't have any entitlements.
func (o *userBuilder) Grants(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func newUserBuilder(c *client.Client) *userBuilder {
	return &userBuilder{
		client: c,
	}
}

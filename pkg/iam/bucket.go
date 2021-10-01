package iam

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/genproto/googleapis/type/expr"

	iampb "google.golang.org/genproto/googleapis/iam/v1"
)

func (c *Client) UpdateBucketAccessControl(ctx context.Context, bucketName, member string, end time.Time) error {
	bucket := c.storageClient.Bucket(bucketName)
	policy, err := bucket.IAM().V3().Policy(ctx)
	if err != nil {
		return fmt.Errorf("getting policy for %v: %v", bucketName, err)
	}
	expression := getCondition(time.Now(), end)

	policy.Bindings = append(policy.Bindings, &iampb.Binding{
		Role:    "roles/storage.objectViewer",
		Members: []string{member},
		Condition: &expr.Expr{
			Title:      "Conditional access",
			Expression: expression,
		},
	})
	if err := bucket.IAM().V3().SetPolicy(ctx, policy); err != nil {
		return fmt.Errorf("setting policy for %v: %v", bucketName, err)
	}
	// NOTE: It may be necessary to retry this operation if IAM policies are
	// being modified concurrently. SetPolicy will return an error if the policy
	// was modified since it was retrieved.
	return nil
}

func getCondition(start, end time.Time) string {
	startString := start.String()
	endString := end.String()
	var expression string
	if len(startString) > 0 {
		expression = "request.time > timestamp('" + startString + "')"
	}
	if len(endString) > 0 {
		if len(startString) > 0 {
			expression = expression + " && request.time < timestamp('" + endString + "')"
		} else {
			expression = "request.time < timestamp('" + endString + "')"
		}

	}

	return expression
}

func (c *Client) RemoveMemberFromBucket(ctx context.Context, bucketName, bucketMember string) error {
	bucket := c.storageClient.Bucket(bucketName)
	policy, err := bucket.IAM().V3().Policy(ctx)
	if err != nil {
		return fmt.Errorf("getting policy for %v: %v", bucketName, err)
	}

	newBindings := make([]*iampb.Binding, 0)
	for _, binding := range policy.Bindings {
		if binding.Role == "roles/storage.objectViewer" {
			for _, member := range binding.Members {
				if !strings.HasSuffix(strings.ToLower(member), strings.ToLower(bucketMember)) {
					newBindings = append(newBindings, binding)
				}
			}
		} else {
			newBindings = append(newBindings, binding)
		}
	}

	policy.Bindings = newBindings

	if err := bucket.IAM().V3().SetPolicy(ctx, policy); err != nil {
		return fmt.Errorf("setting policy for %v: %v", bucketName, err)
	}
	// NOTE: It may be necessary to retry this operation if IAM policies are
	// being modified concurrently. SetPolicy will return an error if the policy
	// was modified since it was retrieved.
	return nil
}

func (c *Client) checkAccessInBucket(ctx context.Context, bucketName, bucketMember string) (bool, error) {
	bucket := c.storageClient.Bucket(bucketName)
	policy, err := bucket.IAM().V3().Policy(ctx)
	if err != nil {
		return false, fmt.Errorf("getting policy for %v: %v", bucketName, err)
	}
	for _, binding := range policy.Bindings {
		if binding.Role == "roles/storage.objectViewer" {
			for _, member := range binding.Members {
				if strings.HasSuffix(strings.ToLower(member), strings.ToLower(bucketMember)) {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

package metabase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type MetabaseSetting struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

type GroupMappingOperation string

const (
	GroupMappingOperationAdd    GroupMappingOperation = "add"
	GroupMappingOperationRemove GroupMappingOperation = "remove"
)

func (c *Client) UpdateGroupMapping(ctx context.Context, azureGroupID string, mbPermissionGroupID int, operation GroupMappingOperation) error {
	current, err := c.getGroupMappings(ctx)
	if err != nil {
		return err
	}

	var updated map[string][]int
	switch operation {
	case GroupMappingOperationAdd:
		updated = addGroupMapping(current, azureGroupID, mbPermissionGroupID)
	case GroupMappingOperationRemove:
		updated = removeGroupMapping(current, azureGroupID, mbPermissionGroupID)
	default:
		return errors.New("invalid group mapping operation")
	}

	payload := map[string]map[string][]int{"saml-group-mappings": updated}
	if err := c.request(ctx, http.MethodPut, "/setting", payload, nil); err != nil {
		return err
	}

	return nil
}

func addGroupMapping(mappings map[string][]int, azureGroupID string, mbPermissionGroupID int) map[string][]int {
	if pGroups, ok := mappings[azureGroupID]; ok {
		mappings[azureGroupID] = addGroup(pGroups, mbPermissionGroupID)
	} else {
		mappings[azureGroupID] = []int{mbPermissionGroupID}
	}

	return mappings
}

func addGroup(groups []int, group int) []int {
	for _, g := range groups {
		if g == group {
			return groups
		}
	}
	groups = append(groups, group)

	return groups
}

func removeGroupMapping(mappings map[string][]int, azureGroupID string, mbPermissionGroupID int) map[string][]int {
	if pGroups, ok := mappings[azureGroupID]; ok {
		mappings[azureGroupID] = removeGroup(pGroups, mbPermissionGroupID)
	}

	return mappings
}

func removeGroup(groups []int, group int) []int {
	for idx, g := range groups {
		if g == group {
			return append(groups[:idx], groups[idx+1:]...)
		}
	}

	return groups
}

func (c *Client) getGroupMappings(ctx context.Context) (map[string][]int, error) {
	settings := []*MetabaseSetting{}
	if err := c.request(ctx, http.MethodGet, "/setting", nil, &settings); err != nil {
		return nil, err
	}

	return getSAMLMappingFromSettings(settings)
}

func getSAMLMappingFromSettings(settings []*MetabaseSetting) (map[string][]int, error) {
	for _, s := range settings {
		if s.Key == "saml-group-mappings" {
			out := map[string][]int{}
			if err := json.Unmarshal(s.Value, &out); err != nil {
				return nil, fmt.Errorf("getSAMLMappingFromSettings: %w", err)
			}

			if out == nil {
				return map[string][]int{}, nil
			}
			return out, nil
		}
	}
	return nil, errors.New("saml group mappings not found in metabase settings")
}

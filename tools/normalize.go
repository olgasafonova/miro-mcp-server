package tools

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/olgasafonova/miro-mcp-server/miro/desirepath"
)

// normalizeArgs applies desire path normalizers to raw request arguments.
// Operates on the raw JSON map, then re-marshals into the typed Args struct.
// Falls back to the original args if any stage fails or no rules fire.
func normalizeArgs[Args any](h *HandlerRegistry, toolName string, req *mcp.CallToolRequest, args Args) Args {
	if h.desireLogger == nil || len(h.normalizers) == 0 {
		return args
	}

	argMap, ok := parseRawArgs(req)
	if !ok {
		return args
	}

	keyChanged := false
	if camel := findCamelNormalizer(h.normalizers); camel != nil {
		argMap, keyChanged = applyKeyRemapping(argMap, camel, toolName, h.desireLogger)
	}
	valChanged := applyValueNormalizers(argMap, h.normalizers, toolName, h.desireLogger)

	if !keyChanged && !valChanged {
		return args
	}
	return remarshalArgs(argMap, args, h.logger)
}

// parseRawArgs unmarshals the raw request arguments into a map. Returns the
// map and true on success, or nil and false if the request has no parameters,
// no raw arguments, or invalid JSON.
func parseRawArgs(req *mcp.CallToolRequest) (map[string]any, bool) {
	if req == nil || req.Params == nil {
		return nil, false
	}
	rawArgs := req.Params.Arguments
	if len(rawArgs) == 0 {
		return nil, false
	}
	var argMap map[string]any
	if err := json.Unmarshal(rawArgs, &argMap); err != nil {
		return nil, false
	}
	return argMap, true
}

// findCamelNormalizer returns the first CamelToSnakeNormalizer in the slice,
// or nil if none is registered.
func findCamelNormalizer(normalizers []desirepath.Normalizer) *desirepath.CamelToSnakeNormalizer {
	for _, n := range normalizers {
		if cn, ok := n.(*desirepath.CamelToSnakeNormalizer); ok {
			return cn
		}
	}
	return nil
}

// applyKeyRemapping rewrites camelCase keys to snake_case using the camel
// normalizer, logging each conversion. Returns the remapped map and whether
// any keys changed.
func applyKeyRemapping(argMap map[string]any, camel *desirepath.CamelToSnakeNormalizer, toolName string, logger *desirepath.Logger) (map[string]any, bool) {
	remapped := make(map[string]any, len(argMap))
	changed := false
	for key, val := range argMap {
		newKey, converted := camel.ConvertKey(key)
		if converted {
			changed = true
			logger.Log(desirepath.Event{
				Timestamp:    time.Now().UTC(),
				Tool:         toolName,
				Parameter:    key,
				Rule:         "camel_to_snake",
				RawValue:     key,
				NormalizedTo: newKey,
			})
		}
		remapped[newKey] = val
	}
	return remapped, changed
}

// applyValueNormalizers runs each non-camel normalizer over every value in
// argMap, chaining results through the same key. Mutates argMap in place.
// Returns true if any value changed.
func applyValueNormalizers(argMap map[string]any, normalizers []desirepath.Normalizer, toolName string, logger *desirepath.Logger) bool {
	changed := false
	for key, val := range argMap {
		for _, n := range normalizers {
			if _, ok := n.(*desirepath.CamelToSnakeNormalizer); ok {
				continue
			}
			newVal, result := n.Normalize(key, val)
			if !result.Changed {
				continue
			}
			argMap[key] = newVal
			val = newVal // chain normalizers
			changed = true
			logger.Log(desirepath.Event{
				Timestamp:    time.Now().UTC(),
				Tool:         toolName,
				Parameter:    key,
				Rule:         result.Rule,
				RawValue:     result.Original,
				NormalizedTo: result.New,
			})
		}
	}
	return changed
}

// remarshalArgs re-marshals the normalized map into the typed Args struct.
// Returns the fallback args on any marshaling error.
func remarshalArgs[Args any](argMap map[string]any, fallback Args, logger *slog.Logger) Args {
	normalized, err := json.Marshal(argMap)
	if err != nil {
		logger.Debug("Desire path: failed to marshal normalized args", "error", err)
		return fallback
	}
	var newArgs Args
	if err := json.Unmarshal(normalized, &newArgs); err != nil {
		logger.Debug("Desire path: failed to unmarshal into typed args", "error", err)
		return fallback
	}
	return newArgs
}

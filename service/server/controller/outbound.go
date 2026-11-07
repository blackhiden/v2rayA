package controller

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/v2rayA/v2rayA/common"
	"github.com/v2rayA/v2rayA/db/configure"
	"github.com/v2rayA/v2rayA/server/service"
)

func GetOutbounds(ctx *gin.Context) {
	outbounds := configure.GetOutbounds()
	common.ResponseSuccess(ctx, gin.H{
		"outbounds":          outbounds,
		"activeOutboundName": configure.GetActiveOutboundName(),
	})
}

func PostOutbound(ctx *gin.Context) {
	release, ok := beginMutation(ctx)
	if !ok {
		return
	}
	defer release()
	var data struct {
		Outbound string `json:"outbound"`
	}
	if err := ctx.ShouldBindJSON(&data); err != nil || data.Outbound == "" {
		common.ResponseError(ctx, badRequest("outbound", "request body must be a JSON object with a non-empty \"outbound\" string"))
		return
	}
	if err := configure.AddOutbound(data.Outbound); err != nil {
		common.ResponseError(ctx, logError(err))
		return
	}
	GetOutbounds(ctx)
}

func GetOutbound(ctx *gin.Context) {
	setting := configure.GetOutboundSetting(ctx.Query("outbound"))
	common.ResponseSuccess(ctx, gin.H{
		"setting": setting,
	})
}

func PutOutbound(ctx *gin.Context) {
	release, ok := beginMutation(ctx)
	if !ok {
		return
	}
	defer release()
	var data struct {
		Outbound string                    `json:"outbound"`
		Setting  configure.OutboundSetting `json:"setting"`
	}
	if err := ctx.ShouldBindJSON(&data); err != nil || data.Outbound == "" {
		common.ResponseError(ctx, badRequest("outbound", "request body must be a JSON object with a non-empty \"outbound\" string"))
		return
	}
	err := service.ApplyCoreConfig(func() func() error {
		previous := configure.GetOutboundSetting(data.Outbound)
		return func() error { return configure.SetOutboundSetting(data.Outbound, previous) }
	}, func() error {
		return configure.SetOutboundSetting(data.Outbound, data.Setting)
	})
	if err != nil {
		var failure *service.ApplyCoreConfigError
		if !errors.As(err, &failure) {
			common.ResponseError(ctx, logError(err))
			return
		}
		invalidConfigErr := fmt.Errorf("invalid config: %w", failure)
		common.ResponseError(ctx, common.Coded("INVALID_CONFIG", invalidConfigErr, map[string]interface{}{"detail": failure.Error()}))
		return
	}
	common.ResponseSuccess(ctx, nil)
}

func DeleteOutbound(ctx *gin.Context) {
	release, ok := beginMutation(ctx)
	if !ok {
		return
	}
	defer release()

	var data struct {
		Outbound string `json:"outbound"`
	}
	if err := ctx.ShouldBindJSON(&data); err != nil || data.Outbound == "" {
		common.ResponseError(ctx, badRequest("outbound", "request body must be a JSON object with a non-empty \"outbound\" string"))
		return
	}
	if data.Outbound == "proxy" {
		common.ResponseError(ctx, logError("outbound \"proxy\" cannot be deleted"))
		return
	}

	// Check if any custom inbound is bound to this outbound group
	boundInbounds := configure.GetCustomInboundsByOutbound(data.Outbound)
	if len(boundInbounds) > 0 {
		names := make([]string, len(boundInbounds))
		for i, ci := range boundInbounds {
			names[i] = fmt.Sprintf("%s (port %d, %s)", ci.Tag, ci.Port, ci.Protocol)
		}
		common.ResponseError(ctx, logError(fmt.Errorf(
			"cannot delete outbound group '%s': the following custom inbounds are bound to it:\n%s\nPlease unbind them first",
			data.Outbound,
			strings.Join(names, "\n"),
		)))
		return
	}

	if w := configure.GetConnectedServersByOutbound(data.Outbound); w != nil {
		if err := service.Disconnect(configure.NodeRef{Outbound: data.Outbound}, true); err != nil {
			common.ResponseError(ctx, logError(err))
			return
		}
	}
	if err := configure.RemoveOutbound(data.Outbound); err != nil {
		common.ResponseError(ctx, logError(err))
		return
	}
	GetOutbounds(ctx)
}

func PutOutboundConnections(ctx *gin.Context) {
	release, ok := beginMutation(ctx)
	if !ok {
		return
	}
	defer release()

	var data struct {
		Outbound string `json:"outbound"`
		Touches  []struct {
			ID        int    `json:"id"`
			TYPE      string `json:"_type"`
			TYPEAlias string `json:"type"`
			Sub       *int   `json:"sub"`
			Outbound  string `json:"outbound"`
		} `json:"touches"`
	}
	if err := ctx.ShouldBindJSON(&data); err != nil {
		common.ResponseError(ctx, badRequest("outbound connections", "request body must be {\"outbound\": string, \"touches\": [...]}"))
		return
	}

	normalizeTouchType := func(raw string) (configure.TouchType, bool) {
		normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(raw, "_", ""), "-", ""))
		switch normalized {
		case "server":
			return configure.ServerType, true
		case "subscriptionserver":
			return configure.SubscriptionServerType, true
		default:
			return "", false
		}
	}

	whiches := make([]configure.NodeRef, 0, len(data.Touches))
	for i, w := range data.Touches {
		rawType := w.TYPE
		if rawType == "" {
			rawType = w.TYPEAlias
		}
		typ, ok := normalizeTouchType(rawType)
		if !ok {
			common.ResponseError(ctx, logError(fmt.Errorf("touches[%d]._type %q is not valid; expected server or subscriptionServer", i, rawType)))
			return
		}
		if w.ID <= 0 {
			common.ResponseError(ctx, logError(fmt.Errorf("touches[%d].id %d must be positive", i, w.ID)))
			return
		}
		sub := 0
		if w.Sub != nil {
			sub = *w.Sub
		}
		if typ == configure.SubscriptionServerType && sub < 0 {
			common.ResponseError(ctx, logError(fmt.Errorf("touches[%d].sub %d must be non-negative", i, sub)))
			return
		}
		if typ == configure.ServerType {
			sub = 0
		}
		whiches = append(whiches, configure.NodeRef{
			TYPE:     typ,
			ID:       w.ID,
			Sub:      sub,
			Outbound: data.Outbound,
		})
	}

	if err := service.ReplaceOutboundConnections(data.Outbound, whiches); err != nil {
		common.ResponseError(ctx, logError(err))
		return
	}
	getTouch(ctx)
}

// PutOutboundSelection chooses the member a group routes through alone, or
// returns the group to balancing when `which` is null. The member must be
// connected in that group.
func PutOutboundSelection(ctx *gin.Context) {
	release, ok := beginMutation(ctx)
	if !ok {
		return
	}
	defer release()

	var data struct {
		Outbound string             `json:"outbound"`
		Which    *configure.NodeRef `json:"which"`
	}
	if err := ctx.ShouldBindJSON(&data); err != nil || data.Outbound == "" {
		common.ResponseError(ctx, badRequest("outbound selection", "request body must be {\"outbound\": string, \"which\": {...} | null}"))
		return
	}
	link := ""
	if data.Which != nil {
		data.Which.Outbound = data.Outbound
		if data.Which.TYPE == configure.ServerType {
			data.Which.Sub = 0
		}
		member := false
		if members := configure.GetConnectedServersByOutbound(data.Outbound); members != nil {
			for _, m := range members.Get() {
				if m.EqualTo(*data.Which) {
					member = true
					break
				}
			}
		}
		if !member {
			common.ResponseError(ctx, logError(fmt.Errorf("the node is not a member of outbound %q", data.Outbound)))
			return
		}
		sr, err := data.Which.LocateServerRaw()
		if err != nil {
			common.ResponseError(ctx, logError(err))
			return
		}
		link = sr.ServerObj.ExportToURL()
	}
	err := service.ApplyCoreConfig(func() func() error {
		previous := configure.GetOutboundSetting(data.Outbound)
		return func() error { return configure.SetOutboundSetting(data.Outbound, previous) }
	}, func() error {
		setting := configure.GetOutboundSetting(data.Outbound)
		setting.Selected = link
		return configure.SetOutboundSetting(data.Outbound, setting)
	})
	if err != nil {
		var failure *service.ApplyCoreConfigError
		if !errors.As(err, &failure) {
			common.ResponseError(ctx, logError(err))
			return
		}
		common.ResponseError(ctx, common.Coded("INVALID_CONFIG", fmt.Errorf("invalid config: %w", failure.UpdateErr), map[string]interface{}{"detail": failure.UpdateErr.Error()}))
		return
	}
	getTouch(ctx)
}

func SetActiveOutbound(ctx *gin.Context) {
	release, ok := beginMutation(ctx)
	if !ok {
		return
	}
	defer release()

	var data struct {
		ActiveOutboundName string `json:"activeOutboundName"`
	}
	if err := ctx.ShouldBindJSON(&data); err != nil {
		common.ResponseError(ctx, badRequest("active outbound", "request body must be a JSON object with a non-empty \"activeOutboundName\" string"))
		return
	}
	data.ActiveOutboundName = strings.TrimSpace(data.ActiveOutboundName)
	if data.ActiveOutboundName == "" {
		common.ResponseError(ctx, badRequest("active outbound", "\"activeOutboundName\" cannot be empty"))
		return
	}
	if !configure.IsOutbound(data.ActiveOutboundName) {
		common.ResponseError(ctx, logError(fmt.Errorf("outbound group %q does not exist", data.ActiveOutboundName)))
		return
	}

	previous := configure.GetActiveOutboundName()
	err := service.ApplyCoreConfig(func() func() error {
		return func() error { return configure.SetActiveOutboundName(previous) }
	}, func() error {
		return configure.SetActiveOutboundName(data.ActiveOutboundName)
	})
	if err != nil {
		var failure *service.ApplyCoreConfigError
		if !errors.As(err, &failure) {
			common.ResponseError(ctx, logError(err))
			return
		}
		common.ResponseError(ctx, common.Coded("INVALID_CONFIG", fmt.Errorf("invalid config: %w", failure.UpdateErr), map[string]interface{}{"detail": failure.UpdateErr.Error()}))
		return
	}

	common.ResponseSuccess(ctx, gin.H{"activeOutboundName": data.ActiveOutboundName})
}

// PostBulkRemoveByLatency removes servers from an outbound group based on latency threshold.
// Servers with latency >= threshold OR servers without latency data (errors/timeouts) are removed.
// At least one server must remain in the group after removal.
func PostBulkRemoveByLatency(ctx *gin.Context) {
	release, ok := beginMutation(ctx)
	if !ok {
		return
	}
	defer release()

	var data struct {
		Outbound       string `json:"outbound"`
		MaxLatencyMs   int    `json:"maxLatencyMs"`
		ServersToCheck []struct {
			ID        int    `json:"id"`
			TYPE      string `json:"_type"`
			TYPEAlias string `json:"type"`
			Sub       *int   `json:"sub"`
		} `json:"serversToCheck"`
	}
	if err := ctx.ShouldBindJSON(&data); err != nil {
		common.ResponseError(ctx, badRequest("bulk remove by latency", "request body must be valid JSON"))
		return
	}
	if data.Outbound == "" {
		common.ResponseError(ctx, badRequest("outbound", "outbound is required"))
		return
	}
	if data.MaxLatencyMs <= 0 {
		common.ResponseError(ctx, badRequest("maxLatencyMs", "maxLatencyMs must be positive"))
		return
	}
	if len(data.ServersToCheck) == 0 {
		common.ResponseError(ctx, badRequest("serversToCheck", "no servers to check"))
		return
	}

	connected := configure.GetConnectedServersByOutbound(data.Outbound)
	if connected == nil || connected.Len() == 0 {
		common.ResponseError(ctx, logError("no connected servers in this outbound"))
		return
	}

	normalizeTouchType := func(raw string) (configure.TouchType, bool) {
		normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(raw, "_", ""), "-", ""))
		switch normalized {
		case "server":
			return configure.ServerType, true
		case "subscriptionserver":
			return configure.SubscriptionServerType, true
		default:
			return "", false
		}
	}

	removeSet := make(map[configure.NodeRef]bool)
	for _, s := range data.ServersToCheck {
		rawType := s.TYPE
		if rawType == "" {
			rawType = s.TYPEAlias
		}
		typ, ok := normalizeTouchType(rawType)
		if !ok {
			continue
		}
		sub := 0
		if s.Sub != nil {
			sub = *s.Sub
		}
		if typ == configure.ServerType {
			sub = 0
		}
		which := configure.NodeRef{
			TYPE:     typ,
			ID:       s.ID,
			Sub:      sub,
			Outbound: data.Outbound,
		}
		removeSet[which] = true
	}

	var remaining []configure.NodeRef
	for _, w := range connected.Get() {
		target := configure.NodeRef{
			TYPE:     w.TYPE,
			ID:       w.ID,
			Sub:      w.Sub,
			Outbound: data.Outbound,
		}
		if !removeSet[target] {
			remaining = append(remaining, target)
		}
	}

	if len(remaining) == 0 {
		common.ResponseError(ctx, logError("cannot remove all servers from the group; at least one server must remain"))
		return
	}

	if err := service.ReplaceOutboundConnections(data.Outbound, remaining); err != nil {
		common.ResponseError(ctx, logError(err))
		return
	}

	common.ResponseSuccess(ctx, gin.H{
		"removedCount":   len(removeSet),
		"remainingCount": len(remaining),
		"outbound":       data.Outbound,
	})
}

